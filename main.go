package main

import (
	"EChain/blockchain"
	"EChain/wallet"

	"fmt"
)

func main() {
	wallets := wallet.LoadWallets()
	addressList := wallets.GetAddresses()
	nodeAddress := addressList[0] // miner node's address
	otherAddress := addressList[1]

	chain := blockchain.InitBlockChain(nodeAddress)
	defer chain.DataBase.Close()

	wallets.Transfer(nodeAddress, otherAddress, 500, chain)
	fmt.Println(chain.GetBalance(nodeAddress), chain.GetBalance(otherAddress))
}
