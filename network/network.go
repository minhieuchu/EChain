package network

import (
	"EChain/blockchain"

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
	GETADDR_MSG   = "getaddr"
	GETBLOCKS_MSG = "getblocks"
	INV_MSG       = "inv"
	GETDATA_MSG   = "getdata"
)

const (
	protocol      = "tcp"
	msgTypeLength = 12 // First 12 bytes of each byte slice exchanged between peers are reserved for message type
)

type p2pNode struct {
	nVersion          int
	networkAddress    string
	connectedPeers    []string
	forwardedAddrList []string
	blockchain        *blockchain.BlockChain
}

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
		// Send inv message
	} else if !blockExisted {
		node.sendGetBlocksMsg(getblocksMsg.AddrFrom)
	}
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
	default:
		fmt.Println("invalid message")
	}
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
