package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/big"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
)

const (
	hashValueLength       = 256 // bits
	difficultyLevel       = 12  // bits
	pubKeyChecksumLength  = 4
	versionByte           = byte(0) // prefixed to pubkey hash when calculating address
	COINBASE_REWARD       = 1000    // satoshi
	LAST_HASH_STOGAGE_KEY = "LAST_HASH"
)

var TARGET_HASH = new(big.Int).Lsh(big.NewInt(1), hashValueLength-difficultyLevel)

func getPubkeyHashFromPubkey(pubkey []byte) []byte {
	sha256Hash := sha256.Sum256(pubkey)
	hasher := ripemd160.New()
	hasher.Write(sha256Hash[:])
	return hasher.Sum(nil)
}

func getECDSAPubkeyFromUncompressedPubkey(uncompressedPubKey []byte) ecdsa.PublicKey {
	pubkeyCoordinates := uncompressedPubKey[1:] // remove the 1st byte prefix for uncompressed version
	pubkeyLength := len(pubkeyCoordinates)
	x := pubkeyCoordinates[:(pubkeyLength / 2)]
	y := pubkeyCoordinates[(pubkeyLength / 2):]
	bigIntX := new(big.Int).SetBytes(x)
	bigIntY := new(big.Int).SetBytes(y)
	curve := elliptic.P256()
	ecdsaPubkey := ecdsa.PublicKey{curve, bigIntX, bigIntY}
	return ecdsaPubkey
}

func getChecksum(versionedHash []byte) []byte {
	firstHash := sha256.Sum256(versionedHash)
	secondHash := sha256.Sum256(firstHash[:])
	checksum := secondHash[:pubKeyChecksumLength]
	return checksum
}

func getAddressFromPubkey(pubkey []byte) string {
	pubkeyHash := getPubkeyHashFromPubkey(pubkey)
	versionedHash := append([]byte{versionByte}, pubkeyHash...)

	encoded := base58.Encode(append(versionedHash, getChecksum(versionedHash)...))
	return encoded
}

func getPubkeyHashFromAddress(address string) []byte {
	decoded := base58.Decode(address)
	return decoded[1:(len(decoded) - pubKeyChecksumLength)] // Exclude the first byte - version byte
}

func Encode(value interface{}) []byte {
	var byteBuffer bytes.Buffer
	encoder := gob.NewEncoder(&byteBuffer)
	encoder.Encode(value)
	return byteBuffer.Bytes()
}

func HandleErr(err error) {
	if err != nil {
		log.Panic(err)
	}
}
