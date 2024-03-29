package network

import (
	"EChain/blockchain"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"time"

	"golang.org/x/exp/slices"
)

type MinerNode struct {
	FullNode
	recipientAddress string // Address to receive block reward after mining new blocks
}

func NewMinerNode(networkAddress, walletAddress string) *MinerNode {
	fullNode := NewFullNode(networkAddress)
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

	go func() {
		time.Sleep(5 * time.Second)
		for {
			node.startMining()
			time.Sleep(10 * time.Second)
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

func (node *MinerNode) mineBlock(newBlock *blockchain.Block) {
	nonce := 1
	for {
		newBlock.Nonce = nonce
		hashValue := new(big.Int).SetBytes(newBlock.GetHash())

		if hashValue.Cmp(blockchain.TARGET_HASH) == -1 {
			newBlock.Nonce = nonce
			break
		}
		nonce++
	}
}

func (node *MinerNode) startMining() {
	txnList := []*blockchain.Transaction{}
	// Simply take all transactions in mempool to new block
	txnList = append(txnList, node.mempool...)

	coinbaseTxn := blockchain.CoinBaseTransaction(node.recipientAddress)
	newBlock := blockchain.Block{
		BlockHeader: blockchain.BlockHeader{
			Timestamp: time.Now().String(),
			PrevHash:  node.Blockchain.LastHash,
		},
		Transactions: append([]*blockchain.Transaction{coinbaseTxn}, txnList...),
	}
	node.mineBlock(&newBlock)

	// Step 1: Update local blockchain & UTXO set
	node.storeNewBlock(&newBlock)

	// Step 2: Relay new block to other full nodes / miner nodes
	for _, connectedNode := range node.connectedPeers {
		if connectedNode.NodeType == FULLNODE || connectedNode.NodeType == MINER {
			node.FullNode.sendBlockdataMessage(connectedNode.Address, NEWBLOCK_FROM_MINER_INDEX, []*blockchain.Block{&newBlock})
		}
	}
}

func (node *MinerNode) sendVerackMsg(toAddress string) {
	fmt.Println("Send Verack msg from", node.NetworkAddress, "to", toAddress)
	verackMsg := VerackMessage{MINER, node.NetworkAddress}
	sentData := append(msgTypeToBytes(VERACK_MSG), serialize(verackMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *MinerNode) handleVersionMsg(msg []byte) {
	var versionMsg VersionMessage
	genericDeserialize(msg, &versionMsg)

	if node.Version == versionMsg.Version {
		node.sendVerackMsg(versionMsg.AddrMe)
		if !slices.Contains(node.getConnectedNodeAddresses(), versionMsg.AddrMe) {
			node.sendVersionMsg(versionMsg.AddrMe)
		}
	}
}

func (node *MinerNode) handleConnection(conn net.Conn) {
	data, err := io.ReadAll(conn)
	defer conn.Close()
	handleError(err)

	msgType := getMsgType(data)
	payload := data[msgTypeLength:]

	switch msgType {
	case VERSION_MSG:
		node.handleVersionMsg(payload)
	case VERACK_MSG:
		node.FullNode.handleVerackMsg(payload)
	case ADDR_MSG:
		node.FullNode.handleAddrMsg(payload)
	case GETBLOCKS_MSG:
		node.FullNode.handleGetblocksMsg(payload)
	case INV_MSG:
		node.FullNode.handleInvMsg(payload)
	case GETDATA_MSG:
		node.FullNode.handleGetdataMsg(payload)
	case BLOCKDATA_MSG:
		node.FullNode.handleBlockdataMsg(payload)
	case GETHEADERS_MSG:
		node.FullNode.handleGetheadersMsg(payload)
	case GETUTXO_MSG:
		node.FullNode.handeGetUTXOMsg(conn, payload)
	case NEWTXN_MSG:
		node.FullNode.handleNewTxnMsg(payload)
	default:
		fmt.Println("invalid message")
	}
}
