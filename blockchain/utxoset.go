package blockchain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type UTXOSet struct {
	database *leveldb.DB
}

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

		for outputIndex, txnOutput := range txnOutputs.Outputs {
			if txnOutput.IsBoundTo(targetAddress) {
				accumulatedAmount += txnOutput.Amount
				utxoMap[string(txnID)] = append(utxoMap[string(txnID)], outputIndex)
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

		for outputIndex, txnOutput := range txnOutputs.Outputs {
			if txnOutput.IsBoundTo(targetAddress) {
				utxoMap[string(txnID)] = append(utxoMap[string(txnID)], outputIndex)
			}
		}
	}

	return utxoMap
}

func (uxtoSet *UTXOSet) ReIndex() {

}

func (utxoSet *UTXOSet) Update(newBlock *Block) {

}
