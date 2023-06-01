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
	nodeAddress := addressList[0] // miner node's wallet address

	chain := blockchain.InitBlockChain(nodeAddress)
	defer chain.DataBase.Close()

	// ======= Testing =======

	network.StartBlockChainNode("localhost:8333", nodeAddress)
}
