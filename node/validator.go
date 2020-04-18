package node

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"simple_chain/genesis"
	"simple_chain/msg"
	"time"
)

const (
	TransactionsPerBlock = 10
)

type Validator struct {
	//embedded node
	Node
	//transaction hash - > transaction
	transactionPool map[string]msg.Transaction
	//validator index
	index int
}

func NewValidator(genesis *genesis.Genesis) (*Validator, error) {
	// generate validator key
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	// append itself to the validators
	genesis.Validators = append(genesis.Validators, privateKey.Public())
	// init node
	nd, err := NewNode(privateKey, genesis)
	if err != nil {
		return nil, err
	}
	// return new validator
	return &Validator{
		Node:            *nd,
		transactionPool: make(map[string]msg.Transaction),
		index:           len(genesis.Validators) - 1,
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

func (c *Validator) startValidating() {
	ctx := context.Background()
	//endless loop
	for {
		if c.isMyTurn() {
			fmt.Println(simplifyAddress(c.address), "validating block...")
			//get new block
			block, err := c.newBlock()
			if err != nil {
				// fixme panic
				panic(err)
			}
			// fixme when get own block in peer loop
			c.state.Put(c.address, 1000)
			//send new block
			fmt.Println(simplifyAddress(c.address), "generated new block [", simplifyAddress(block.BlockHash), "]")
			c.Broadcast(ctx, msg.Message{
				From: c.address,
				Data: block,
			})
		}
	}
}

func (c *Validator) newBlock() (msg.Block, error) {
	// remove first 0-n transactions from transaction pool
	var txs = make([]msg.Transaction, 0, TransactionsPerBlock)
	var i = 0
	for hash, tr := range c.transactionPool {
		if i++; i > TransactionsPerBlock {
			break
		}
		// fixme hmm, seems not safe...
		delete(c.transactionPool, hash)
		txs = append(txs, tr)
	}

	prevBlockHash, err := c.GetBlockByNumber(c.lastBlockNum).Hash()
	if err != nil {
		return msg.Block{}, err
	}

	block := msg.Block{
		BlockNum:      c.lastBlockNum + 1,
		Timestamp:     time.Now().Unix(),
		Transactions:  txs,
		BlockHash:     "", // fill later
		PrevBlockHash: prevBlockHash,
		StateHash:     nil, // fill later
		Signature:     nil, // fill later
	}
	// apply block
	err = c.insertBlock(block)
	if err != nil {
		return msg.Block{}, err
	}
	// state hash
	block.StateHash, err = c.state.Hash()
	if err != nil {
		return msg.Block{}, err
	}
	// block hash
	block.BlockHash, err = block.Hash()
	if err != nil {
		return msg.Block{}, err
	}
	// block signature
	bts, err := block.Bytes()
	if err != nil {
		return msg.Block{}, err
	}
	block.Signature = ed25519.Sign(c.key, bts)

	return block, nil
}

func (c *Validator) isMyTurn() bool {
	//blockNum remainder
	r := int(c.lastBlockNum) % len(c.validators)
	return r == c.index
}
