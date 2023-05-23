package wallet

import (
	"bytes"
	"encoding/gob"
	"log"
)

func HandleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func Encode(value interface{}) []byte {
	var byteBuffer bytes.Buffer
	encoder := gob.NewEncoder(&byteBuffer)
	encoder.Encode(value)
	return byteBuffer.Bytes()
}
