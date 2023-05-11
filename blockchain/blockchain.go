package blockchain

import (
	badger "github.com/dgraph-io/badger/v3"
)

type BlockChain struct {
	DataBase badger.DB
	LastHash []byte
}

type BlockChainIterator struct {
	DataBase    badger.DB
	CurrentHash []byte
}
