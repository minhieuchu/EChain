package network

import (
	"EChain/blockchain"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"

	"golang.org/x/crypto/ripemd160"
)

func getPubkeyHashFromPubkey(pubkey []byte) []byte {
	sha256Hash := sha256.Sum256(pubkey)
	hasher := ripemd160.New()
	hasher.Write(sha256Hash[:])
	return hasher.Sum(nil)
}

func getECDSAPubkeyFromUncompressedPubkey(uncompressedPubKey []byte) ecdsa.PublicKey {
	pubkeyCoordinates := uncompressedPubKey[1:] // remove the 1st byte prefix for uncompressed version
	pubkeyLength := len(pubkeyCoordinates)
	x := pubkeyCoordinates[:(pubkeyLength / 2)]
	y := pubkeyCoordinates[(pubkeyLength / 2):]
	bigIntX := new(big.Int).SetBytes(x)
	bigIntY := new(big.Int).SetBytes(y)
	curve := elliptic.P256()
	ecdsaPubkey := ecdsa.PublicKey{Curve: curve, X: bigIntX, Y: bigIntY}
	return ecdsaPubkey
}

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

func isTransactionOfInterest(transaction blockchain.Transaction, bloomFilter []string) bool {
	for _, targetAddr := range bloomFilter {
		for _, input := range transaction.Inputs {
			if input.IsSignedBy(targetAddr) {
				return true
			}
		}
		for _, output := range transaction.Outputs {
			if output.IsBoundTo(targetAddr) {
				return true
			}
		}
	}
	return false
}

func genericDeserialize[T any](data []byte, target *T) {
	byteBuffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(target)
}

func handleError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
