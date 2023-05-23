package wallet

type Wallets struct {
	wallets map[string]Wallet
}

func (wallets *Wallets) GetWallet(address string) Wallet {
	return wallets.wallets[address]
}

func (wallets *Wallets) CreateWallet() string {
	newWallet := CreateWallet()
	walletAddress := newWallet.Address()
	wallets.wallets[walletAddress] = *newWallet
	return walletAddress
}
