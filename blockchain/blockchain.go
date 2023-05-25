package blockchain

import (
	"crypto/ecdsa"
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

var NODE_ADDRESS string

func InitBlockChain(address string) *BlockChain {
	NODE_ADDRESS = address
	db, err := leveldb.OpenFile("storage", nil) // Entrust the task of closing levelDB to the caller site
	if err != nil {
		log.Fatal(err)
	}

	genesisBlock := Genesis()
	blockchain := BlockChain{db, genesisBlock.Hash}
	blockchain.StoreNewBlock(genesisBlock)

	return &blockchain
}

func (chainIterator *BlockChainIterator) CurrentBlock() *Block {
	encodedBlock, err := chainIterator.DataBase.Get(chainIterator.CurrentHash, nil)
	HandleErr(err)
	return DecodeBlock(encodedBlock)
}

func (blockchain *BlockChain) StoreNewBlock(block *Block) {
	blockchain.LastHash = block.Hash
	blockchain.DataBase.Put(block.Hash, Encode(block), nil)
	blockchain.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), block.Hash, nil)
}

func (blockchain *BlockChain) getTransactionMapFromInputs(transaction Transaction) map[string]Transaction {
	txnIDs := map[string]bool{}
	txnMap := map[string]Transaction{}

	for _, txnInput := range transaction.Inputs {
		txnIDs[string(txnInput.TxID)] = true
	}

	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}
	for {
		currentBlock := chainIterator.CurrentBlock()

		for _, transaction := range currentBlock.Transactions {
			if _, exists := txnIDs[string(transaction.Hash)]; exists {
				txnMap[string(transaction.Hash)] = *transaction
				delete(txnIDs, string(transaction.Hash))
			}
		}

		if len(currentBlock.PrevHash) == 0 {
			break
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}

	return txnMap
}

func (blockchain *BlockChain) AddBlock(transactions []*Transaction) error {
	for _, transaction := range transactions {
		txnMap := blockchain.getTransactionMapFromInputs(*transaction)
		if !transaction.Verify(txnMap) {
			return errors.New("invalid transaction")
		}
	}
	coinbaseTransaction := CoinBaseTransaction(NODE_ADDRESS)
	newBlock := Block{
		Transactions: append([]*Transaction{coinbaseTransaction}, transactions...),
		Timestamp:    time.Now().String(),
		PrevHash:     blockchain.LastHash,
	}
	newBlock.Mine()
	blockchain.StoreNewBlock(&newBlock)
	return nil
}

func (blockchain *BlockChain) GetUnspentTransactionOutputs(address string) []surplusTxOutput {
	unspentTransactionOutputs := []surplusTxOutput{}
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}
	spentTxnOutputs := map[string][]int{} // Mapping between transaction hash and spent transaction output indexes

	// Scan through the blockchain starting from the most recent block
	for {
		currentBlock := chainIterator.CurrentBlock()

		for _, transaction := range currentBlock.Transactions {
			for outputIndex, txOutput := range transaction.Outputs {
				if !slices.Contains(spentTxnOutputs[string(transaction.Hash)], outputIndex) && txOutput.IsBoundTo(address) {
					surplusOutput := surplusTxOutput{
						TxOutput: txOutput,
						TxID:     transaction.Hash,
						VOut:     outputIndex,
					}
					unspentTransactionOutputs = append(unspentTransactionOutputs, surplusOutput)
				}
			}
			for _, txInput := range transaction.Inputs {
				if txInput.IsSignedBy(address) {
					spentTxnOutputs[string(txInput.TxID)] = append(spentTxnOutputs[string(txInput.TxID)], txInput.VOut)
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

// Use both pubkey & fromAddress for validation of pubkey
func (blockchain *BlockChain) Transfer(privKey ecdsa.PrivateKey, pubKey []byte, fromAddress, toAddress string, amount int) error {
	if getAddressFromPubkey(pubKey) != fromAddress {
		return errors.New("public key and sender address do not match")
	}

	transferAmount := 0
	unspentTxnOutputs := blockchain.GetUnspentTransactionOutputs(fromAddress)
	newTxnInputs := []TxInput{}
	newTxnOutputs := []TxOutput{}

	for _, txnOutput := range unspentTxnOutputs {
		transferAmount += txnOutput.Amount
		newTxnInputs = append(newTxnInputs, createTxnInput(txnOutput.TxID, txnOutput.VOut, pubKey))
		if transferAmount >= amount {
			break
		}
	}

	if transferAmount < amount {
		return errors.New("not enough balance to transfer")
	}

	newTxnOutputs = append(newTxnOutputs, createTxnOutput(amount, toAddress))
	if transferAmount > amount {
		newTxnOutputs = append(newTxnOutputs, createTxnOutput(transferAmount-amount, fromAddress))
	}

	newTransaction := Transaction{[]byte{}, newTxnInputs, newTxnOutputs}
	newTransaction.Sign(privKey)
	newTransaction.SetHash()
	err := blockchain.AddBlock([]*Transaction{&newTransaction})

	return err
}
