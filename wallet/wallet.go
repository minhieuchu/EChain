package wallet

import (
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

func (wallet *Wallet) Address() string {
	pubkeyHash := wallet.PubKeyHash()
	versionedHash := append([]byte{versionByte}, pubkeyHash...)
	
	firstHash := sha256.Sum256(versionedHash)
	secondHash := sha256.Sum256(firstHash[:])
	checksum := secondHash[:checksumLength]

	encoded := base58.Encode(append(versionedHash, checksum...))
	return encoded
}
