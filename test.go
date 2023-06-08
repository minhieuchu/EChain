package main

import (
	"EChain/blockchain"
	"EChain/network"
	"EChain/wallet"
	"fmt"
	"sync"
	"time"
)

func runTest() {
	// ======= Init =======

	wallets := wallet.LoadWallets()
	addressList := wallets.GetAddresses()
	walletAddress := addressList[0] // miner node's wallet address

	// ======= Testing =======

	var wg sync.WaitGroup
	for i := 0; i < NETWORK_NODES_NUM; i++ {
		wg.Add(1)
		portNumber := 8333 + i
		go func() {
			defer wg.Done()
			fullnode := network.NewFullNode("localhost:"+fmt.Sprint(portNumber), walletAddress)
			for i := 0; i < FULLNODE_BLOCK_NUM; i++ {
				var block blockchain.Block
				lastHash, _ := fullnode.Blockchain.DataBase.Get([]byte(blockchain.LAST_HASH_STOGAGE_KEY), nil)
				block.PrevHash = lastHash
				block.Mine()
				fullnode.Blockchain.StoreNewBlock(&block)
			}
			fullnode.StartP2PNode()
		}()
	}
	go func() {
		time.Sleep(3 * time.Second)
		wg.Add(1)
		defer wg.Done()
		spvNode := network.NewSPVNode("localhost:8888")
		spvNode.StartP2PNode()
	}()
	wg.Wait()
}
