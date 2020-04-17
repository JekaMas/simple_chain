package node

import (
	"crypto/ed25519"
	bc "simple_chain"
	"simple_chain/msg"
)

type Validator struct {
	//embedded node
	Node
	//transaction hash - > transaction
	transactionPool map[string]msg.Transaction
	//validator index
	index uint64
}

func NewValidator(key ed25519.PrivateKey, genesis bc.Genesis, index uint64) (*Validator, error) {
	// init node
	nd, err := NewNode(key, genesis)
	if err != nil {
		return nil, err
	}
	return &Validator{
		Node:            *nd,
		transactionPool: make(map[string]msg.Transaction),
		index:           index,
	}, nil
}

func (c *Validator) AddTransaction(tr msg.Transaction) error {
	hash, err := tr.Hash()
	if err != nil {
		return err
	}
	c.transactionPool[hash] = tr
	return nil
}

/* --- Processes ---------------------------------------------------------------------------------------------------- */

func (c *Validator) startValidating() {
	for {
		if c.isMyBlock() {
			block := c.newBlock()
		}
	}
}

/* --- Common ------------------------------------------------------------------------------------------------------- */

func (c *Validator) newBlock() msg.Block {

}

func (c *Validator) isMyBlock() bool {
	//blockNum remainder
	r := (c.lastBlockNum + 1) % c.index
	return r == 0
}
