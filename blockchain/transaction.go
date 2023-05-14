package blockchain

import (
	"bytes"
	"crypto/sha256"
)

type Transaction struct {
	Hash    []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

type TxInput struct {
	TxHash       []byte
	OutputIndex  int
	UnlockScript string
}

type TxOutput struct {
	Amount     int
	LockScript string
}

type surplusTxOutput struct {
	TxOutput
	TxHash      []byte
	OutputIndex int
}

func CoinBaseTransaction(toAddress string) *Transaction {
	txOutput := TxOutput{COINBASE_REWARD, toAddress}
	transaction := Transaction{
		Inputs: []TxInput{},
		Outputs: []TxOutput{txOutput},
	}
	transaction.SetHash()
	return &transaction
}

func (transaction *Transaction) SetHash() {
	encodedTransaction := bytes.Join([][]byte{Encode(transaction.Inputs), Encode(transaction.Outputs)}, []byte{})
	txHash := sha256.Sum256(encodedTransaction)
	transaction.Hash = txHash[:]
}

func (txInput *TxInput) IsSignedBy(address string) bool {
	return txInput.UnlockScript == address
}

func (txOutput *TxOutput) CanBeUnlocked(address string) bool {
	return txOutput.LockScript == address
}
