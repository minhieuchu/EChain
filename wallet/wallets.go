package wallet

import (
	"EChain/blockchain"

	"crypto/elliptic"
	"encoding/json"
	"os"
)

const walletFilePath = "wallets.json"

type Wallets struct {
	connectedChain blockchain.BlockChain
	wallets        map[string]Wallet
}

func (wallets *Wallets) ConnectChain(chain *blockchain.BlockChain) {
	wallets.connectedChain = *chain
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

		newWallets := Wallets{}
		newWallets.wallets = make(map[string]Wallet)
		return &newWallets
	}
	jsonStr, err := os.ReadFile(walletFilePath)
	handleError(err)

	wallets := make(map[string]Wallet)
	json.Unmarshal(jsonStr, &wallets)
	for key, wallet := range wallets {
		wallet.PrivateKey.Curve = elliptic.P256()
		wallets[key] = wallet
	}

	return &Wallets{blockchain.BlockChain{}, wallets}
}

func (wallets *Wallets) SaveFile() {
	jsonStr, _ := json.Marshal(wallets.wallets)
	err := os.WriteFile(walletFilePath, jsonStr, 0644)
	handleError(err)
}

func (wallets *Wallets) Transfer(toAddress string, amount int) error {
	senderWallet := wallets.GetWallet(blockchain.WALLET_ADDRESS)
	err := wallets.connectedChain.Transfer(senderWallet.PrivateKey, senderWallet.PublickKey, toAddress, amount)
	return err
}

func (wallets *Wallets) GetBalance(address string) int {
	return wallets.connectedChain.GetBalance(address)
}
