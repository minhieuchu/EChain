package blockchain

import (
	"log"

	"github.com/syndtr/goleveldb/leveldb"
)

const myAddress = "My Address"

type BlockChain struct {
	DataBase *leveldb.DB
	LastHash []byte
}

type BlockChainIterator struct {
	DataBase    *leveldb.DB
	CurrentHash []byte
}

func InitBlockChain() *BlockChain {
	db, err := leveldb.OpenFile("storage", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	genesisBlock := Genesis(myAddress)
	db.Put(genesisBlock.Hash, Encode(genesisBlock), nil)
	db.Put([]byte(LAST_HASH_STOGAGE_KEY), genesisBlock.Hash, nil)

	return &BlockChain{db, genesisBlock.Hash}
}
