package network

import (
	"net"
)

var initialPeers = []string{"localhost:8333", "localhost:8334", "localhost:8335"}

const (
	VERSION_MSG     = "version"
	VERACK_MSG      = "verack"
	ADDR_MSG        = "addr"
	GETBLOCKS_MSG   = "getblocks"
	INV_MSG         = "inv"
	HEADERS_MSG     = "headers"
	GETDATA_MSG     = "getdata"
	GETHEADERS_MSG  = "getheaders"
	BLOCKDATA_MSG   = "blockdata"
	HEADERDATA_MSG  = "headerdata"
	GETUTXO_MSG     = "getutxo"
	NEWTXN_MSG      = "newtxn"
	NEWADDR_MSG     = "newaddr"
	FILTERLOAD_MSG  = "filterload"
	MERKLEBLOCK_MSG = "merkleblock"
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

type NodeInfo struct {
	NodeType string
	Address  string
}

type P2PNode struct {
	Version           int
	NetworkAddress    string
	connectedPeers    []NodeInfo
	forwardedAddrList []string
}

type Node interface {
	sendVersionMsg(string)
	sendVerackMsg(string)
	sendAddrMsg(string)
	handleVersionMsg([]byte)
	handleVerackMsg([]byte)
	handleAddrMsg([]byte)
	handleConnection(net.Conn)
	StartP2PNode()
}

func (node *P2PNode) getConnectedNodeAddresses() []string {
	var addrList []string
	for _, connectedNode := range node.connectedPeers {
		addrList = append(addrList, connectedNode.Address)
	}
	return addrList
}
