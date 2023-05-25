package main

import (
	"EChain/blockchain"
	"EChain/wallet"
)

func main() {
	wallets := wallet.LoadWallets()
	nodeAddress := wallets.GetAddresses()[0]

	chain := blockchain.InitBlockChain(nodeAddress)
	defer chain.DataBase.Close()
}
