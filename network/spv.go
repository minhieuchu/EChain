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

type SPVNode struct {
	P2PNode
	BlockChainHeader *blockchain.BlockChainHeader
	monitorAddrList  []string // list of wallet addresses monitored by SPV node
	bloomFilter      []string
}

func NewSPVNode(networkAddress string) *SPVNode {
	localBlockchainHeader := blockchain.InitBlockChainHeader(networkAddress)
	p2pNode := P2PNode{
		Version:        1,
		NetworkAddress: networkAddress,
	}
	return &SPVNode{
		P2PNode:          p2pNode,
		BlockChainHeader: localBlockchainHeader,
	}
}

func (node *SPVNode) sendAddrMsg(toAddress string) {
	fmt.Println("Send Addr msg from", node.NetworkAddress, "to", toAddress)
	addrMsg := AddrMessage{node.NetworkAddress}
	sentData := append(msgTypeToBytes(ADDR_MSG), serialize(addrMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *SPVNode) sendVersionMsg(toAddress string) {
	fmt.Println("Send Version msg from", node.NetworkAddress, "to", toAddress)
	nBestHeight := node.BlockChainHeader.GetHeight()
	versionMsg := VersionMessage{node.Version, toAddress, node.NetworkAddress, nBestHeight}
	sentData := append(msgTypeToBytes(VERSION_MSG), serialize(versionMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *SPVNode) sendVerackMsg(toAddress string) {
	fmt.Println("Send Verack msg from", node.NetworkAddress, "to", toAddress)
	verackMsg := VerackMessage{SPV, node.NetworkAddress}
	sentData := append(msgTypeToBytes(VERACK_MSG), serialize(verackMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *SPVNode) sendGetheadersMsg(toAddress string) {
	fmt.Println("Send Getheaders msg from", node.NetworkAddress, "to", toAddress)
	lastHeaderHash := node.BlockChainHeader.LastHash
	getheadersMsg := GetheadersMessage{lastHeaderHash, node.NetworkAddress}
	sentData := append(msgTypeToBytes(GETHEADERS_MSG), serialize(getheadersMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *SPVNode) sendHeaderMessage(toAddress string, headerMsg *HeaderMessage) {
	fmt.Println("Send Headers msg from", node.NetworkAddress, "to", toAddress)
	sentData := append(msgTypeToBytes(HEADERS_MSG), serialize(headerMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *SPVNode) sendFilterloadMsg(toAddress string) {
	fmt.Println("Send filterload msg from", node.NetworkAddress, "to", toAddress)
	filterloadMsg := FilterloadMessage{node.NetworkAddress, node.bloomFilter}
	sentData := append(msgTypeToBytes(FILTERLOAD_MSG), serialize(filterloadMsg)...)
	sendMessage(toAddress, sentData)
}

func (node *SPVNode) handleConnection(conn net.Conn) {
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
	case GETHEADERS_MSG:
		node.handleGetheadersMsg(payload)
	case HEADERS_MSG:
		node.handleHeadersMsg(payload)
	case NEWADDR_MSG:
		node.handleNewAddrMsg(payload)
	default:
		fmt.Println("invalid message")
	}
}

func (node *SPVNode) handleNewAddrMsg(msg []byte) {
	var newAddrMsg NewAddrMessage
	genericDeserialize(msg, &newAddrMsg)
	node.monitorAddrList = append(node.monitorAddrList, newAddrMsg.WalletAddress)
	node.updateBloomFilter()
	for _, peerNode := range node.connectedPeers {
		if peerNode.NodeType == FULLNODE {
			node.sendFilterloadMsg(peerNode.Address)
		}
	}
}

func (node *SPVNode) handleHeadersMsg(msg []byte) {
	var headerMsg HeaderMessage
	genericDeserialize(msg, &headerMsg)

	for _, header := range headerMsg.HeaderList {
		node.BlockChainHeader.SetHeader(header)
	}
	node.BlockChainHeader.SetLastHash(headerMsg.HeaderList[len(headerMsg.HeaderList)-1].GetHash())
}

func (node *SPVNode) handleGetheadersMsg(msg []byte) {
	var getheadersMsg GetheadersMessage
	genericDeserialize(msg, &getheadersMsg)

	remoteLastHeaderHash := getheadersMsg.TopHeaderHash
	headerExisted, unmatchedHeaders := node.BlockChainHeader.GetUnmatchedHeaders(remoteLastHeaderHash)
	if headerExisted && len(unmatchedHeaders) > 0 {
		headerList := []*blockchain.BlockHeader{}
		for i := len(unmatchedHeaders) - 1; i >= 0; i-- {
			headerList = append(headerList, unmatchedHeaders[i])
			if len(headerList) >= 2000 {
				break
			}
		}
		headerMsg := HeaderMessage{headerList}
		node.sendHeaderMessage(getheadersMsg.AddrFrom, &headerMsg)
	} else if !headerExisted {
		node.sendGetheadersMsg(getheadersMsg.AddrFrom)
	}
}

func (node *SPVNode) handleVersionMsg(msg []byte) {
	var versionMsg VersionMessage
	genericDeserialize(msg, &versionMsg)

	if node.Version == versionMsg.Version {
		node.sendVerackMsg(versionMsg.AddrMe)
		if !slices.Contains(node.getConnectedNodeAddresses(), versionMsg.AddrMe) {
			node.sendVersionMsg(versionMsg.AddrMe)
		}
	}
}

func (node *SPVNode) handleVerackMsg(msg []byte) {
	var verackMsg VerackMessage
	genericDeserialize(msg, &verackMsg)

	if slices.Contains(node.getConnectedNodeAddresses(), verackMsg.AddrFrom) {
		return
	}
	node.connectedPeers = append(node.connectedPeers, NodeInfo{verackMsg.NodeType, verackMsg.AddrFrom})
	node.sendAddrMsg(verackMsg.AddrFrom)
	node.sendGetheadersMsg(verackMsg.AddrFrom)
}

func (node *SPVNode) handleAddrMsg(msg []byte) {
	var addrMsg AddrMessage
	genericDeserialize(msg, &addrMsg)

	if !slices.Contains(node.getConnectedNodeAddresses(), addrMsg.Address) {
		node.sendVersionMsg(addrMsg.Address)
	}

	if !slices.Contains(node.forwardedAddrList, addrMsg.Address) {
		node.forwardedAddrList = append(node.forwardedAddrList, addrMsg.Address)
		for _, peerAddr := range node.connectedPeers {
			if peerAddr.Address != addrMsg.Address {
				sentData := append(msgTypeToBytes(ADDR_MSG), msg...)
				sendMessage(peerAddr.Address, sentData)
			}
		}
	}
}

func (node *SPVNode) updateBloomFilter() {
	// Todo: Implement a realistic bloom filter
	node.bloomFilter = node.monitorAddrList
}

func (node *SPVNode) StartP2PNode() {
	fmt.Println(" ===== Starting blockchain node at", node.NetworkAddress, "=====")
	ln, err := net.Listen(protocol, node.NetworkAddress)
	if err != nil {
		fmt.Println("can not start server at", node.NetworkAddress)
		return
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
