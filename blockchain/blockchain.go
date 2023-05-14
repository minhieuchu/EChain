package blockchain

import (
	"errors"
	"log"
	"time"

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
	blockchain := BlockChain{db, genesisBlock.Hash}
	blockchain.StoreNewBlock(genesisBlock)

	return &blockchain
}

func (blockchain *BlockChain) StoreNewBlock(block *Block) {
	blockchain.LastHash = block.Hash
	blockchain.DataBase.Put(block.Hash, Encode(block), nil)
	blockchain.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), block.Hash, nil)
}

func (blockchain *BlockChain) AddBlock(transactions []*Transaction) {
	newBlock := Block{
		Transactions: transactions,
		Timestamp:    time.Now().String(),
		PrevHash:     blockchain.LastHash,
	}
	newBlock.Mine()
	blockchain.StoreNewBlock(&newBlock)
}

func (blockchain *BlockChain) GetUnspentTransactionOutputs(address string) []surplusTxOutput {
	unspentTransactionOutputs := []surplusTxOutput{}
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}
	spentTxnOutputs := map[string][]int{} // Mapping between transaction hash and spent transaction output indexes

	// Scan through the blockchain starting from the most recent block
	for {
		encodedBlock, err := chainIterator.DataBase.Get(chainIterator.CurrentHash, nil)
		HandleErr(err)
		currentBlock := DecodeBlock(encodedBlock)

		for _, transaction := range currentBlock.Transactions {
			for outputIndex, txOutput := range transaction.Outputs {
				if !slices.Contains(spentTxnOutputs[string(transaction.Hash)], outputIndex) && txOutput.CanBeUnlocked(address) {
					surplusOutput := surplusTxOutput{
						TxOutput:    txOutput,
						TxHash:      transaction.Hash,
						OutputIndex: outputIndex,
					}
					unspentTransactionOutputs = append(unspentTransactionOutputs, surplusOutput)
				}
			}
			for _, txInput := range transaction.Inputs {
				if txInput.IsSignedBy(address) {
					spentTxnOutputs[string(txInput.TxHash)] = append(spentTxnOutputs[string(txInput.TxHash)], txInput.OutputIndex)
				}
			}
		}

		if len(currentBlock.PrevHash) == 0 {
			break
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}
	return unspentTransactionOutputs
}

func (blockchain *BlockChain) GetBalance(address string) int {
	balance := 0
	unspentTransactionOutputs := blockchain.GetUnspentTransactionOutputs(address)

	for _, txnOutput := range unspentTransactionOutputs {
		balance += txnOutput.Amount
	}

	return balance
}

func (blockchain *BlockChain) Transfer(fromAddress, toAddress string, amount int) error {
	transferAmount := 0
	unspentTxnOutputs := blockchain.GetUnspentTransactionOutputs(fromAddress)
	newTxnInputs := []TxInput{}
	newTxnOutputs := []TxOutput{}

	for _, txnOutput := range unspentTxnOutputs {
		transferAmount += txnOutput.Amount
		newTxnInputs = append(newTxnInputs, TxInput{txnOutput.TxHash, txnOutput.OutputIndex, fromAddress})
		if transferAmount >= amount {
			break
		}
	}

	if transferAmount < amount {
		return errors.New("not enough balance to transfer")
	}

	newTxnOutputs = append(newTxnOutputs, TxOutput{amount, toAddress})
	if transferAmount > amount {
		newTxnOutputs = append(newTxnOutputs, TxOutput{transferAmount - amount, fromAddress})
	}

	newTransaction := Transaction{[]byte{}, newTxnInputs, newTxnOutputs}
	newTransaction.SetHash()
	blockchain.AddBlock([]*Transaction{&newTransaction})

	return nil
}
