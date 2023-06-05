package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
)

func sendMessageBlocking(toAddress string, msg []byte) {
	conn, err := net.Dial(protocol, toAddress)
	if err != nil {
		fmt.Println("can not connect to", toAddress)
		return
	}
	conn.Write(msg)
	conn.(*net.TCPConn).CloseWrite()
	io.ReadAll(conn)
	conn.Close()
}

func sendMessage(toAddress string, msg []byte) {
	conn, err := net.Dial(protocol, toAddress)
	if err != nil {
		fmt.Println("can not connect to", toAddress)
		return
	}
	conn.Write(msg)
	conn.(*net.TCPConn).CloseWrite()
	conn.Close()
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

func genericDeserialize[T any] (data []byte, target *T) {
	byteBuffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(target)
}

func handleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
