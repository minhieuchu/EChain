package blockchain

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
