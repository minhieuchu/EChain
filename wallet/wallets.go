package wallet

import (
	"EChain/blockchain"
	"EChain/network"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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

type Wallets struct {
	connectedNodes []network.NodeInfo
	wallets        map[string]Wallet
}

func (wallets *Wallets) ConnectNode(nodeType, address string) {
	wallets.connectedNodes = append(wallets.connectedNodes, network.NodeInfo{NodeType: nodeType, Address: address})
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

	return &Wallets{wallets: wallets}
}

func (wallets *Wallets) SaveFile() {
	jsonStr, _ := json.Marshal(wallets.wallets)
	err := os.WriteFile(walletFilePath, jsonStr, 0644)
	handleError(err)
}

func (wallets *Wallets) Transfer(fromAddress, toAddress string, amount int) error {
	senderWallet, existed := wallets.wallets[fromAddress]
	if !existed {
		return fmt.Errorf("wallet does not contain keys for address %s", fromAddress)
	}
	utxoMap, err := wallets.getUTXOs(fromAddress)
	if err != nil {
		return err
	}
	transferAmount := 0
	newTxnInputs := []blockchain.TxInput{}
	newTxnOutputs := []blockchain.TxOutput{}

OuterLoop:
	for txnID, txnOutputs := range utxoMap {
		for _, output := range txnOutputs {
			transferAmount += output.Value
			newTxnInputs = append(newTxnInputs, createTxnInput([]byte(txnID), output.Index, senderWallet.PublickKey))
			if transferAmount >= amount {
				break OuterLoop
			}
		}
	}

	if transferAmount < amount {
		return fmt.Errorf("not enough balance")
	}

	newTxnOutputs = append(newTxnOutputs, createTxnOutput(amount, toAddress))
	if transferAmount > amount {
		newTxnOutputs = append(newTxnOutputs, createTxnOutput(transferAmount-amount, fromAddress))
	}

	newTransaction := blockchain.Transaction{Inputs: newTxnInputs, Outputs: newTxnOutputs, Locktime: getCurrentTimeInMilliSec()}
	wallets.signTransaction(&newTransaction, senderWallet.PrivateKey)
	newTransaction.SetHash()

	sentData := append(msgTypeToBytes(network.NEWTXN_MSG), serialize(newTransaction)...)

	// Broadcast new transaction to network
	for _, connectedNode := range wallets.connectedNodes {
		go func(targetAddress string) {
			conn, err := net.DialTimeout(protocol, targetAddress, time.Second)
			if err != nil {
				return
			}
			conn.Write(sentData)
			conn.(*net.TCPConn).CloseWrite()
			conn.Close()
		}(connectedNode.Address)
	}

	return nil
}

func (wallets *Wallets) signTransaction(transaction *blockchain.Transaction, privKey ecdsa.PrivateKey) {
	for inputIndex, txnInput := range transaction.Inputs {
		inputHash := txnInput.Hash()
		r, s, _ := ecdsa.Sign(rand.Reader, &privKey, inputHash)
		signature := append(r.Bytes(), s.Bytes()...)
		transaction.Inputs[inputIndex].ScriptSig.Signature = signature
	}
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
			accountBalance += output.Value
		}
	}
	return accountBalance
}

func (wallets *Wallets) getUTXOs(walletAddress string) (map[string]blockchain.TxOutputs, error) {
	getUTXOMsg := network.GetUTXOMessage{TargetAddress: walletAddress}
	sentData := append(msgTypeToBytes(network.GETUTXO_MSG), serialize(getUTXOMsg)...)

	successFlag := make(chan bool, len(wallets.connectedNodes))
	resultChan := make(chan map[string]blockchain.TxOutputs, len(wallets.connectedNodes))

	for _, connectedNode := range wallets.connectedNodes {
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
		}(connectedNode.Address)
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

func (wallets *Wallets) AddWalletAddrToSPVNodes(walletAddress string) {
	newAddrMsg := network.NewAddrMessage{WalletAddress: walletAddress}
	sentData := append(msgTypeToBytes(network.NEWADDR_MSG), serialize(newAddrMsg)...)
	for _, connectedNode := range wallets.connectedNodes {
		if connectedNode.NodeType == network.SPV {
			go func(targetAddress string) {
				conn, err := net.DialTimeout(protocol, targetAddress, time.Second)
				if err != nil {
					return
				}
				conn.Write(sentData)
				conn.(*net.TCPConn).CloseWrite()
				conn.Close()
			}(connectedNode.Address)
		}
	}
}
