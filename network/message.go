package network

type versionMessage struct {
	Version    int
	AddrYou    string
	AddrMe     string
	BestHeight int
}

type verackMessage struct {
	AddrFrom string
}

type addrMessage struct {
	Address string
}

type getblocksMessage struct {
	TopBlockHash []byte
	AddrFrom     string
}
