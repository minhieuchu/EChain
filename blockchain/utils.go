package blockchain

import (
	"bytes"
	"encoding/gob"
	"math/big"
)

const hashValueLength = 256 // bits
const difficultyLevel = 12  // bits

const COINBASE_REWARD = 1000 // satoshi
var TARGET_HASH = new(big.Int).Lsh(big.NewInt(1), hashValueLength-difficultyLevel)

func Encode(value interface{}) []byte {
	var byteBuffer bytes.Buffer
	encoder := gob.NewEncoder(&byteBuffer)
	encoder.Encode(value)
	return byteBuffer.Bytes()
}