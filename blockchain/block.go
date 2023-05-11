package blockchain

type Block struct {
	Transactions []*Transaction
	Timestamp    string
	PrevHash     []byte
	Hash         []byte
	Nonce        int
}
