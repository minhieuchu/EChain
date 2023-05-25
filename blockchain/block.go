package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"math/big"
	"time"
)

type Block struct {
	Transactions []*Transaction
	Timestamp    string
	PrevHash     []byte
	Hash         []byte
	Nonce        int
}

func (block *Block) Mine() {
	nonce := 1
	for {
		encodedBlock := bytes.Join(
			[][]byte{
				Encode(block.Transactions),
				[]byte(block.Timestamp),
				block.PrevHash,
				Encode(nonce),
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
	coinbaseTransaction := CoinBaseTransaction(NODE_ADDRESS)
	block := Block{
		Transactions: []*Transaction{coinbaseTransaction},
		Timestamp:    time.Now().String(),
		PrevHash:     []byte{},
	}
	block.Mine()
	return &block
}

func DecodeBlock(input []byte) *Block {
	var block Block
	byteBuffer := bytes.NewBuffer(input)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&block)
	return &block
}
