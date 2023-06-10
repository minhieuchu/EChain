package wallet

import (
	"EChain/blockchain"
	"EChain/network"
	"crypto/elliptic"
	"encoding/json"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

const (
	protocol       = "tcp"
	walletFilePath = "wallets.json"
	msgTypeLength  = 12
)

var initialConnectedNodes = []string{"localhost:8333", "localhost:8334", "localhost:8335"}

type Wallets struct {
	connectedNodes []string
	wallets        map[string]Wallet
}

func (wallets *Wallets) ConnectNode(nodeAddress string) {
	wallets.connectedNodes = append(wallets.connectedNodes, nodeAddress)
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

	return &Wallets{initialConnectedNodes, wallets}
}

func (wallets *Wallets) SaveFile() {
	jsonStr, _ := json.Marshal(wallets.wallets)
	err := os.WriteFile(walletFilePath, jsonStr, 0644)
	handleError(err)
}

func (wallets *Wallets) Transfer(fromAddress, toAddress string, amount int) error {
	return nil
}

func (wallets *Wallets) GetBalance(address string) int {
	var wg sync.WaitGroup
	getUTXOMsg := network.GetUTXOMessage{TargetAddress: address}
	sentData := append(msgTypeToBytes(network.GETUTXO_MSG), serialize(getUTXOMsg)...)
	balanceChannel := make(chan int, len(wallets.connectedNodes))

	for _, nodeAddress := range wallets.connectedNodes {
		wg.Add(1)
		go func(targetAddress string) {
			go func() {
				time.Sleep(2 * time.Second)
				wg.Done()
			}()
			conn, err := net.Dial(protocol, targetAddress)
			if err != nil {
				return
			}
			conn.Write(sentData)
			conn.(*net.TCPConn).CloseWrite()

			resp, er := io.ReadAll(conn)
			if er != nil {
				return
			}
			defer conn.Close()
			var utxoMap map[string]blockchain.TxOutputs
			genericDeserialize(resp, &utxoMap)

			balance := 0
			for _, txOutputs := range utxoMap {
				for _, output := range txOutputs {
					balance += output.Amount
				}
			}
			balanceChannel <- balance
		}(nodeAddress)
	}
	wg.Wait()
	if len(balanceChannel) == 0 {
		return 0
	}
	accountBalance := <-balanceChannel
	return accountBalance
}
