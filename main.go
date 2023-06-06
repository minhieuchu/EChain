package main

import (
	"EChain/blockchain"
	"EChain/network"
	"fmt"
	"os"
	"time"
)

const (
	NETWORK_NODES_NUM  = 1
	FULLNODE_BLOCK_NUM = 50
)

func main() {
	networkAddress := os.Args[1]
	nodeType := os.Args[2]
	blockchainNode := network.NewBlockChainNode(nodeType, networkAddress, "15Hgpfs67bXWcFPHxF4mCjSbtXXMwbttge")
	if nodeType == network.FULLNODE || nodeType == network.MINER {
		for i := 0; i < FULLNODE_BLOCK_NUM; i++ {
			var block blockchain.Block
			lastHash, _ := blockchainNode.Blockchain.DataBase.Get([]byte(blockchain.LAST_HASH_STOGAGE_KEY), nil)
			block.PrevHash = lastHash
			block.Height = i + 1
			block.Mine()
			blockchainNode.Blockchain.StoreNewBlock(&block)
		}
	} else {
		go func() {
			time.Sleep(3 * time.Second)
			fmt.Println("Synchronized blockchain's height:", blockchainNode.Blockchain.GetHeight(), "at", networkAddress)
		}()
	}
	blockchainNode.StartP2PNode()
}
