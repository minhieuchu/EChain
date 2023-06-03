package network

import (
	"EChain/blockchain"
	"sync"

	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/davecgh/go-spew/spew"
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

type p2pNode struct {
	nVersion          int
	networkAddress    string
	connectedPeers    []string
	forwardedAddrList []string
	blockchain        *blockchain.BlockChain
}

// ======= Handle requests =======

func (node *p2pNode) handleVersionMsg(msg []byte) {
	var versionMsg versionMessage
	genericDeserialize(msg, &versionMsg)

	if node.nVersion == versionMsg.Version {
		node.sendVerackMsg(versionMsg.AddrMe)
		if !slices.Contains(node.connectedPeers, versionMsg.AddrMe) {
			node.sendVersionMsg(versionMsg.AddrMe)
		}
	}
}

func (node *p2pNode) handleVerackMsg(msg []byte) {
	var verackMsg verackMessage
	genericDeserialize(msg, &verackMsg)

	node.connectedPeers = append(node.connectedPeers, verackMsg.AddrFrom)
	node.sendAddrMsg(verackMsg.AddrFrom)
	node.sendGetBlocksMsg(verackMsg.AddrFrom)
}

func (node *p2pNode) handleAddrMsg(msg []byte) {
	var addrMsg addrMessage
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

func (node *p2pNode) handleGetblocksMsg(msg []byte) {
	var getblocksMsg getblocksMessage
	genericDeserialize(msg, &getblocksMsg)

	remoteLastBlockHash := getblocksMsg.TopBlockHash
	blockExisted, unmatchedBlocks := node.blockchain.GetUnmatchedBlocks(remoteLastBlockHash)
	if blockExisted && len(unmatchedBlocks) > 0 {
		blockHashesToSend := [][]byte{}
		for i := len(unmatchedBlocks) - 1; i >= 0; i-- {
			blockHashesToSend = append(blockHashesToSend, unmatchedBlocks[i])
			if len(blockHashesToSend) > 500 {
				break
			}
		}
		invMsg := invMessage{blockHashesToSend}
		node.sendInvMessage(getblocksMsg.AddrFrom, &invMsg)
	} else if !blockExisted {
		node.sendGetBlocksMsg(getblocksMsg.AddrFrom)
	}
}

func (node *p2pNode) handleGetdataMsg(msg []byte) {
	var getdataMsg getdataMessage
	genericDeserialize(msg, &getdataMsg)

	blockList := node.blockchain.GetBlocksFromHashes(getdataMsg.HashList)
	sendMessage(getdataMsg.AddrFrom, serialize(blockList))
}

func (node *p2pNode) handleInvMsg(msg []byte) {
	var invMsg invMessage
	genericDeserialize(msg, &invMsg)

	getdataMsgList := []getdataMessage{}
	for _, blockHash := range invMsg.HashList {
		lastIndex := len(getdataMsgList) - 1
		if len(getdataMsgList) == 0 || len(getdataMsgList[lastIndex].HashList) >= MAX_BLOCKS_IN_TRANSIT_PER_PEER {
			newHashList := [][]byte{blockHash}
			newGetDataMsg := getdataMessage{newHashList, node.networkAddress}
			getdataMsgList = append(getdataMsgList, newGetDataMsg)
		} else {
			getdataMsgList[lastIndex].HashList = append(getdataMsgList[lastIndex].HashList, blockHash)
		}
	}

	// Send getdata messages to peers
	var wg sync.WaitGroup
	var mutex sync.Mutex
	msgIndex := 1
	wg.Add(len(getdataMsgList))

	for peerIndex, peerAddr := range node.connectedPeers {
		if peerIndex >= len(getdataMsgList) {
			break
		}
		go func(toAddress string) {
			for {
				mutex.Lock()
				if msgIndex >= len(getdataMsgList) {
					return
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
				spew.Dump(blockList)
				wg.Done()
			}
		}(peerAddr)
	}
	wg.Wait()
}

func (node *p2pNode) handleConnection(conn net.Conn) {
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

func (node *p2pNode) sendInvMessage(toAddress string, invMsg *invMessage) {
	fmt.Println("Send Inv msg from", node.networkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(INV_MSG), serialize(invMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *p2pNode) sendGetBlocksMsg(toAddress string) {
	fmt.Println("Send Getblocks msg from", node.networkAddress, "to", toAddress)
	lastBlockHash := node.blockchain.LastHash
	getblocksMsg := getblocksMessage{lastBlockHash, node.networkAddress}
	sentData := append(msgTypeToBytes(GETBLOCKS_MSG), serialize(getblocksMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *p2pNode) sendAddrMsg(toAddress string) {
	fmt.Println("Send Addr msg from", node.networkAddress, "to", toAddress)
	addrMsg := addrMessage{node.networkAddress}
	sentData := append(msgTypeToBytes(ADDR_MSG), serialize(addrMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *p2pNode) sendVersionMsg(toAddress string) {
	fmt.Println("Send Version msg from", node.networkAddress, "to", toAddress)
	nBestHeight := node.blockchain.GetHeight()
	versionMsg := versionMessage{node.nVersion, toAddress, node.networkAddress, nBestHeight}
	sentData := append(msgTypeToBytes(VERSION_MSG), serialize(versionMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *p2pNode) sendVerackMsg(toAddress string) {
	fmt.Println("Send Verack msg from", node.networkAddress, "to", toAddress)
	verackMsg := verackMessage{node.networkAddress}
	sentData := append(msgTypeToBytes(VERACK_MSG), serialize(verackMsg)...)
	sendMessage(toAddress, sentData)
}

func StartBlockChainNode(networkAddress, walletAddress string) {
	localBlockchain := blockchain.InitBlockChain(networkAddress, walletAddress)
	blockchainNode := p2pNode{
		nVersion:       1,
		networkAddress: networkAddress,
		blockchain:     localBlockchain,
	}

	fmt.Println("Starting blockchain node at", networkAddress)
	ln, err := net.Listen(protocol, networkAddress)
	if err != nil {
		log.Fatal("can not start server at", networkAddress)
	}

	go func() {
		time.Sleep(2 * time.Second)
		for _, peerAddr := range initialPeers {
			if peerAddr != blockchainNode.networkAddress {
				blockchainNode.sendVersionMsg(peerAddr)
			}
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err.Error())
		}

		go blockchainNode.handleConnection(conn)
	}
}
