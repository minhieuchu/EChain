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

type BlockChainHeader struct {
	DataBase *leveldb.DB
	LastHash []byte
}

type BlockChainIterator struct {
	DataBase    *leveldb.DB
	CurrentHash []byte
}

func InitBlockChain(networkAddress string) *BlockChain {
	db, err := leveldb.OpenFile("storage/"+networkAddress, nil)
	if err != nil {
		log.Fatal(err)
	}

	genesisBlock := GenerateGenesisBlock()
	blockchain := BlockChain{db, genesisBlock.GetHash()}
	blockchain.StoreNewBlock(genesisBlock)

	utxoSet := blockchain.UTXOSet()
	utxoSet.ReIndex()

	return &blockchain
}

func InitBlockChainHeader(database *leveldb.DB) *BlockChainHeader {
	blockchainHeader := BlockChainHeader{
		DataBase: database,
	}
	genesisBlock := GenerateGenesisBlock()
	blockchainHeader.LastHash = genesisBlock.GetHash()
	blockchainHeader.DataBase.Put(blockchainHeader.LastHash, serialize(genesisBlock.BlockHeader), nil)
	blockchainHeader.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), blockchainHeader.LastHash, nil)
	return &blockchainHeader
}

func (chainIterator *BlockChainIterator) CurrentBlock() *Block {
	encodedBlock, _ := chainIterator.DataBase.Get(chainIterator.CurrentHash, nil)
	return DeserializeBlock(encodedBlock)
}

func (blockchainHeader *BlockChainHeader) GetHeight() int {
	currentHash := blockchainHeader.LastHash
	height := 0
	for {
		encodedData, _ := blockchainHeader.DataBase.Get(currentHash, nil)
		var header BlockHeader
		genericDeserialize(encodedData, &header)
		height++
		if len(header.PrevHash) == 0 {
			break
		}
		currentHash = header.PrevHash
	}
	return height
}

func (blockchainHeader *BlockChainHeader) SetHeader(header *BlockHeader) {
	blockchainHeader.DataBase.Put(header.GetHash(), serialize(header), nil)
}

func (blockchainHeader *BlockChainHeader) SetLastHash(lastHash []byte) {
	blockchainHeader.LastHash = lastHash
	blockchainHeader.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), lastHash, nil)
}

func (blockchainHeader *BlockChainHeader) GetUnmatchedHeaders(targetHeaderHash []byte) (bool, []*BlockHeader) {
	headerExisted := false
	unmatchedHeaders := make([]*BlockHeader, 0)
	currentHash := blockchainHeader.LastHash

	for {
		encodedData, _ := blockchainHeader.DataBase.Get(currentHash, nil)
		var currentHeader BlockHeader
		genericDeserialize(encodedData, &currentHeader)

		if slices.Equal(currentHash, targetHeaderHash) {
			headerExisted = true
			break
		}

		if len(currentHeader.PrevHash) == 0 {
			break
		} else {
			unmatchedHeaders = append(unmatchedHeaders, &currentHeader)
		}
		currentHash = currentHeader.PrevHash
	}

	return headerExisted, unmatchedHeaders
}

func (blockchainHeader *BlockChainHeader) CheckHeaderExistence(header *BlockHeader) bool {
	headerHash, err := blockchainHeader.DataBase.Get(header.GetHash(), nil)
	var blockHeader BlockHeader
	genericDeserialize(headerHash, &blockHeader)
	if err != nil || blockHeader.Timestamp == "" {
		return false
	}
	return true
}

func (blockchain *BlockChain) GetHeight() int {
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}
	blockHeight := 0
	for {
		currentBlock := chainIterator.CurrentBlock()
		blockHeight++
		if len(currentBlock.PrevHash) == 0 {
			break
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}
	return blockHeight
}

func (blockchain *BlockChain) UTXOSet() UTXOSet {
	return UTXOSet{blockchain.DataBase}
}

func (blockchain *BlockChain) SetBlock(block *Block) {
	blockchain.DataBase.Put(block.GetHash(), serialize(block), nil)
}

func (blockchain *BlockChain) SetLastHash(hash []byte) {
	blockchain.LastHash = hash
	blockchain.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), hash, nil)
}

func (blockchain *BlockChain) StoreNewBlock(block *Block) {
	blockchain.LastHash = block.GetHash()
	blockchain.DataBase.Put(blockchain.LastHash, serialize(block), nil)
	blockchain.DataBase.Put([]byte(LAST_HASH_STOGAGE_KEY), blockchain.LastHash, nil)

	utxoSet := blockchain.UTXOSet()
	utxoSet.UpdateWithNewBlock(block)
}

func (blockchain *BlockChain) GetTransactionMapFromInputs(transaction *Transaction) map[string]Transaction {
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

func (blockchain *BlockChain) GetUTXOs(address string) map[string]TxOutputs {
	utxoSet := blockchain.UTXOSet()
	unspentTransactionOutputs := utxoSet.FindUTXO(address)
	return unspentTransactionOutputs
}

func (blockchain *BlockChain) GetUnmatchedBlocks(targetBlockHash []byte) (bool, [][]byte) {
	blockExisted := false
	unmatchedBlocks := make([][]byte, 0)
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}

	for {
		currentBlock := chainIterator.CurrentBlock()

		if slices.Equal(currentBlock.GetHash(), targetBlockHash) {
			blockExisted = true
			break
		}

		if len(currentBlock.PrevHash) == 0 {
			break
		} else {
			unmatchedBlocks = append(unmatchedBlocks, currentBlock.GetHash())
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}

	return blockExisted, unmatchedBlocks
}

func (blockchain *BlockChain) GetUnmatchedHeaders(targetHeaderHash []byte) (bool, []*BlockHeader) {
	headerExisted := false
	unmatchedHeaders := make([]*BlockHeader, 0)
	chainIterator := BlockChainIterator{blockchain.DataBase, blockchain.LastHash}

	for {
		currentBlock := chainIterator.CurrentBlock()

		if slices.Equal(currentBlock.GetHash(), targetHeaderHash) {
			headerExisted = true
			break
		}

		if len(currentBlock.PrevHash) == 0 {
			break
		} else {
			unmatchedHeaders = append(unmatchedHeaders, &currentBlock.BlockHeader)
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}

	return headerExisted, unmatchedHeaders
}

func (blockchain *BlockChain) GetBlocksFromHashes(hashList [][]byte) []*Block {
	blockList := []*Block{}
	for _, blockHash := range hashList {
		encodedBlock, _ := blockchain.DataBase.Get(blockHash, nil)
		blockList = append(blockList, DeserializeBlock(encodedBlock))
	}
	return blockList
}
