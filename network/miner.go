package network

type MinerNode struct {
	FullNode
}

func NewMinerNode(networkAddress, walletAddress string) *MinerNode {
	fullNode := NewFullNode(networkAddress, walletAddress)
	return &MinerNode{*fullNode}
}
