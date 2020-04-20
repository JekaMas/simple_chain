package genesis

import (
	"crypto"
	"simple_chain/encode"
	"simple_chain/msg"
	"sort"
	"strings"
	"time"
)

var (
	genesisTime = time.Now().Unix()
)

// first block with blockchain settings
type Genesis struct {
	// account -> funds
	Alloc map[string]uint64
	// list of validators public keys
	Validators []crypto.PublicKey
}

func New() Genesis {
	return Genesis{
		Alloc:      make(map[string]uint64),
		Validators: []crypto.PublicKey{},
	}
}

func (g Genesis) ToBlock() msg.Block {
	// get slice of genesis transactions from initial funds
	var trs []msg.Transaction
	for account, fund := range g.Alloc {
		trs = append(trs, msg.Transaction{
			From:      "",
			To:        account,
			Amount:    fund,
			Fee:       0,
			PubKey:    nil,
			Signature: nil,
		})
	}
	// sort genesis transactions
	sort.Slice(trs, func(i, j int) bool {
		return strings.Compare(trs[i].To, trs[j].To) > 0
	})
	// state hash
	allocHash, err := encode.HashAlloc(g.Alloc)
	if err != nil {
		panic("can't convert genesis to block: can't get alloc bytes")
	}

	block := msg.Block{
		BlockNum:      0,
		Timestamp:     genesisTime,
		Transactions:  trs,
		BlockHash:     "",
		PrevBlockHash: "",
		StateHash:     allocHash,
		Signature:     nil,
	}
	// block hash
	blockBytes, err := encode.Bytes(block)
	if err != nil {
		panic("can't convert genesis to block: can't get block bytes")
	}
	block.BlockHash = encode.Hash(blockBytes)

	return block
}
