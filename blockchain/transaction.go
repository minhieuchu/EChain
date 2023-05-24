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

type UnlockingScript struct {
	Signature []byte
	PubKey    []byte
}

type LockingScript struct {
	PubKeyHash []byte
}

type TxInput struct {
	TxID      []byte
	VOut      int
	ScriptSig UnlockingScript
}

type TxOutput struct {
	Amount       int
	ScriptPubKey LockingScript
}

type surplusTxOutput struct {
	TxOutput
	TxID []byte
	VOut int
}

func createTxnInput(txnID []byte, vOut int, pubkey []byte) TxInput {
	unlockingScript := UnlockingScript{[]byte{}, pubkey}
	return TxInput{txnID, vOut, unlockingScript}
}

func createTxnOutput(amount int, address string) TxOutput {
	lockingScript := LockingScript{getPubkeyHashFromAddress(address)}
	return TxOutput{amount, lockingScript}
}

func CoinBaseTransaction(toAddress string) *Transaction {
	txOutput := createTxnOutput(COINBASE_REWARD, toAddress)
	transaction := Transaction{
		Inputs:  []TxInput{},
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
	return bytes.Equal(getPubkeyHashFromPubkey(txInput.ScriptSig.PubKey), getPubkeyHashFromAddress(address))
}

func (txOutput *TxOutput) IsBoundTo(address string) bool {
	return bytes.Equal(txOutput.ScriptPubKey.PubKeyHash, getPubkeyHashFromAddress(address))
}
