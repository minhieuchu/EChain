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

type Block struct {
	Transactions []*Transaction
	Timestamp    string
	PrevHash     []byte
	Hash         []byte
	Nonce        int
	Height       int
}

func (block *Block) Mine() {
	nonce := 1
	for {
		encodedBlock := bytes.Join(
			[][]byte{
				serialize(block.Transactions),
				[]byte(block.Timestamp),
				block.PrevHash,
				serialize(nonce),
			},
			[]byte{},
		)
		blockHash := sha256.Sum256(encodedBlock)
		hashValue := new(big.Int).SetBytes(blockHash[:])

		if hashValue.Cmp(TARGET_HASH) == -1 {
			block.Nonce = nonce
			block.Hash = blockHash[:]
			break
		}
		nonce++
	}
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
		Transactions: []*Transaction{&coinbaseTransaction},
		Timestamp:    genesisBlockDate.String(),
		PrevHash:     []byte{},
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
