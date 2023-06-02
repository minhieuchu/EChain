package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"time"
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
	nVersion       int
	networkAddress string
	connectedPeers []string
}

func (node *p2pNode) handleVersionMsg(msg []byte) {
	var versionMsg versionMessage
	byteBuffer := bytes.NewBuffer(msg)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&versionMsg)
	if node.nVersion == versionMsg.Version {
		node.sendVerackMsg(versionMsg.AddrMe)
	}
}

func (node *p2pNode) handleVerackMsg(msg []byte) {
	var verackMsg verackMessage
	byteBuffer := bytes.NewBuffer(msg)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&verackMsg)
	node.connectedPeers = append(node.connectedPeers, verackMsg.AddrFrom)
	// Todo: Send addr message
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
	default:
		fmt.Println("invalid message")
	}
}

func (node *p2pNode) sendVersionMsg(toAddress string) {
	fmt.Println("Send Version msg from", node.networkAddress, "to", toAddress)
	nBestHeight := 1
	versionMsg := versionMessage{node.nVersion, toAddress, node.networkAddress, nBestHeight}
	sentData := append(msgTypeToBytes(VERSION_MSG), serialize(versionMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *p2pNode) sendVerackMsg(toAddress string) {
	verackMsg := verackMessage{node.networkAddress}
	sentData := append(msgTypeToBytes(VERACK_MSG), serialize(verackMsg)...)
	sendMessage(toAddress, sentData)
}

func StartBlockChainNode(networkAddress string) {
	blockchainNode := p2pNode{
		nVersion:       1,
		networkAddress: networkAddress,
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