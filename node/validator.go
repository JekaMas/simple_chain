package node

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"simple_chain/genesis"
	"simple_chain/msg"
	"simple_chain/storage"
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
	index uint64
}

func NewValidator(key ed25519.PrivateKey, index uint64, genesis *genesis.Genesis) (*Validator, error) {
	// init node
	nd, err := NewNode(key, genesis)
	if err != nil {
		return nil, err
	}

	// return new validator
	return &Validator{
		Node:            *nd,
		transactionPool: make(map[string]msg.Transaction),
		index:           index,
	}, nil
}

// NewValidatorFromGenesis - additional constructor
func NewValidatorFromGenesis(index uint64, genesis *genesis.Genesis) (*Validator, error) {
	// generate validator key
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return NewValidator(privateKey, index, genesis)
}

// AddTransaction - add to transaction pool (for validator)
func (c *Validator) AddTransaction(tr msg.Transaction) error {
	hash, err := tr.Hash()
	if err != nil {
		return err
	}
	c.transactionPool[hash] = tr
	return nil
}

func (c *Validator) processBlockMessage(ctx context.Context, peer connectedPeer, block msg.Block) error {
	err := c.Node.processBlockMessage(ctx, peer, block)
	if err != nil {
		return fmt.Errorf("can't process block: %v", err)
	}

	for _, tr := range block.Transactions {
		hash, err := tr.Hash()
		if err != nil {
			return fmt.Errorf("can't process transaction: %v", err)
		}

		delete(c.transactionPool, hash)
	}
	return nil
}

func (c *Validator) startValidating() {
	ctx := context.Background()
	//endless loop
	for {
		if c.isMyTurn() {
			c.logger.Infof("%v validating block...", simplifyAddress(c.address))
			//get new block
			block, err := c.newBlock()
			if err != nil {
				// fixme panic
				panic(err)
			}
			// fixme when get own block in peer loop
			err = c.state.PutOrAdd(c.address, 1000)
			if err != nil {
				// fixme panic
				panic(err)
			}
			//send new block
			c.logger.Infof("%v generated new block [%v]", simplifyAddress(c.address), simplifyAddress(block.BlockHash))
			c.Broadcast(ctx, msg.Message{
				From: c.address,
				Data: block,
			})
		}
	}
}

func (c *Validator) newBlock() (msg.Block, error) {
	txs := c.popTransactions(TransactionsPerBlock)
	err := verifyTransactions(c.state.Copy(), c.NodeAddress(), txs)
	if err != nil {
		return msg.Block{}, fmt.Errorf("can't varify transactions: %v", err)
	}

	prevBlockHash := c.GetBlockByNumber(c.lastBlockNum).BlockHash

	block := msg.Block{
		BlockNum:      c.lastBlockNum + 1,
		Timestamp:     time.Now().Unix(),
		Transactions:  txs,
		PrevBlockHash: prevBlockHash,
		StateHash:     "",  // fill later
		BlockHash:     "",  // fill later
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

	// apply additional fields
	c.blocks[c.lastBlockNum].StateHash = block.StateHash
	c.blocks[c.lastBlockNum].BlockHash = block.BlockHash
	c.blocks[c.lastBlockNum].Signature = block.Signature
	// return new block
	return block, nil
}

func verifyTransactions(stateCopy storage.Storage, validatorAddress string, txs []msg.Transaction) error {
	for _, tr := range txs {
		hash, err := tr.Hash()
		if err != nil {
			return fmt.Errorf("can't get transaction hash: %v", err)
		}
		if err := verifyTransaction(stateCopy, tr); err != nil {
			return fmt.Errorf("transaction %v verify failure: %v", simplifyAddress(hash), err)
		}
		if err := applyTransaction(stateCopy, validatorAddress, tr); err != nil {
			return fmt.Errorf("transaction %v apply failure: %v", simplifyAddress(hash), err)
		}
	}
	return nil
}

// popTransactions - remove first 0-n transactions from transaction pool
func (c *Validator) popTransactions(maxCount uint64) []msg.Transaction {
	txs := make([]msg.Transaction, 0, maxCount)
	i := uint64(0)

	for hash, tr := range c.transactionPool {
		if i++; i > maxCount {
			break
		}
		// fixme hmm, seems not safe...
		delete(c.transactionPool, hash)
		txs = append(txs, tr)
	}
	return txs
}

func (c *Validator) isMyTurn() bool {
	//blockNum remainder
	r := c.lastBlockNum % uint64(len(c.validators))
	//c.logger.Infof("validator%v index=%v/%v my_turn=%v",
	//	c.index, r, len(c.validators)-1, r == uint64(c.index))
	return r == uint64(c.index)
}
