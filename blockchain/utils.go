package blockchain

import (
	"bytes"
	"encoding/gob"
)

const COINBASE_REWARD = 1000 // satoshi

func Encode (value interface{}) []byte {
	var byteBuffer bytes.Buffer
	encoder := gob.NewEncoder(&byteBuffer)
	encoder.Encode(value)
	return byteBuffer.Bytes()
}
