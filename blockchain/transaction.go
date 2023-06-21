package blockchain

import (
	"bytes"
	"crypto/sha256"
)

type Transaction struct {
	Hash     []byte
	Inputs   []TxInput
	Outputs  []TxOutput
	Locktime int64
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
	Value        int
	ScriptPubKey LockingScript
}

func createTxnOutput(amount int, address string) TxOutput {
	lockingScript := LockingScript{getPubkeyHashFromAddress(address)}
	return TxOutput{amount, lockingScript}
}

func (txnInput TxInput) Hash() []byte {
	txnInput.ScriptSig.Signature = []byte{}
	hash := sha256.Sum256(serialize(txnInput))
	return hash[:]
}

func CoinBaseTransaction(toAddress string) *Transaction {
	txOutput := createTxnOutput(COINBASE_REWARD, toAddress)
	transaction := Transaction{
		Inputs:   []TxInput{},
		Outputs:  []TxOutput{txOutput},
		Locktime: getCurrentTimeInMilliSec(),
	}
	transaction.SetHash()
	return &transaction
}

func (transaction *Transaction) SetHash() {
	transaction.Hash = []byte{}
	txHash := sha256.Sum256(serialize(transaction))
	transaction.Hash = txHash[:]
}

func (txInput *TxInput) IsSignedBy(address string) bool {
	return bytes.Equal(getPubkeyHashFromPubkey(txInput.ScriptSig.PubKey), getPubkeyHashFromAddress(address))
}

func (txOutput *TxOutput) IsBoundTo(address string) bool {
	return bytes.Equal(txOutput.ScriptPubKey.PubKeyHash, getPubkeyHashFromAddress(address))
}
