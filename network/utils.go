package network

import (
	"bytes"
	"encoding/gob"
	"log"
	"net"
)

func sendMessage(toAddress string, msg []byte) {
	conn, err := net.Dial(protocol, toAddress)
	handleError(err)
	conn.Write(msg)
}

func msgTypeToBytes(msgType string) []byte {
	var res [msgTypeLength]byte
	for i := 0; i < len(msgType); i++ {
		res[i] = msgType[i]
	}
	return res[:]
}

func getMsgType(msg []byte) string {
	 trimmedMsg := bytes.Trim(msg[:msgTypeLength], "\x00")
	 return string(trimmedMsg)
}

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
