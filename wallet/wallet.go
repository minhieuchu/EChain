package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
)

const (
	pubKeyConstantPrefix = byte(4) // For uncompressed public key
	versionByte          = byte(0) // version byte prefixed to public key hash when calculating address
	checksumLength       = 4       // length of checksum embedded in address
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublickKey []byte
}

func createWallet() *Wallet {
	curve := elliptic.P256()
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	HandleError(err)

	pubKey := privKey.PublicKey
	uncompressedPubKey := append(pubKey.X.Bytes(), pubKey.Y.Bytes()...)
	uncompressedPubKey = append([]byte{pubKeyConstantPrefix}, uncompressedPubKey...)

	return &Wallet{*privKey, uncompressedPubKey}
}

func (wallet *Wallet) PubKeyHash() []byte {
	sha256Hash := sha256.Sum256(wallet.PublickKey)
	hasher := ripemd160.New()
	hasher.Write(sha256Hash[:])
	return hasher.Sum(nil)
}

func getChecksum(versionedHash []byte) []byte {
	firstHash := sha256.Sum256(versionedHash)
	secondHash := sha256.Sum256(firstHash[:])
	checksum := secondHash[:checksumLength]
	return checksum
}

func (wallet *Wallet) Address() string {
	pubkeyHash := wallet.PubKeyHash()
	versionedHash := append([]byte{versionByte}, pubkeyHash...)

	encoded := base58.Encode(append(versionedHash, getChecksum(versionedHash)...))
	return encoded
}

func IsAddressValid(address string) bool {
	decoded := base58.Decode(address)
	version := decoded[:1]
	if !bytes.Equal(version, []byte{versionByte}) {
		return false
	}
	payloadLastIndex := len(decoded) - checksumLength
	payload := decoded[:payloadLastIndex]
	checksum := decoded[payloadLastIndex:]
	return bytes.Equal(getChecksum(payload), checksum)
}
