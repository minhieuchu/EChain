package network

import (
	"EChain/blockchain"
	"sync"

	"fmt"
	"io"
	"log"
	"net"
	"time"

	"golang.org/x/exp/slices"
)

var initialPeers = []string{"localhost:8333", "localhost:8334", "localhost:8335"}

const (
	VERSION_MSG   = "version"
	VERACK_MSG    = "verack"
	ADDR_MSG      = "addr"
	GETBLOCKS_MSG = "getblocks"
	INV_MSG       = "inv"
	GETDATA_MSG   = "getdata"
	BLOCKDATA_MSG = "blockdata"
)

const (
	FULLNODE = "fullnode"
	SPV      = "spv"
	MINER    = "miner"
)

const (
	protocol                       = "tcp"
	msgTypeLength                  = 12 // First 12 bytes of each byte slice exchanged between peers are reserved for message type
	MAX_BLOCKS_IN_TRANSIT_PER_PEER = 10
)

type P2PNode struct {
	NodeType            string
	Version             int
	NetworkAddress      string
	Blockchain          *blockchain.BlockChain
	connectedPeers      []string
	forwardedAddrList   []string
	getdataMessageCount int
}

// ======= Handle requests =======

func (node *P2PNode) handleVersionMsg(msg []byte) {
	var versionMsg VersionMessage
	genericDeserialize(msg, &versionMsg)

	if node.Version == versionMsg.Version {
		node.sendVerackMsg(versionMsg.AddrMe)
		if !slices.Contains(node.connectedPeers, versionMsg.AddrMe) {
			node.sendVersionMsg(versionMsg.AddrMe)
		}
	}
}

func (node *P2PNode) handleVerackMsg(msg []byte) {
	var verackMsg VerackMessage
	genericDeserialize(msg, &verackMsg)

	if slices.Contains(node.connectedPeers, verackMsg.AddrFrom) {
		return
	}
	node.connectedPeers = append(node.connectedPeers, verackMsg.AddrFrom)
	node.sendAddrMsg(verackMsg.AddrFrom)
	node.sendGetBlocksMsg(verackMsg.AddrFrom)
}

func (node *P2PNode) handleAddrMsg(msg []byte) {
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

func (node *P2PNode) handleGetblocksMsg(msg []byte) {
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

func (node *P2PNode) handleGetdataMsg(msg []byte) {
	var getdataMsg GetdataMessage
	genericDeserialize(msg, &getdataMsg)

	blockList := node.Blockchain.GetBlocksFromHashes(getdataMsg.HashList)
	node.sendBlockdataMessage(getdataMsg.AddrFrom, getdataMsg.Index, blockList)
}

func (node *P2PNode) handleInvMsg(msg []byte) {
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

func (node *P2PNode) handleBlockdataMsg(msg []byte) {
	var blockdataMsg BlockdataMessage
	genericDeserialize(msg, &blockdataMsg)
	for _, block := range blockdataMsg.BlockList {
		node.Blockchain.SetBlock(block)
	}
	if blockdataMsg.Index == node.getdataMessageCount-1 {
		node.Blockchain.SetLastHash(blockdataMsg.BlockList[len(blockdataMsg.BlockList)-1].Hash)
	}
}

func (node *P2PNode) handleConnection(conn net.Conn) {
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

func (node *P2PNode) sendGetdataMessage(toAddress string, getdataMsg *GetdataMessage) {
	fmt.Println("Send Getdata msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(GETDATA_MSG), serialize(getdataMsg)...)
	sendMessageBlocking(toAddress, sentData)
}

func (node *P2PNode) sendBlockdataMessage(toAddress string, msgIndex int, blockList []*blockchain.Block) {
	fmt.Println("Send Blockdata msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(BLOCKDATA_MSG), serialize(BlockdataMessage{msgIndex, blockList})...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendInvMessage(toAddress string, invMsg *InvMessage) {
	fmt.Println("Send Inv msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(INV_MSG), serialize(invMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendGetBlocksMsg(toAddress string) {
	fmt.Println("Send Getblocks msg from", node.NetworkAddress, "to", toAddress)
	lastBlockHash := node.Blockchain.LastHash
	getblocksMsg := GetblocksMessage{lastBlockHash, node.NetworkAddress}
	sentData := append(msgTypeToBytes(GETBLOCKS_MSG), serialize(getblocksMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendAddrMsg(toAddress string) {
	fmt.Println("Send Addr msg from", node.NetworkAddress, "to", toAddress)
	addrMsg := AddrMessage{node.NetworkAddress}
	sentData := append(msgTypeToBytes(ADDR_MSG), serialize(addrMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendVersionMsg(toAddress string) {
	fmt.Println("Send Version msg from", node.NetworkAddress, "to", toAddress)
	nBestHeight := node.Blockchain.GetHeight()
	versionMsg := VersionMessage{node.Version, toAddress, node.NetworkAddress, nBestHeight}
	sentData := append(msgTypeToBytes(VERSION_MSG), serialize(versionMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendVerackMsg(toAddress string) {
	fmt.Println("Send Verack msg from", node.NetworkAddress, "to", toAddress)
	verackMsg := VerackMessage{node.NetworkAddress}
	sentData := append(msgTypeToBytes(VERACK_MSG), serialize(verackMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) StartP2PNode() {
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

func NewBlockChainNode(nodeType, networkAddress, walletAddress string) *P2PNode {
	localBlockchain := blockchain.InitBlockChain(networkAddress, walletAddress)
	blockchainNode := P2PNode{
		NodeType:       nodeType,
		Version:        1,
		NetworkAddress: networkAddress,
		Blockchain:     localBlockchain,
	}
	return &blockchainNode
}
