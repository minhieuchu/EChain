package wallet

import (
	"EChain/blockchain"
	"EChain/network"
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
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

func (wallets *Wallets) GetBalance(walletAddress string) int {
	accountBalance := 0
	utxoMap, err := wallets.getUTXOs(walletAddress)
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}
	for _, txOutputs := range utxoMap {
		for _, output := range txOutputs {
			accountBalance += output.Amount
		}
	}
	return accountBalance
}

func (wallets *Wallets) getUTXOs(walletAddress string) (map[string]blockchain.TxOutputs, error) {
	getUTXOMsg := network.GetUTXOMessage{TargetAddress: walletAddress}
	sentData := append(msgTypeToBytes(network.GETUTXO_MSG), serialize(getUTXOMsg)...)

	successFlag := make(chan bool, len(wallets.connectedNodes))
	resultChan := make(chan map[string]blockchain.TxOutputs, len(wallets.connectedNodes))

	for _, nodeAddr := range wallets.connectedNodes {
		go func(targetAddress string) {
			conn, err := net.DialTimeout(protocol, targetAddress, time.Second)
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
			successFlag <- true
			resultChan <- utxoMap
		}(nodeAddr)
	}

	var utxoMap map[string]blockchain.TxOutputs
	var loopCounter int
	var didRequestSucceed bool

OuterLoop:
	for {
		time.Sleep(200 * time.Millisecond)
		select {
		case <-successFlag:
			utxoMap = <-resultChan
			didRequestSucceed = true
			break OuterLoop
		default:
			loopCounter++
		}
		if loopCounter > 10 {
			break OuterLoop
		}
	}

	if didRequestSucceed {
		return utxoMap, nil
	}

	return nil, fmt.Errorf("can not query UTXOs for %s", walletAddress)
}
