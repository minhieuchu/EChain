package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/davecgh/go-spew/spew"
)

var (
	initialPeers   = []string{}
	connectedPeers = []string{}
)

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
	nVersion      = 1
	msgTypeLength = 12 // First 12 bytes of each byte slice exchanged between peers are reserved for message type
)

type version_message struct {
	nVersion    int
	addrYou     string
	addrMe      string
	nBestHeight int
}

var blockchainNode = struct {
	nodeAddress   string
	walletAddress string
}{}

func sendMessage(toAddress string, msg []byte) {
	conn, err := net.Dial(protocol, toAddress)
	handleError(err)
	conn.Write(msg)
}

func msgTypeToBytes(msgType string) []byte {
	var res [msgTypeLength]byte
	for i := 0; i < len(msgType); i++ {
		res[i] = msgType[i]
	}
	return res[:]
}

func getMsgType(msg []byte) string {
	return string(msg[:msgTypeLength])
}

func handleVersionMsg(msg []byte) {
	var versionMsg version_message
	byteBuffer := bytes.NewBuffer(msg)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&versionMsg)
	spew.Dump(versionMsg)
}

func handleVerackMsg(msg []byte) {
}

func handleConection(conn net.Conn) {
	data, err := io.ReadAll(conn)
	handleError(err)

	msgType := getMsgType(data)
	switch msgType {
	case VERSION_MSG:
		handleVersionMsg(data[msgTypeLength:])
	case VERACK_MSG:
		handleVerackMsg(data[msgTypeLength:])
	default:
		fmt.Println("default")
	}
}

func sendVersionMsg(toAddress string) {
	nBestHeight := 1
	versionMsg := version_message{nVersion, toAddress, blockchainNode.nodeAddress, nBestHeight}
	sentData := append(msgTypeToBytes(VERSION_MSG), serialize(versionMsg)...)
	sendMessage(toAddress, sentData)
}

func StartBlockChainNode(nodeAddress, walletAddress string) {
	blockchainNode.nodeAddress = nodeAddress
	blockchainNode.walletAddress = walletAddress

	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Fatal("can not start server at ", nodeAddress)
	}

	for _, peer := range initialPeers {
		sendVersionMsg(peer)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err.Error())
		}

		go handleConection(conn)
	}
}
