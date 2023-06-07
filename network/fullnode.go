package network

import (
	"EChain/blockchain"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/exp/slices"
)

type FullNode struct {
	P2PNode
	Blockchain          *blockchain.BlockChain
	getdataMessageCount int
}

func NewFullNode(networkAddress, walletAddress string) *FullNode {
	localBlockchain := blockchain.InitBlockChain(networkAddress, walletAddress)
	p2pNode := P2PNode{
		Version:        1,
		NetworkAddress: networkAddress,
	}
	return &FullNode{
		P2PNode:    p2pNode,
		Blockchain: localBlockchain,
	}
}

func (node *FullNode) handleGetblocksMsg(msg []byte) {
	var getblocksMsg GetblocksMessage
	genericDeserialize(msg, &getblocksMsg)

	remoteLastBlockHash := getblocksMsg.TopBlockHash
	blockExisted, unmatchedBlocks := node.Blockchain.GetUnmatchedBlocks(remoteLastBlockHash)
	if blockExisted && len(unmatchedBlocks) > 0 {
		blockHashesToSend := [][]byte{}
		for i := len(unmatchedBlocks) - 1; i >= 0; i-- {
			blockHashesToSend = append(blockHashesToSend, unmatchedBlocks[i])
			if len(blockHashesToSend) >= 500 {
				break
			}
		}
		invMsg := InvMessage{blockHashesToSend}
		node.sendInvMessage(getblocksMsg.AddrFrom, &invMsg)
	} else if !blockExisted {
		node.sendGetBlocksMsg(getblocksMsg.AddrFrom)
	}
}

func (node *FullNode) handleGetdataMsg(msg []byte) {
	var getdataMsg GetdataMessage
	genericDeserialize(msg, &getdataMsg)

	blockList := node.Blockchain.GetBlocksFromHashes(getdataMsg.HashList)
	node.sendBlockdataMessage(getdataMsg.AddrFrom, getdataMsg.Index, blockList)
}

func (node *FullNode) handleInvMsg(msg []byte) {
	var invMsg InvMessage
	genericDeserialize(msg, &invMsg)

	getdataMsgList := []GetdataMessage{}
	for _, blockHash := range invMsg.HashList {
		lastIndex := len(getdataMsgList) - 1
		if len(getdataMsgList) == 0 || len(getdataMsgList[lastIndex].HashList) >= MAX_BLOCKS_IN_TRANSIT_PER_PEER {
			newHashList := [][]byte{blockHash}
			newGetDataMsg := GetdataMessage{lastIndex + 1, newHashList, node.NetworkAddress}
			getdataMsgList = append(getdataMsgList, newGetDataMsg)
		} else {
			getdataMsgList[lastIndex].HashList = append(getdataMsgList[lastIndex].HashList, blockHash)
		}
	}

	// Send getdata messages to peers
	var wg sync.WaitGroup
	var mutex sync.Mutex
	messageIndex := 0
	messageCount := len(getdataMsgList)
	node.getdataMessageCount = messageCount
	wg.Add(messageCount)

	for peerIndex, peerAddr := range node.connectedPeers {
		if peerIndex >= messageCount {
			break
		}
		go func(toAddress string) {
			for {
				mutex.Lock()
				if messageIndex >= messageCount {
					return
				}
				getdataMsg := getdataMsgList[messageIndex]
				messageIndex++
				mutex.Unlock()

				node.sendGetdataMessage(toAddress, &getdataMsg)
				wg.Done()
			}
		}(peerAddr)
	}
	wg.Wait()
	time.Sleep(3 * time.Second) // Wait for all blockdata messages to be processed
	utxoSet := node.Blockchain.UTXOSet()
	utxoSet.ReIndex()
	for _, peerAddr := range node.connectedPeers {
		node.sendGetBlocksMsg(peerAddr)
	}
}

func (node *FullNode) handleBlockdataMsg(msg []byte) {
	var blockdataMsg BlockdataMessage
	genericDeserialize(msg, &blockdataMsg)
	for _, block := range blockdataMsg.BlockList {
		node.Blockchain.SetBlock(block)
	}
	if blockdataMsg.Index == node.getdataMessageCount-1 {
		node.Blockchain.SetLastHash(blockdataMsg.BlockList[len(blockdataMsg.BlockList)-1].GetHash())
	}
}

