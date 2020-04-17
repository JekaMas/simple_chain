package bc

import (
	"crypto"
	"simple_chain/msg"
	"sort"
	"time"
)

// first block with blockchain settings
type Genesis struct {
	//Account -> funds
	Alloc map[string]uint64
	//list of validators public keys
	Validators []crypto.PublicKey
}

func (g Genesis) ToBlock() msg.Block {
	// sort Alloc keys (lexicographical order)
	keys := make([]string, 0, len(g.Alloc))
	for k := range g.Alloc {
		keys = append(keys, k)
	}
	sort.Strings(keys)

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

	// state hash
	allocBytes, err := Bytes(g.Alloc)
	if err != nil {
		panic("can't convert genesis to block: can't get alloc bytes")
	}

	block := msg.Block{
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
