package main

import (
	"EChain/blockchain"
	"EChain/wallet"

	"fmt"
)

func main() {
	// ======= Init =======

	wallets := wallet.LoadWallets()
	addressList := wallets.GetAddresses()
	nodeAddress := addressList[0] // miner node's wallet address
	otherAddress := addressList[1]

	chain := blockchain.InitBlockChain(nodeAddress)
	defer chain.DataBase.Close()

	wallets.ConnectChain(chain)

	// ======= Testing =======

	wallets.Transfer(nodeAddress, otherAddress, 500)
	fmt.Println(wallets.GetBalance(nodeAddress), wallets.GetBalance(otherAddress))
}
