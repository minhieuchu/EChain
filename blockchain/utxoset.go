package blockchain

import (
	"bytes"
	"encoding/gob"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type UTXOSet struct {
	database *leveldb.DB
}

type TxOutputWithIndex struct {
	TxOutput
	Index int
}

type TxOutputs []TxOutputWithIndex

var (
	utxoPrefix       = []byte("utxo-")
	utxoPrefixLength = len(utxoPrefix)
)

func (utxoSet *UTXOSet) FindSpendableOutput(pubkeyHash []byte, amount int) (int, map[string][]int) {
	accumulatedAmount := 0
	utxoMap := make(map[string][]int)

	targetAddress := getAddressFromPubkeyHash(pubkeyHash)
	iter := utxoSet.database.NewIterator(util.BytesPrefix(utxoPrefix), nil)

OuterLoop:
	for iter.Next() {
		txnID := iter.Key()[utxoPrefixLength:]
		txnOutputs := DeserializeTxnOutputs(iter.Value())

		for _, txnOutput := range txnOutputs {
			if txnOutput.IsBoundTo(targetAddress) {
				accumulatedAmount += txnOutput.Amount
				utxoMap[string(txnID)] = append(utxoMap[string(txnID)], txnOutput.Index)
			}
			if accumulatedAmount >= amount {
				break OuterLoop
			}
		}
	}
	iter.Release()

	return accumulatedAmount, utxoMap
}

func (utxoSet *UTXOSet) FindUTXO(pubkeyHash []byte) map[string][]int {
	utxoMap := make(map[string][]int)

	targetAddress := getAddressFromPubkeyHash(pubkeyHash)
	iter := utxoSet.database.NewIterator(util.BytesPrefix(utxoPrefix), nil)

	for iter.Next() {
		txnID := iter.Key()[utxoPrefixLength:]
		txnOutputs := DeserializeTxnOutputs(iter.Value())

		for _, txnOutput := range txnOutputs {
			if txnOutput.IsBoundTo(targetAddress) {
				utxoMap[string(txnID)] = append(utxoMap[string(txnID)], txnOutput.Index)
			}
		}
	}
	iter.Release()

	return utxoMap
}

func (utxoSet *UTXOSet) Update(newBlock *Block) {

}

func (uxtoSet *UTXOSet) ReIndex() {

}

func DeserializeTxnOutputs(outputs []byte) TxOutputs {
	var txnOutputs TxOutputs
	byteBuffer := bytes.NewBuffer(outputs)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&txnOutputs)
	return txnOutputs
}
