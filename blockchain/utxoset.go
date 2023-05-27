package blockchain

import "github.com/syndtr/goleveldb/leveldb"

type UTXOSet struct {
	database *leveldb.DB
}

func (utxoSet *UTXOSet) FindSpendableOutput(pubkeyHash []byte, amount int) (int, []TxOutput) {

}

func (utxoSet *UTXOSet) FindUTXO(pubkeyHash []byte) []TxOutput {

}

func (uxtoSet *UTXOSet) ReIndex() {

}

func (utxoSet *UTXOSet) Update(transaction *Transaction) {
	
}
