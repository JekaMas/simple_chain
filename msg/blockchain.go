package msg

import (
	"bytes"
	"crypto/ed25519"
	"encoding/gob"
	"errors"
	bc "simple_chain"
)

// Default message struct
type Message struct {
	From string
	Data interface{}
}

// --- Chain block -----------------------------------------------------------------------------------------------------
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
	b, err := bc.Bytes(bl)
	if err != nil {
		return "", err
	}
	return bc.Hash(b), nil
}

// --- Send money transaction ------------------------------------------------------------------------------------------
type Transaction struct {
	From      string
	To        string
	Amount    uint64
	Fee       uint64
	PubKey    ed25519.PublicKey
	Signature []byte `json:"-"`
}

func (t Transaction) Hash() (string, error) {
	b, err := t.Bytes()
	if err != nil {
		return "", err
	}
	return bc.Hash(b), nil
}

func (t Transaction) Bytes() ([]byte, error) {
	b := bytes.NewBuffer(nil)
	err := gob.NewEncoder(b).Encode(t)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// --- Node representation ---------------------------------------------------------------------------------------------
type NodeInfoResp struct {
	NodeName        string
	BlockNum        uint64
	LastBlockHash   string
	TotalDifficulty uint64
}

// --- Request for needed blocks ---------------------------------------------------------------------------------------
type BlocksRequest struct {
	BlockNumFrom uint64
	BlockNumTo   uint64
}
