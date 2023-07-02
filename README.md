# EChain (Emulated Chain)

A simplified version of Bitcoin's P2P network.
This implementation is based on the book [Mastering Bitcoin, 2nd Edition](https://www.oreilly.com/library/view/mastering-bitcoin-2nd/9781491954379/).

EChain supports 3 types of Bitcoin nodes: `Full Node`, `Miner Node`, `Simple Payment Verification Node` (SPV Node).

Users can interact with the network through the `wallet` module, which stores public-private key pairs, and is responsible for creating transactions, requesting account balance.

## Overview

The diagram below describles the structure of this P2P network, focusing on the flow when users create transantions from `wallet` module.
![EChain drawio](https://github.com/minhieuchu/EChain/assets/25933120/cd2f8102-8925-43bd-add3-fb3ea7607526)

## Installation

```
go build
```

This will produce an executable file `EChain`.

To start a blockchain node, open terminal and type the following:

```
./EChain [network address] [node type]
```

The node type can be one of the following `fullnode`, `miner`, `spv`.

Example

```
./EChain localhost:8333 fullnode
./EChain localhost:8334 miner
```

## Testing

Test files are placed inside `wallet` and `network` modules.
Each test needs to be run individually as tests use the same ports on localhost.

Example

```
cd wallet
go test -v -run TestGetBalance 
```
