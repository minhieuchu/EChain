package wallet

import (
	"EChain/blockchain"
	"bytes"
	"encoding/gob"
	"log"
	"time"

	"github.com/btcsuite/btcutil/base58"
)

const (
	pubKeyChecksumLength = 4
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

func genericDeserialize[T any](data []byte, target *T) {
	byteBuffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(target)
}

func getPubkeyHashFromAddress(address string) []byte {
	decoded := base58.Decode(address)
	return decoded[1:(len(decoded) - pubKeyChecksumLength)] // Exclude the first byte - version byte
}

func createTxnInput(txnID []byte, vOut int, pubkey []byte) blockchain.TxInput {
	unlockingScript := blockchain.UnlockingScript{Signature: []byte{}, PubKey: pubkey}
	return blockchain.TxInput{
		TxID:      txnID,
		VOut:      vOut,
		ScriptSig: unlockingScript,
	}
}

func createTxnOutput(amount int, address string) blockchain.TxOutput {
	lockingScript := blockchain.LockingScript{PubKeyHash: getPubkeyHashFromAddress(address)}
	return blockchain.TxOutput{Value: amount, ScriptPubKey: lockingScript}
}

func getCurrentTimeInMilliSec() int64 {
	return time.Now().UnixMilli()
}
