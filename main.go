package main

import (
	"EChain/network"
	"os"
)

const (
	NETWORK_NODES_NUM  = 1
	FULLNODE_BLOCK_NUM = 50
)

func main() {
	networkAddress := os.Args[1]
	nodeType := os.Args[2]

	if nodeType == network.FULLNODE {
		fullNode := network.NewFullNode(networkAddress, "15Hgpfs67bXWcFPHxF4mCjSbtXXMwbttge")
		fullNode.StartP2PNode()
	} else if nodeType == network.MINER {
		minerNode := network.NewMinerNode(networkAddress, "15Hgpfs67bXWcFPHxF4mCjSbtXXMwbttge")
		minerNode.StartP2PNode()
	} else if nodeType == network.SPV {
		spvNode := network.NewSPVNode(networkAddress)
		spvNode.StartP2PNode()
	}
}
