package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublickKey ecdsa.PublicKey
}

func CreateWallet() *Wallet {
	curve := elliptic.P256()
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	HandleError(err)

	pubKey := privKey.PublicKey

	return &Wallet{*privKey, pubKey}
}
