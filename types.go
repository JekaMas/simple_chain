package bc

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/gob"
	"errors"
	"sort"
	"time"
)

type Block struct {
	BlockNum      uint64
	Timestamp     int64
	Transactions  []Transaction
	BlockHash     string `json:"-"`
	PrevBlockHash string
	StateHash     string
	Signature     []byte `json:"-"`
}

func (bl *Block) Hash() (string, error) {
	if bl == nil {
		return "", errors.New("empty block")
	}
	b, err := Bytes(bl)
	if err != nil {
		return "", err
	}
	return Hash(b), nil
}

type Transaction struct {
	From   string
	To     string
	Amount uint64
	Fee    uint64
	PubKey ed25519.PublicKey

	Signature []byte `json:"-"`
}

func (t Transaction) Hash() (string, error) {
	b, err := Bytes(t)
	if err != nil {
		return "", err
	}
	return Hash(b), nil
}

func (t Transaction) Bytes() ([]byte, error) {
	b := bytes.NewBuffer(nil)
	err := gob.NewEncoder(b).Encode(t)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// first block with blockchain settings
type Genesis struct {
	//Account -> funds
	Alloc map[string]uint64
	//list of validators public keys
	Validators []crypto.PublicKey
}

func (g Genesis) ToBlock() Block {
	// sort Alloc keys (lexicographical order)
	keys := make([]string, 0, len(g.Alloc))
	for k := range g.Alloc {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// get slice of genesis transactions from initial funds
	var trs []Transaction
	for account, fund := range g.Alloc {
		trs = append(trs, Transaction{
			From:      "",
			To:        account,
			Amount:    fund,
			Fee:       0,
			PubKey:    nil,
			Signature: nil,
		})
	}

	// state hash
	allocBytes, err := Bytes(g.Alloc)
	if err != nil {
		panic("can't convert genesis to block: can't get alloc bytes")
	}

	block := Block{
		BlockNum:      0,
		Timestamp:     time.Now().Unix(),
		Transactions:  trs,
		BlockHash:     "",
		PrevBlockHash: "",
		StateHash:     Hash(allocBytes),
		Signature:     nil,
	}

	// block hash
	blockBytes, err := Bytes(block)
	if err != nil {
		panic("can't convert genesis to block: can't get block bytes")
	}
	block.BlockHash = Hash(blockBytes)

	return block
}

type Message struct {
	From string
	Data interface{}
}

type NodeInfoResp struct {
	NodeName string
	BlockNum uint64
	LastBlockHash string
	TotalDifficulty uint64
}

type connectedPeer struct {
	Address string
	In      chan Message
	Out     chan Message
	cancel  context.CancelFunc
}

func (cp connectedPeer) Send(ctx context.Context, m Message) {
	// todo timeout using context + done check
	cp.Out <- m
}