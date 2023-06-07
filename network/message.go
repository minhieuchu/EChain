package network

import "EChain/blockchain"

type VersionMessage struct {
	Version    int
	AddrYou    string
	AddrMe     string
	BestHeight int
}

type VerackMessage struct {
	AddrFrom string
}

type AddrMessage struct {
	Address string
}

type GetblocksMessage struct {
	TopBlockHash []byte
	AddrFrom     string
}

type GetheadersMessage struct {
	TopHeaderHash []byte
	AddrFrom     string
}

type InvMessage struct {
	HashList [][]byte
}

type HeaderMessage struct {
	HeaderList [][]byte
}

type GetdataMessage struct {
	Index    int
	HashList [][]byte
	AddrFrom string
}

type BlockdataMessage struct {
	Index     int
	BlockList []*blockchain.Block
}
