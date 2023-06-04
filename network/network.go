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

var initialPeers = []string{"127.0.0.1:8333", "127.0.0.1:8334", "127.0.0.1:8335"}

const (
	VERSION_MSG   = "version"
	VERACK_MSG    = "verack"
	ADDR_MSG      = "addr"
	GETBLOCKS_MSG = "getblocks"
	INV_MSG       = "inv"
	GETDATA_MSG   = "getdata"
)

const (
	protocol                       = "tcp"
	msgTypeLength                  = 12 // First 12 bytes of each byte slice exchanged between peers are reserved for message type
	MAX_BLOCKS_IN_TRANSIT_PER_PEER = 10
)

type P2PNode struct {
	Version           int
	NetworkAddress    string
	ConnectedPeers    []string
	ForwardedAddrList []string
	Blockchain        *blockchain.BlockChain
}

// ======= Handle requests =======

func (node *P2PNode) handleVersionMsg(msg []byte) {
	var versionMsg versionMessage
	genericDeserialize(msg, &versionMsg)

	if node.Version == versionMsg.Version {
		node.sendVerackMsg(versionMsg.AddrMe)
		if !slices.Contains(node.ConnectedPeers, versionMsg.AddrMe) {
			node.sendVersionMsg(versionMsg.AddrMe)
		}
	}
}

func (node *P2PNode) handleVerackMsg(msg []byte) {
	var verackMsg verackMessage
	genericDeserialize(msg, &verackMsg)

	node.ConnectedPeers = append(node.ConnectedPeers, verackMsg.AddrFrom)
	node.sendAddrMsg(verackMsg.AddrFrom)
	node.sendGetBlocksMsg(verackMsg.AddrFrom)
}

func (node *P2PNode) handleAddrMsg(msg []byte) {
	var addrMsg addrMessage
	genericDeserialize(msg, &addrMsg)

	if !slices.Contains(node.ConnectedPeers, addrMsg.Address) {
		node.sendVersionMsg(addrMsg.Address)
	}

	if !slices.Contains(node.ForwardedAddrList, addrMsg.Address) {
		node.ForwardedAddrList = append(node.ForwardedAddrList, addrMsg.Address)
		for _, peerAddr := range node.ConnectedPeers {
			if peerAddr != addrMsg.Address {
				sentData := append(msgTypeToBytes(ADDR_MSG), msg...)
				sendMessage(peerAddr, sentData)
			}
		}
	}
}

func (node *P2PNode) handleGetblocksMsg(msg []byte) {
	var getblocksMsg getblocksMessage
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
		invMsg := invMessage{blockHashesToSend}
		node.sendInvMessage(getblocksMsg.AddrFrom, &invMsg)
	} else if !blockExisted {
		node.sendGetBlocksMsg(getblocksMsg.AddrFrom)
	}
}

func (node *P2PNode) handleGetdataMsg(msg []byte) {
	var getdataMsg getdataMessage
	genericDeserialize(msg, &getdataMsg)

	blockList := node.Blockchain.GetBlocksFromHashes(getdataMsg.HashList)
	sendMessage(getdataMsg.AddrFrom, serialize(blockList))
}

func (node *P2PNode) handleInvMsg(msg []byte) {
	var invMsg invMessage
	genericDeserialize(msg, &invMsg)

	getdataMsgList := []getdataMessage{}
	for _, blockHash := range invMsg.HashList {
		lastIndex := len(getdataMsgList) - 1
		if len(getdataMsgList) == 0 || len(getdataMsgList[lastIndex].HashList) >= MAX_BLOCKS_IN_TRANSIT_PER_PEER {
			newHashList := [][]byte{blockHash}
			newGetDataMsg := getdataMessage{newHashList, node.NetworkAddress}
			getdataMsgList = append(getdataMsgList, newGetDataMsg)
		} else {
			getdataMsgList[lastIndex].HashList = append(getdataMsgList[lastIndex].HashList, blockHash)
		}
	}

	// Send getdata messages to peers
	var wg sync.WaitGroup
	var mutex sync.Mutex
	msgIndex := 1
	msgNum := len(getdataMsgList)
	wg.Add(msgNum)

	for peerIndex, peerAddr := range node.ConnectedPeers {
		if peerIndex >= msgNum {
			break
		}
		go func(toAddress string) {
			for {
				isLastMsg := false
				mutex.Lock()
				if msgIndex >= msgNum {
					return
				} else if msgIndex == msgNum-1 {
					isLastMsg = true
				}
				getdataMsg := getdataMsgList[msgIndex]
				msgIndex++
				mutex.Unlock()

				sentData := append(msgTypeToBytes(GETDATA_MSG), serialize(getdataMsg)...)
				conn, _ := net.Dial(protocol, toAddress)
				conn.Write(sentData)
				defer conn.Close()

				response, _ := io.ReadAll(conn)
				var blockList []*blockchain.Block
				genericDeserialize(response, &blockList)

				for _, block := range blockList {
					node.Blockchain.SetBlock(block)
				}
				if isLastMsg {
					node.Blockchain.SetLastHash(blockList[len(blockList)-1].Hash)
				}
				wg.Done()
			}
		}(peerAddr)
	}
	wg.Wait()
	for _, peerAddr := range node.ConnectedPeers {
		node.sendGetBlocksMsg(peerAddr)
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
	default:
		fmt.Println("invalid message")
	}
}

// ======= Send messages =======

func (node *P2PNode) sendInvMessage(toAddress string, invMsg *invMessage) {
	fmt.Println("Send Inv msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(INV_MSG), serialize(invMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendGetBlocksMsg(toAddress string) {
	fmt.Println("Send Getblocks msg from", node.NetworkAddress, "to", toAddress)
	lastBlockHash := node.Blockchain.LastHash
	getblocksMsg := getblocksMessage{lastBlockHash, node.NetworkAddress}
	sentData := append(msgTypeToBytes(GETBLOCKS_MSG), serialize(getblocksMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendAddrMsg(toAddress string) {
	fmt.Println("Send Addr msg from", node.NetworkAddress, "to", toAddress)
	addrMsg := addrMessage{node.NetworkAddress}
	sentData := append(msgTypeToBytes(ADDR_MSG), serialize(addrMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendVersionMsg(toAddress string) {
	fmt.Println("Send Version msg from", node.NetworkAddress, "to", toAddress)
	nBestHeight := node.Blockchain.GetHeight()
	versionMsg := versionMessage{node.Version, toAddress, node.NetworkAddress, nBestHeight}
	sentData := append(msgTypeToBytes(VERSION_MSG), serialize(versionMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *P2PNode) sendVerackMsg(toAddress string) {
	fmt.Println("Send Verack msg from", node.NetworkAddress, "to", toAddress)
	verackMsg := verackMessage{node.NetworkAddress}
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

func NewBlockChainNode(networkAddress, walletAddress string) *P2PNode {
	localBlockchain := blockchain.InitBlockChain(networkAddress, walletAddress)
	blockchainNode := P2PNode{
		Version:        1,
		NetworkAddress: networkAddress,
		Blockchain:     localBlockchain,
	}
	return &blockchainNode
}
