package main

import (
	"EChain/blockchain"

	"github.com/google/uuid"
)

var myAddress = uuid.New().String()

func main() {
	chain := blockchain.InitBlockChain(myAddress)
	defer chain.DataBase.Close()
}
