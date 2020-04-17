package node

import (
	"bytes"
	"crypto/ed25519"
	bc "simple_chain"
)

type Validator struct {
	//embedded node
	Node
	//transaction hash - > transaction
	transactionPool map[string]bc.Transaction
}

func NewValidator(key ed25519.PrivateKey, genesis bc.Genesis) (*Validator, error) {
	// init node
	nd, err := NewNode(key, genesis)
	if err != nil {
		return nil, err
	}
	return &Validator{
		Node:            *nd,
		transactionPool: make(map[string]bc.Transaction),
	}, nil
}

func (c *Validator) AddTransaction(tr bc.Transaction) error {
	hash, err := tr.Hash()
	if err != nil {
		return err
	}
	c.transactionPool[hash] = tr
	return nil
}

func (c *Validator) isValidator() bool {
	for _, key := range c.genesis.Validators {
		bts1, _ := bc.Bytes(key)
		bts2, _ := bc.Bytes(c.key.Public())
		if bytes.Equal(bts1, bts2) {
			return true
		}
	}
	return false
}

/* --- Processes ---------------------------------------------------------------------------------------------------- */

func (c *Validator) startValidating() {

}
