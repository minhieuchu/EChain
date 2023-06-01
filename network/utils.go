package network

import (
	"bytes"
	"encoding/gob"
	"log"
)

func serialize(value interface{}) []byte {
	var byteBuffer bytes.Buffer
	encoder := gob.NewEncoder(&byteBuffer)
	encoder.Encode(value)
	return byteBuffer.Bytes()
}

func handleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
