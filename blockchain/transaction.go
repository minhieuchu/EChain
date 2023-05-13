package blockchain

import "crypto/sha256"

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

func CoinBaseTransaction(toAddress string) *Transaction {
	txOutput := TxOutput{COINBASE_REWARD, toAddress}
	transaction := Transaction{[]byte{}, []TxInput{}, []TxOutput{txOutput}}
	txHash := sha256.Sum256(Encode(transaction))
	transaction.Hash = txHash[:]
	return &transaction
}

func (txInput *TxInput) IsSignedBy(address string) bool {
	return txInput.UnlockScript == address
}

func (txOutput *TxOutput) CanBeUnlocked(address string) bool {
	return txOutput.LockScript == address
}
