package network

import (
	"EChain/blockchain"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type MinerNode struct {
	FullNode
	mempool          []*blockchain.Transaction
	recipientAddress string // Address to receive block reward after mining new blocks
}

func NewMinerNode(networkAddress, walletAddress string) *MinerNode {
	fullNode := NewFullNode(networkAddress, walletAddress)
	return &MinerNode{
		FullNode:         *fullNode,
		recipientAddress: walletAddress,
	}
}

func (node *MinerNode) StartP2PNode() {
	fmt.Println(" ===== Starting blockchain node at", node.NetworkAddress, "=====")
	ln, err := net.Listen(protocol, node.NetworkAddress)
	if err != nil {
		log.Fatal("can not start server at", node.NetworkAddress)
	}

	go func() {
		time.Sleep(2 * time.Second)
		for _, peerAddr := range initialPeers {
			if peerAddr != node.NetworkAddress {
				node.FullNode.sendVersionMsg(peerAddr)
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

func (node *MinerNode) handleNewTxnMsg(msg []byte) {
	var newTransaction blockchain.Transaction
	genericDeserialize(msg, &newTransaction)

	err := node.FullNode.handleNewTxnMsg(msg)
	if err != nil {
		return
	}
	node.mempool = append(node.mempool, &newTransaction)
}

func (node *MinerNode) handleConnection(conn net.Conn) {
	data, err := io.ReadAll(conn)
	defer conn.Close()
	handleError(err)

	msgType := getMsgType(data)
	payload := data[msgTypeLength:]

	switch msgType {
	case NEWTXN_MSG:
		node.handleNewTxnMsg(payload)
	}
}
