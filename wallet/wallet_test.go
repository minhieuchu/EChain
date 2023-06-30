package wallet

import (
	"EChain/blockchain"
	"EChain/network"
	"testing"
	"time"
)

func TestGetBalance(t *testing.T) {
	wallets := NewWallets()
	walletAddr := wallets.AddNewWallet()
	minerAddr := "localhost:8333"
	fullnodeAddr := "localhost:8334"
	spvAddr := "localhost:8335"

	minerNode := network.NewMinerNode(minerAddr, walletAddr)
	go minerNode.StartP2PNode()

	fullNode := network.NewFullNode(fullnodeAddr)
	go fullNode.StartP2PNode()

	spvNode := network.NewSPVNode(spvAddr)
	go spvNode.StartP2PNode()

	wallets.ConnectNode(network.SPV, spvAddr)
	time.Sleep(2 * time.Second) // Wait for 3 nodes to finish connecting / synchronizing data
	wallets.AddWalletAddrToSPVNodes(walletAddr)

	time.Sleep(7 * time.Second)
	balance := wallets.GetBalance(walletAddr)
	if balance != blockchain.COINBASE_REWARD {
		t.Fatalf("Expected balance to be %d , actual: %d", blockchain.COINBASE_REWARD, balance)
	}
}
