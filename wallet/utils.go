package wallet

import (
	"bytes"
	"encoding/gob"
	"log"
)

func handleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func msgTypeToBytes(msgType string) []byte {
	var res [msgTypeLength]byte
	for i := 0; i < len(msgType); i++ {
		res[i] = msgType[i]
	}
	return res[:]
}

func serialize(value interface{}) []byte {
	var byteBuffer bytes.Buffer
	encoder := gob.NewEncoder(&byteBuffer)
	encoder.Encode(value)
	return byteBuffer.Bytes()
}

func genericDeserialize[T any] (data []byte, target *T) {
	byteBuffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(target)
}
