package blockchain

import (
	"bytes"
	"encoding/gob"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type BlockHeader struct {
	PrevHash   []byte
	MerkleRoot []byte
	Timestamp  string
	Nonce      int
}

type Block struct {
	BlockHeader
	Transactions []*Transaction
}

func (blockHeader *BlockHeader) GetHash() []byte {
	return getDoubleSHA256(serialize(blockHeader))
}

func (block *Block) GetHash() []byte {
	if len(block.MerkleRoot) == 0 {
		block.SetMerkleRoot()
	}
	return block.BlockHeader.GetHash()
}

func GenerateGenesisBlock() *Block {
	err := godotenv.Load("../.env")
	handleErr(err)

	satoshiAddress := os.Getenv("SATOSHI_ADDRESS")
	genesisBlockDate, _ := time.Parse("2006-Jan-02", "2009-Jan-03")
	txOutput := createTxnOutput(COINBASE_REWARD, satoshiAddress)
	coinbaseTransaction := Transaction{
		Inputs:   []TxInput{},
		Outputs:  []TxOutput{txOutput},
		Locktime: genesisBlockDate.UnixMilli(),
	}
	coinbaseTransaction.SetHash()

	block := Block{
		BlockHeader: BlockHeader{
			Timestamp: genesisBlockDate.String(),
			PrevHash:  []byte{},
			Nonce:     4436,
		},
		Transactions: []*Transaction{&coinbaseTransaction},
	}
	return &block
}

func DeserializeBlock(input []byte) *Block {
	var block Block
	byteBuffer := bytes.NewBuffer(input)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&block)
	return &block
}
