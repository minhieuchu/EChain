package main

import (
	"EChain/blockchain"
	"EChain/network"
	"EChain/wallet"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBlockHeaderHeightSPVNode( t *testing.T) {
	wallets := wallet.LoadWallets()
	addressList := wallets.GetAddresses()
	var blockHeaderHeight int

	var wg sync.WaitGroup
	for i := 0; i < NETWORK_NODES_NUM; i++ {
		wg.Add(1)
		portNumber := 8333 + i
		go func() {
			fullnode := network.NewFullNode("localhost:"+fmt.Sprint(portNumber), addressList[0])
			for i := 0; i < FULLNODE_BLOCK_NUM; i++ {
				var block blockchain.Block
				lastHash, _ := fullnode.Blockchain.DataBase.Get([]byte(blockchain.LAST_HASH_STOGAGE_KEY), nil)
				block.PrevHash = lastHash
				block.Mine()
				fullnode.Blockchain.StoreNewBlock(&block)
			}
			go func() {
				time.Sleep(5 * time.Second) // wait for SPV node to finish synchronizing block headers
				wg.Done()
			}()
			fullnode.StartP2PNode()
		}()
	}
	go func() {
		time.Sleep(3 * time.Second) // Wait for fullnode to finish building blocks (including mining time for each block)
		wg.Add(1)
		spvNode := network.NewSPVNode("localhost:8888")
		go func() {
			time.Sleep(3 * time.Second)
			blockHeaderHeight = spvNode.BlockChainHeader.GetHeight()
			wg.Done()
		}()
		spvNode.StartP2PNode()
	}()
	wg.Wait()

	if blockHeaderHeight != FULLNODE_BLOCK_NUM + 1 {
		t.Fatalf("Expected SPV header's length to be %d", FULLNODE_BLOCK_NUM + 1)
	}
}
