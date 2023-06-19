package blockchain

import "bytes"

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
			if i == len(currentHashList) - 1 {
				nextHashList = append(nextHashList, getDoubleSHA256(append(currentHashList[i], currentHashList[i]...)))
			} else {
				nextHashList = append(nextHashList, getDoubleSHA256(append(currentHashList[i], currentHashList[i+1]...)))
			}
		}
		currentHashList = nextHashList
	}
}

func (block *Block) GetMerklePath(targetTransaction *Transaction) [][]byte {
	currentHashList := [][]byte{}
	nextHashList := [][]byte{}
	targetHash := getDoubleSHA256(serialize(targetTransaction))
	merklePath := [][]byte{}

	for _, transaction := range block.Transactions {
		currentHashList = append(currentHashList, getDoubleSHA256(serialize(transaction)))
	}

	for {
		if len(currentHashList) == 1 {
			break
		}
		for i := 0; i < len(currentHashList); i += 2 {
			if i == len(currentHashList) - 1 {
				nextHash := getDoubleSHA256(append(currentHashList[i], currentHashList[i]...))
				nextHashList = append(nextHashList, nextHash)
				if bytes.Equal(targetHash, currentHashList[i]) {
					merklePath = append(merklePath, currentHashList[i])
					targetHash = nextHash
				}
			} else {
				nextHash := getDoubleSHA256(append(currentHashList[i], currentHashList[i+1]...))
				nextHashList = append(nextHashList, nextHash)
				if bytes.Equal(targetHash, currentHashList[i]) {
					merklePath = append(merklePath, currentHashList[i+1])
					targetHash = nextHash
				} else if bytes.Equal(targetHash, currentHashList[i+1]) {
					merklePath = append(merklePath, currentHashList[i])
					targetHash = nextHash
				}
			}
		}
		currentHashList = nextHashList
	}

	return merklePath
}
