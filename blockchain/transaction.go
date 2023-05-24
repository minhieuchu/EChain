package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
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

func (transaction *Transaction) Sign(privKey ecdsa.PrivateKey) {
	for inputIndex, txnInput := range transaction.Inputs {
		inputHash := sha256.Sum256(Encode(txnInput))
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, inputHash[:])
		HandleErr(err)
		signature := append(r.Bytes(), s.Bytes()...)
		transaction.Inputs[inputIndex].ScriptSig.Signature = signature
	}
}

func (transaction *Transaction) SetHash() {
	transaction.Hash = []byte{}
	txHash := sha256.Sum256(Encode(transaction))
	transaction.Hash = txHash[:]
}

func (txInput *TxInput) IsSignedBy(address string) bool {
	return bytes.Equal(getPubkeyHashFromPubkey(txInput.ScriptSig.PubKey), getPubkeyHashFromAddress(address))
}

func (txOutput *TxOutput) IsBoundTo(address string) bool {
	return bytes.Equal(txOutput.ScriptPubKey.PubKeyHash, getPubkeyHashFromAddress(address))
}
