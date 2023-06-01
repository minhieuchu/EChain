package main

import (
	"EChain/blockchain"
	"EChain/network"
	"EChain/wallet"
)

func main() {
	// ======= Init =======

	wallets := wallet.LoadWallets()
	addressList := wallets.GetAddresses()
	walletAddress := addressList[0] // miner node's wallet address

	chain := blockchain.InitBlockChain(walletAddress)
	defer chain.DataBase.Close()

	// ======= Testing =======

	go func() {
		network.StartBlockChainNode("localhost:8333")
	}()
	go func() {
		network.StartBlockChainNode("localhost:8334")
	}()
	network.StartBlockChainNode("localhost:8335")
}
