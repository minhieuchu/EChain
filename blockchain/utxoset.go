package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/exp/slices"
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

func (utxoSet *UTXOSet) FindSpendableOutput(address string, amount int) (int, map[string]TxOutputs) {
	accumulatedAmount := 0
	utxoMap := make(map[string]TxOutputs)
	iter := utxoSet.database.NewIterator(util.BytesPrefix(utxoPrefix), nil)

OuterLoop:
	for iter.Next() {
		txnID := iter.Key()[utxoPrefixLength:]
		txnOutputs := deserializeTxnOutputs(iter.Value())

		for _, txnOutput := range txnOutputs {
			if txnOutput.IsBoundTo(address) {
				accumulatedAmount += txnOutput.Value
				utxoMap[string(txnID)] = append(utxoMap[string(txnID)], txnOutput)
			}
			if accumulatedAmount >= amount {
				break OuterLoop
			}
		}
	}
	iter.Release()

	return accumulatedAmount, utxoMap
}

func (utxoSet *UTXOSet) FindUTXO(address string) map[string]TxOutputs {
	utxoMap := make(map[string]TxOutputs)
	iter := utxoSet.database.NewIterator(util.BytesPrefix(utxoPrefix), nil)

	for iter.Next() {
		txnID := iter.Key()[utxoPrefixLength:]
		txnOutputs := deserializeTxnOutputs(iter.Value())

		for _, txnOutput := range txnOutputs {
			if txnOutput.IsBoundTo(address) {
				utxoMap[string(txnID)] = append(utxoMap[string(txnID)], txnOutput)
			}
		}
	}
	iter.Release()

	return utxoMap
}

func (utxoSet *UTXOSet) Update(newBlock *Block) {
	spentTxnOutputs := make(map[string][]int)
	batch := new(leveldb.Batch)

	for _, transaction := range newBlock.Transactions {
		var txOutputs TxOutputs
		for _, txnInput := range transaction.Inputs {
			spentTxnOutputs[string(txnInput.TxID)] = append(spentTxnOutputs[string(txnInput.TxID)], txnInput.VOut)
		}

		for outputIndex, txOutput := range transaction.Outputs {
			txOutputs = append(txOutputs, TxOutputWithIndex{txOutput, outputIndex})
		}

		utxoSetTxnID := append(utxoPrefix, transaction.Hash...)
		batch.Put(utxoSetTxnID, serialize(txOutputs))
	}

	for txnID, spentTxnOutputIDs := range spentTxnOutputs {
		utxoSetTxnID := append(utxoPrefix, []byte(txnID)...)
		encodedTxnOutputs, _ := utxoSet.database.Get(utxoSetTxnID, nil)
		currentTxnOutputs := deserializeTxnOutputs(encodedTxnOutputs)

		var newTxOutputs TxOutputs
		for _, unspentTxOutput := range currentTxnOutputs {
			if !slices.Contains(spentTxnOutputIDs, unspentTxOutput.Index) {
				newTxOutputs = append(newTxOutputs, unspentTxOutput)
			}
		}

		if len(newTxOutputs) > 0 {
			batch.Put(utxoSetTxnID, serialize(newTxOutputs))
		} else {
			batch.Delete(utxoSetTxnID)
		}
	}
	utxoSet.database.Write(batch, nil)
}

func (utxoSet *UTXOSet) GetTxOutputFromTxInput(txnInput *TxInput) *TxOutput {
	referencedTxnID := txnInput.TxID
	utxoSetTxnID := append(utxoPrefix, referencedTxnID...)
	encodedTxnOutputs, _ := utxoSet.database.Get(utxoSetTxnID, nil)
	currentTxnOutputs := deserializeTxnOutputs(encodedTxnOutputs)
	for _, txOutput := range currentTxnOutputs {
		if txOutput.Index == txnInput.VOut {
			return &txOutput.TxOutput
		}
	}
	return nil
}

func (utxoSet *UTXOSet) ReIndex() {
	// ===== Batch delete existing UTXO set =====
	batch := new(leveldb.Batch)
	iter := utxoSet.database.NewIterator(util.BytesPrefix(utxoPrefix), nil)

	for iter.Next() {
		utxoKey := iter.Key()
		batch.Delete(utxoKey)
	}
	iter.Release()
	err := utxoSet.database.Write(batch, nil)
	handleErr(err)

	// ===== Traverse the blockchain to create new UTXO set
	lastHash, _ := utxoSet.database.Get([]byte(LAST_HASH_STOGAGE_KEY), nil)
	chainIterator := BlockChainIterator{utxoSet.database, lastHash}
	spentTxnOutputs := make(map[string][]int)

	for {
		currentBlock := chainIterator.CurrentBlock()
		for _, transaction := range currentBlock.Transactions {
			for _, txnInput := range transaction.Inputs {
				spentTxnOutputs[string(txnInput.TxID)] = append(spentTxnOutputs[string(txnInput.TxID)], txnInput.VOut)
			}

			var txnOutputs TxOutputs
			for outputIndex, txnOutput := range transaction.Outputs {
				if !slices.Contains(spentTxnOutputs[string(transaction.Hash)], outputIndex) {
					txnOutputWithIndex := TxOutputWithIndex{txnOutput, outputIndex}
					txnOutputs = append(txnOutputs, txnOutputWithIndex)
				}
			}

			utxoSetTxnID := append(utxoPrefix, transaction.Hash...)
			utxoSet.database.Put(utxoSetTxnID, serialize(txnOutputs), nil)
		}

		if len(currentBlock.PrevHash) == 0 {
			break
		}
		chainIterator.CurrentHash = currentBlock.PrevHash
	}
}

func deserializeTxnOutputs(outputs []byte) TxOutputs {
	var txnOutputs TxOutputs
	byteBuffer := bytes.NewBuffer(outputs)
	decoder := gob.NewDecoder(byteBuffer)
	decoder.Decode(&txnOutputs)
	return txnOutputs
}

func (utxoSet *UTXOSet) print() {
	iter := utxoSet.database.NewIterator(util.BytesPrefix(utxoPrefix), nil)

	fmt.Println("===== Start Logging UTXO Set =====")
	for iter.Next() {
		txnOutputs := deserializeTxnOutputs(iter.Value())
		fmt.Println("TxnID: ", iter.Key()[utxoPrefixLength:])
		for _, output := range txnOutputs {
			fmt.Println("UTXO: ")
			spew.Dump(output)
		}
	}
	fmt.Println("===== End Logging UTXO Set =====")
	fmt.Println("")
	iter.Release()
}