func (node *FullNode) sendGetBlocksMsg(toAddress string) {
	fmt.Println("Send Getblocks msg from", node.NetworkAddress, "to", toAddress)
	lastBlockHash := node.Blockchain.LastHash
	getblocksMsg := GetblocksMessage{lastBlockHash, node.NetworkAddress}
	sentData := append(msgTypeToBytes(GETBLOCKS_MSG), serialize(getblocksMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *FullNode) sendVersionMsg(toAddress string) {
	fmt.Println("Send Version msg from", node.NetworkAddress, "to", toAddress)
	nBestHeight := node.Blockchain.GetHeight()
	versionMsg := VersionMessage{node.Version, toAddress, node.NetworkAddress, nBestHeight}
	sentData := append(msgTypeToBytes(VERSION_MSG), serialize(versionMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *FullNode) sendGetdataMessage(toAddress string, getdataMsg *GetdataMessage) {
	fmt.Println("Send Getdata msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(GETDATA_MSG), serialize(getdataMsg)...)
	sendMessageBlocking(toAddress, sentData)
}

func (node *FullNode) sendBlockdataMessage(toAddress string, msgIndex int, blockList []*blockchain.Block) {
	fmt.Println("Send Blockdata msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(BLOCKDATA_MSG), serialize(BlockdataMessage{msgIndex, blockList})...)
	sendMessage(toAddress, sentData)
}

func (node *FullNode) sendInvMessage(toAddress string, invMsg *InvMessage) {
	fmt.Println("Send Inv msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(INV_MSG), serialize(invMsg)...)
	sendMessage(toAddress, sentData)
}

// ======= Handle requests =======

func (node *FullNode) handleVersionMsg(msg []byte) {
	var versionMsg VersionMessage
	genericDeserialize(msg, &versionMsg)

	if node.Version == versionMsg.Version {
		node.sendVerackMsg(versionMsg.AddrMe)
		if !slices.Contains(node.connectedPeers, versionMsg.AddrMe) {
			node.sendVersionMsg(versionMsg.AddrMe)
		}
	}
}

func (node *FullNode) handleVerackMsg(msg []byte) {
	var verackMsg VerackMessage
	genericDeserialize(msg, &verackMsg)

	if slices.Contains(node.connectedPeers, verackMsg.AddrFrom) {
		return
	}
	node.connectedPeers = append(node.connectedPeers, verackMsg.AddrFrom)
	node.sendAddrMsg(verackMsg.AddrFrom)
	node.sendGetBlocksMsg(verackMsg.AddrFrom)
}

func (node *FullNode) handleAddrMsg(msg []byte) {
	var addrMsg AddrMessage
	genericDeserialize(msg, &addrMsg)

	if !slices.Contains(node.connectedPeers, addrMsg.Address) {
		node.sendVersionMsg(addrMsg.Address)
	}

	if !slices.Contains(node.forwardedAddrList, addrMsg.Address) {
		node.forwardedAddrList = append(node.forwardedAddrList, addrMsg.Address)
		for _, peerAddr := range node.connectedPeers {
			if peerAddr != addrMsg.Address {
				sentData := append(msgTypeToBytes(ADDR_MSG), msg...)
				sendMessage(peerAddr, sentData)
			}
		}
	}
}

func (node *FullNode) handleConnection(conn net.Conn) {
	data, err := io.ReadAll(conn)
	defer conn.Close()
	handleError(err)

	msgType := getMsgType(data)
	payload := data[msgTypeLength:]
	switch msgType {
	case VERSION_MSG:
		node.handleVersionMsg(payload)
	case VERACK_MSG:
		node.handleVerackMsg(payload)
	case ADDR_MSG:
		node.handleAddrMsg(payload)
	case GETBLOCKS_MSG:
		node.handleGetblocksMsg(payload)
	case INV_MSG:
		node.handleInvMsg(payload)
	case GETDATA_MSG:
		node.handleGetdataMsg(payload)
	case BLOCKDATA_MSG:
		node.handleBlockdataMsg(payload)
	default:
		fmt.Println("invalid message")
	}
}

// ======= Send messages =======

func (node *FullNode) sendAddrMsg(toAddress string) {
	fmt.Println("Send Addr msg from", node.NetworkAddress, "to", toAddress)
	addrMsg := AddrMessage{node.NetworkAddress}
	sentData := append(msgTypeToBytes(ADDR_MSG), serialize(addrMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *FullNode) sendVerackMsg(toAddress string) {
	fmt.Println("Send Verack msg from", node.NetworkAddress, "to", toAddress)
	verackMsg := VerackMessage{node.NetworkAddress}
	sentData := append(msgTypeToBytes(VERACK_MSG), serialize(verackMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *FullNode) StartP2PNode() {
	fmt.Println(" ===== Starting blockchain node at", node.NetworkAddress, "=====")
	ln, err := net.Listen(protocol, node.NetworkAddress)
	if err != nil {
		log.Fatal("can not start server at", node.NetworkAddress)
	}

	go func() {
		time.Sleep(2 * time.Second)
		for _, peerAddr := range initialPeers {
			if peerAddr != node.NetworkAddress {
				node.sendVersionMsg(peerAddr)
			}
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err.Error())
		}

		go node.handleConnection(conn)
	}
}
