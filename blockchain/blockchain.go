package blockchain

import (
	"log"

	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/exp/slices"
)

type BlockChain struct {
	DataBase *leveldb.DB
	LastHash []byte
}

type BlockChainIterator struct {
	DataBase    *leveldb.DB
	CurrentHash []byte
}

func InitBlockChain(firstAddress string) *BlockChain {
	db, err := leveldb.OpenFile("storage", nil) // Entrust the task of closing levelDB to the caller site
	if err != nil {
		log.Fatal(err)
	}

	genesisBlock := Genesis(firstAddress)
	db.Put(genesisBlock.Hash, Encode(genesisBlock), nil)
	db.Put([]byte(LAST_HASH_STOGAGE_KEY), genesisBlock.Hash, nil)

	return &BlockChain{db, genesisBlock.Hash}
}

func (blockchain *BlockChain) GetBalance(address string) int {
	balance := 0
	spentTxOuts := map[string][]int{} // mapping between transaction hash and transaction output indexes
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}

	// Scan through the blockchain starting from the most recent block
	// to find UTXOs binded to the input address
	for {
		encodedBlock, err := chainIterator.DataBase.Get(chainIterator.CurrentHash, nil)
		HandleErr(err)
		currentBlock := DecodeBlock(encodedBlock)
		for _, transaction := range currentBlock.Transactions {
			transactionHash := string(transaction.Hash)
			for outputIndex, txOutput := range transaction.Outputs {
				if !slices.Contains(spentTxOuts[transactionHash], outputIndex) && txOutput.CanBeUnlocked(address) {
					balance += txOutput.Amount
				}
			}

			for _, txInput := range transaction.Inputs {
				if txInput.IsSignedBy(address) {
					spentTxOuts[string(txInput.TxHash)] = append(spentTxOuts[string(txInput.TxHash)], txInput.OutputIndex)
				}
			}
		}
		if len(currentBlock.PrevHash) == 0 {
			break
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}

	return balance
}
