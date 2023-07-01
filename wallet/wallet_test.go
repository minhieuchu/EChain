package wallet

import (
	"EChain/blockchain"
	"EChain/network"
	"testing"
	"time"
)

func setup() (*Wallets, string, string) {
	wallets := NewWallets()
	walletAddr1 := wallets.AddNewWallet()
	walletAddr2 := wallets.AddNewWallet()
	minerAddr := "localhost:8333"
	fullnodeAddr := "localhost:8334"
	spvAddr := "localhost:8335"

	minerNode := network.NewMinerNode(minerAddr, walletAddr1)
	go minerNode.StartP2PNode()

	fullNode := network.NewFullNode(fullnodeAddr)
	go fullNode.StartP2PNode()

	spvNode := network.NewSPVNode(spvAddr)
	go spvNode.StartP2PNode()

	wallets.ConnectNode(network.SPV, spvAddr)
	time.Sleep(2 * time.Second) // Wait for 3 nodes to finish connecting / synchronizing data
	wallets.AddWalletAddrToSPVNodes(walletAddr1)

	time.Sleep(7 * time.Second) // Wait for miner node to finish mining the first block after the Genesis block

	return &wallets, walletAddr1, walletAddr2
}

func TestGetBalance(t *testing.T) {
	wallets, walletAddr, _ := setup()
	balance := wallets.GetBalance(walletAddr)
	if balance != blockchain.COINBASE_REWARD {
		t.Fatalf("Expected balance to be %d , actual: %d", blockchain.COINBASE_REWARD, balance)
	}
}

func TestTransfer(t *testing.T) {
	wallets, walletAddr1, walletAddr2 := setup()
	wallets.Transfer(walletAddr1, walletAddr2, 500)
	// Wait for new transaction to be propagated to SPV node
	time.Sleep(time.Second)
	// Wait for miner node to pick new transaction from mempool and start mining
	// Miner nodes create new block every 10 seconds
	time.Sleep(10 * time.Second)
	minerWalletBalance := wallets.GetBalance(walletAddr1)
	receiverWalletBalance := wallets.GetBalance(walletAddr2)

	if minerWalletBalance != blockchain.COINBASE_REWARD-500 || receiverWalletBalance != 500 {
		t.Fatalf("Expected wallet balances to be %d and %d, actual: %d and %d", blockchain.COINBASE_REWARD-500, 500, minerWalletBalance, receiverWalletBalance)
	}
}
