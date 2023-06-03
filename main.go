package main

import (
	"EChain/blockchain"
	"EChain/network"
	"EChain/wallet"
	"fmt"
	"sync"
)

const NETWORK_NODES_NUM = 3

func main() {
	// ======= Init =======

	wallets := wallet.LoadWallets()
	addressList := wallets.GetAddresses()
	walletAddress := addressList[0] // miner node's wallet address

	chain := blockchain.InitBlockChain(walletAddress)
	defer chain.DataBase.Close()

	// ======= Testing =======

	var wg sync.WaitGroup
	for i := 0; i < NETWORK_NODES_NUM; i++ {
		wg.Add(1)
		portNumber := 8333 + i
		go func() {
			defer wg.Done()
			network.StartBlockChainNode("127.0.0.1:" + fmt.Sprint(portNumber), walletAddress)
		}()
	}
	wg.Wait()
}
