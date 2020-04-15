package bc

import (
	"context"
	"crypto/ed25519"
)

type connectedPeer struct {
	Address string
	In      chan Message
	Out     chan Message
	cancel  context.CancelFunc
}

// Default message struct
type Message struct {
	Id   uint64
	From string
	Data interface{}
}

// Node representation
type NodeInfoResp struct {
	NodeName        string
	BlockNum        uint64
	LastBlockHash   string
	TotalDifficulty uint64
}

type BlocksRequest struct {
	BlockNumFrom uint64
	BlockNumTo   uint64
}

// Chain block
type Block struct {
	BlockNum      uint64
	Timestamp     int64
	Transactions  []Transaction
	BlockHash     string `json:"-"`
	PrevBlockHash string
	StateHash     string
	Signature     []byte `json:"-"`
}

// Send money transaction
type Transaction struct {
	From   string
	To     string
	Amount uint64
	Fee    uint64
	PubKey ed25519.PublicKey

	Signature []byte `json:"-"`
}
