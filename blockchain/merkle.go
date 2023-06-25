package blockchain

import "bytes"

type MerkleProofNode struct {
	Hash       []byte
	IsLeftNode bool
}

func (block *Block) SetMerkleRoot() {
	currentHashList := [][]byte{}
	nextHashList := [][]byte{}

	for _, transaction := range block.Transactions {
		currentHashList = append(currentHashList, getDoubleSHA256(serialize(transaction)))
	}

	for {
		if len(currentHashList) == 1 {
			block.MerkleRoot = currentHashList[0]
			break
		}
		for i := 0; i < len(currentHashList); i += 2 {
			if i == len(currentHashList)-1 {
				nextHashList = append(nextHashList, getDoubleSHA256(append(currentHashList[i], currentHashList[i]...)))
			} else {
				nextHashList = append(nextHashList, getDoubleSHA256(append(currentHashList[i], currentHashList[i+1]...)))
			}
		}
		currentHashList = nextHashList
	}
}

func (block *Block) GetMerkleProof(targetTransaction *Transaction) []MerkleProofNode {
	currentHashList := [][]byte{}
	nextHashList := [][]byte{}
	targetHash := getDoubleSHA256(serialize(targetTransaction))
	merkleProof := []MerkleProofNode{}

	for _, transaction := range block.Transactions {
		currentHashList = append(currentHashList, getDoubleSHA256(serialize(transaction)))
	}

	for {
		if len(currentHashList) == 1 {
			break
		}
		for i := 0; i < len(currentHashList); i += 2 {
			if i == len(currentHashList)-1 {
				nextHash := getDoubleSHA256(append(currentHashList[i], currentHashList[i]...))
				nextHashList = append(nextHashList, nextHash)
				if bytes.Equal(targetHash, currentHashList[i]) {
					merkleProof = append(merkleProof, MerkleProofNode{currentHashList[i], true})
					targetHash = nextHash
				}
			} else {
				nextHash := getDoubleSHA256(append(currentHashList[i], currentHashList[i+1]...))
				nextHashList = append(nextHashList, nextHash)
				if bytes.Equal(targetHash, currentHashList[i]) {
					merkleProof = append(merkleProof, MerkleProofNode{currentHashList[i+1], false})
					targetHash = nextHash
				} else if bytes.Equal(targetHash, currentHashList[i+1]) {
					merkleProof = append(merkleProof, MerkleProofNode{currentHashList[i], true})
					targetHash = nextHash
				}
			}
		}
		currentHashList = nextHashList
	}

	return merkleProof
}
