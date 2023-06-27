package network

import (
	"EChain/blockchain"
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	NETWORK_NODES_NUM  = 1
	FULLNODE_BLOCK_NUM = 50
)

func TestBlockHeaderHeightSPVNode(t *testing.T) {
	var blockHeaderHeight int
	minerNode := NewMinerNode("", "")

	var wg sync.WaitGroup
	for i := 0; i < NETWORK_NODES_NUM; i++ {
		wg.Add(1)
		portNumber := 8333 + i
		go func() {
			fullnode := NewFullNode("localhost:" + fmt.Sprint(portNumber))
			for i := 0; i < FULLNODE_BLOCK_NUM; i++ {
				var block blockchain.Block
				lastHash, _ := fullnode.Blockchain.DataBase.Get([]byte(blockchain.LAST_HASH_STOGAGE_KEY), nil)
				block.PrevHash = lastHash
				minerNode.mineBlock(&block)
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
		spvNode := NewSPVNode("localhost:8888")
		go func() {
			time.Sleep(3 * time.Second)
			blockHeaderHeight = spvNode.GetHeaderHeight()
			wg.Done()
		}()
		spvNode.StartP2PNode()
	}()
	wg.Wait()

	if blockHeaderHeight != FULLNODE_BLOCK_NUM+1 {
		t.Fatalf("Expected SPV header's length to be %d", FULLNODE_BLOCK_NUM+1)
	}
}
