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

var WALLET_ADDRESS string

func InitBlockChain(networkAddress, walletAddress string) *BlockChain {
	WALLET_ADDRESS = walletAddress
	db, err := leveldb.OpenFile("storage/"+networkAddress, nil)
	if err != nil {
		log.Fatal(err)
	}

	genesisBlock := Genesis()
	blockchain := BlockChain{db, genesisBlock.Hash}
	blockchain.StoreNewBlock(genesisBlock)

	utxoSet := blockchain.UTXOSet()
	utxoSet.ReIndex()

	return &blockchain
}

func (chainIterator *BlockChainIterator) CurrentBlock() *Block {
	encodedBlock, err := chainIterator.DataBase.Get(chainIterator.CurrentHash, nil)
	handleErr(err)
	return DeserializeBlock(encodedBlock)
}

func (blockchain *BlockChain) GetHeight() int {
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}
	height := 0
	for {
		currentBlock := chainIterator.CurrentBlock()
		height++
		if len(currentBlock.PrevHash) == 0 {
			break
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}
	return height
}

func (blockchain *BlockChain) UTXOSet() UTXOSet {
	return UTXOSet{blockchain.DataBase}
}

func (blockchain *BlockChain) SetBlock(block *Block) {
	blockchain.DataBase.Put(block.Hash, serialize(block), nil)
}

func (blockchain *BlockChain) SetLastHash(hash []byte) {
	blockchain.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), hash, nil)
}

func (blockchain *BlockChain) StoreNewBlock(block *Block) {
	blockchain.LastHash = block.Hash
	blockchain.DataBase.Put(block.Hash, serialize(block), nil)
	blockchain.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), block.Hash, nil)

	utxoSet := blockchain.UTXOSet()
	utxoSet.Update(block)
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
	coinbaseTransaction := CoinBaseTransaction(WALLET_ADDRESS)
	newBlock := Block{
		Transactions: append([]*Transaction{coinbaseTransaction}, transactions...),
		Timestamp:    time.Now().String(),
		PrevHash:     blockchain.LastHash,
	}
	newBlock.Mine()
	blockchain.StoreNewBlock(&newBlock)
	return nil
}

func (blockchain *BlockChain) GetBalance(address string) int {
	balance := 0
	utxoSet := blockchain.UTXOSet()
	unspentTransactionOutputs := utxoSet.FindUTXO(address)

	for _, txnOutputs := range unspentTransactionOutputs {
		for _, output := range txnOutputs {
			balance += output.Amount
		}
	}
	return balance
}

func (blockchain *BlockChain) Transfer(privKey ecdsa.PrivateKey, pubKey []byte, toAddress string, amount int) error {
	fromAddress := getAddressFromPubkey(pubKey)
	utxoSet := blockchain.UTXOSet()
	transferAmount, unspentTxnOutputs := utxoSet.FindSpendableOutput(fromAddress, amount)
	if transferAmount < amount {
		return errors.New("not enough balance to transfer")
	}

	newTxnInputs := []TxInput{}
	newTxnOutputs := []TxOutput{}

	for txnID, txnOutputs := range unspentTxnOutputs {
		for _, output := range txnOutputs {
			newTxnInputs = append(newTxnInputs, createTxnInput([]byte(txnID), output.Index, pubKey))
		}
	}

	newTxnOutputs = append(newTxnOutputs, createTxnOutput(amount, toAddress))
	if transferAmount > amount {
		newTxnOutputs = append(newTxnOutputs, createTxnOutput(transferAmount-amount, fromAddress))
	}

	newTransaction := Transaction{[]byte{}, newTxnInputs, newTxnOutputs, getCurrentTimeInMilliSec()}
	newTransaction.Sign(privKey)
	newTransaction.SetHash()
	err := blockchain.AddBlock([]*Transaction{&newTransaction})

	return err
}

func (blockchain *BlockChain) GetUnmatchedBlocks(targetBlockHash []byte) (bool, [][]byte) {
	blockExisted := false
	unmatchedBlocks := make([][]byte, 0)
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}

	for {
		currentBlock := chainIterator.CurrentBlock()

		if slices.Equal(currentBlock.Hash, targetBlockHash) {
			blockExisted = true
			break
		}

		if len(currentBlock.PrevHash) == 0 {
			break
		} else {
			unmatchedBlocks = append(unmatchedBlocks, currentBlock.Hash)
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}

	return blockExisted, unmatchedBlocks
}

func (blockchain *BlockChain) GetBlocksFromHashes(hashList [][]byte) []*Block {
	blockList := []*Block{}
	for _, blockHash := range hashList {
		encodedBlock, _ := blockchain.DataBase.Get(blockHash, nil)
		blockList = append(blockList, DeserializeBlock(encodedBlock))
	}
	return blockList
}
