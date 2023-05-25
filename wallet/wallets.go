package wallet

import (
	"EChain/blockchain"

	"crypto/elliptic"
	"encoding/json"
	"os"
)

const walletFilePath = "wallets.json"

type Wallets struct {
	wallets map[string]Wallet
}

func (wallets *Wallets) GetWallet(address string) Wallet {
	return wallets.wallets[address]
}

func (wallets *Wallets) GetAddresses() []string {
	addresses := []string{}
	for key := range wallets.wallets {
		addresses = append(addresses, key)
	}
	return addresses
}

func (wallets *Wallets) AddNewWallet() string {
	newWallet := createWallet()
	walletAddress := newWallet.Address()
	wallets.wallets[walletAddress] = *newWallet
	return walletAddress
}

func LoadWallets() *Wallets {
	if _, err := os.Stat(walletFilePath); os.IsNotExist(err) {
		f, _ := os.Create(walletFilePath)
		defer f.Close()

		return &Wallets{make(map[string]Wallet)}
	}
	jsonStr, err := os.ReadFile(walletFilePath)
	HandleError(err)

	wallets := make(map[string]Wallet)
	json.Unmarshal(jsonStr, &wallets)
	for key, wallet := range wallets {
		wallet.PrivateKey.Curve = elliptic.P256()
		wallets[key] = wallet
	}

	return &Wallets{wallets}
}

func (wallets *Wallets) SaveFile() {
	jsonStr, _ := json.Marshal(wallets.wallets)
	err := os.WriteFile(walletFilePath, jsonStr, 0644)
	HandleError(err)
}

func (wallets *Wallets) Transfer(fromAddress, toAddress string, amount int, chain *blockchain.BlockChain) error {
	senderWallet := wallets.GetWallet(fromAddress)
	err := chain.Transfer(senderWallet.PrivateKey, senderWallet.PublickKey, fromAddress, toAddress, amount)
	return err
}
