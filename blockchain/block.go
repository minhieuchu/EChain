package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"math/big"
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

func (block *Block) Mine() {
	nonce := 1
	for {
		block.Nonce = nonce
		hashValue := new(big.Int).SetBytes(block.GetHash())

		if hashValue.Cmp(TARGET_HASH) == -1 {
			block.Nonce = nonce
			break
		}
		nonce++
	}
}

func (block *Block) GetHash() []byte {
	firstHash := sha256.Sum256(serialize(block.BlockHeader))
	secondHash := sha256.Sum256(firstHash[:])
	return secondHash[:]
}

func Genesis() *Block {
	err := godotenv.Load()
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
		},
		Transactions: []*Transaction{&coinbaseTransaction},
	}
	block.Mine()
	return &block
}

func DeserializeBlock(input []byte) *Block {
	var block Block
	byteBuffer := bytes.NewBuffer(input)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&block)
	return &block
}
