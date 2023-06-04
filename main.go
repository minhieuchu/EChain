package main

import (
	"EChain/blockchain"
	"EChain/network"
	"EChain/wallet"
	"fmt"
	"sync"
	"time"
)

const NETWORK_NODES_NUM = 3

func main() {
	// ======= Init =======

	wallets := wallet.LoadWallets()
	addressList := wallets.GetAddresses()
	walletAddress := addressList[0] // miner node's wallet address

	// ======= Testing =======

	var wg sync.WaitGroup
	for i := 0; i < NETWORK_NODES_NUM; i++ {
		wg.Add(1)
		portNumber := 8333 + i
		var blockchainNode *network.P2PNode
		go func() {
			defer wg.Done()
			var transaction blockchain.Transaction
			blockchainNode = network.NewBlockChainNode("127.0.0.1:"+fmt.Sprint(portNumber), walletAddress)
			blockchainNode.Blockchain.AddBlock([]*blockchain.Transaction{&transaction})
			blockchainNode.StartP2PNode()
		}()
	}
	go func() {
		time.Sleep(3 * time.Second)
		wg.Add(1)
		defer wg.Done()
		blockchainNode := network.NewBlockChainNode("127.0.0.1:8888", walletAddress)
		blockchainNode.StartP2PNode()
	}()
	wg.Wait()
}
