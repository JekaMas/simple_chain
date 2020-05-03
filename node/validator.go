package node

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"simple_chain/genesis"
	"simple_chain/msg"
	"simple_chain/pool"
	"simple_chain/storage"
	"time"
)

const (
	TransactionsPerBlock = 10
	BlockReward          = 1000
	BlockDifficulty      = 3
)

type Validator struct {
	Node
	transactionPool  pool.TransactionPool
	validatingCancel context.CancelFunc
}

func NewValidator(genesis genesis.Genesis) (*Validator, error) {
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return NewValidatorWithKey(genesis, key)
}

func NewValidatorWithKey(genesis genesis.Genesis, key ed25519.PrivateKey) (*Validator, error) {
	// init node
	nd, err := NewNodeWithKey(genesis, key)
	if err != nil {
		return nil, err
	}

	// return new validator
	return &Validator{
		Node:            *nd,
		transactionPool: pool.NewTransactionPool(),
	}, nil
}

// AddTransaction - add verified ! transaction to transaction pool (for validator)
func (c *Validator) AddTransaction(tr msg.Transaction) error {
	return c.transactionPool.Insert(tr)
}

func (c *Validator) processBlockMessage(ctx context.Context, peer connectedPeer, block msg.Block) error {
	err := c.Node.processBlockMessage(ctx, peer, block)
	if err != nil {
		return fmt.Errorf("can't process block: %v", err)
	}
	// stop if possible
	needStart := c.stopValidating() == nil
	// remove transactions from pool
	for _, tr := range block.Transactions {
		c.transactionPool.Delete(tr)
	}
	// start if possible
	if needStart {
		c.startValidating()
	}
	return nil
}

func (c *Validator) startValidating() {
	ctx, cancel := context.WithCancel(context.Background())
	c.validatingCancel = cancel

	c.logger.Infof("%v validating blocks...", simplifyAddress(c.address))
	go func() {
		for {
			// validate new block
			block, err := c.newBlock()
			if err != nil {
				c.logger.Errorf("error generating block: %v", err)
				// stop validating
				return
			}
			c.logger.Infof("%v generated new block [%v]",
				simplifyAddress(c.address), simplifyAddress(block.BlockHash))

			select {
			case <-ctx.Done():
				// return transactions
				if len(block.Transactions) > 1 {
					for _, tr := range block.Transactions[1:] {
						err := c.AddTransaction(tr)
						if err != nil {
							c.logger.Errorf("can't restore transaction: %v", err)
						}
					}
				}
				return
			default:
				// send new block
				c.Broadcast(ctx, msg.Message{
					From: c.address,
					Data: block,
				})
			}
		}
	}()
}

func (c *Validator) stopValidating() error {
	if c.validatingCancel == nil {
		return errors.New("no validating was started")
	}

	c.validatingCancel()
	c.validatingCancel = nil
	return nil
}

func (c *Validator) newBlock() (msg.Block, error) {
	txs := c.transactionPool.Peek(TransactionsPerBlock)
	err := verifyTransactions(c.state.Copy(), c.NodeAddress(), txs)
	if err != nil {
		return msg.Block{}, fmt.Errorf("can't varify transactions: %v", err)
	}
	// add reward transaction
	txs = append([]msg.Transaction{c.coinbaseTransaction()}, txs...)

	prevBlockHash := c.GetBlockByNumber(c.lastBlockNum).BlockHash

	block := msg.Block{
		BlockNum:      c.lastBlockNum + 1,
		Nonce:         0,
		Timestamp:     time.Now().UnixNano(),
		Transactions:  txs,
		PrevBlockHash: prevBlockHash,
		StateHash:     "",  // fill later
		BlockHash:     "",  // fill later
		Signature:     nil, // fill later
		PubKey:        nil, // fill later
	}

	// apply block to state copy
	stateCopy := c.state.Copy()
	for _, tr := range block.Transactions[1:] {
		err := verifyTransaction(stateCopy, tr)
		// skip incorrect transactions
		if err != nil {
			continue
		}
		// if all correct - apply and delete from pool
		err = applyTransaction(stateCopy, c.NodeAddress(), tr)
		if err != nil {
			return msg.Block{}, err
		}
		c.transactionPool.Delete(tr)
	}
	err = applyCoinbaseTransaction(stateCopy, block.Transactions[0])
	if err != nil {
		return msg.Block{}, err
	}

	// state hash
	block.StateHash, err = stateCopy.Hash()
	if err != nil {
		return msg.Block{}, err
	}

	// block hash
	block.BlockHash, err = c.validateBlockHash(&block)
	if err != nil {
		return msg.Block{}, err
	}

	// block signature
	bts, err := block.Bytes()
	if err != nil {
		return msg.Block{}, err
	}
	block.Signature = ed25519.Sign(c.key, bts)
	block.PubKey = c.NodeKey().(ed25519.PublicKey)

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

func (c *Validator) validateBlockHash(block *msg.Block) (string, error) {
	for {
		// endless - get hash
		hash, err := block.Hash()
		if err != nil {
			return "", err
		}
		// if found
		difficulty := leadingZeros(hash)
		if difficulty >= BlockDifficulty {
			return hash, nil
		}
		// else - next nonce
		block.Nonce++
	}
}

func (c *Validator) coinbaseTransaction() msg.Transaction {
	return msg.Transaction{
		From:      "",
		To:        c.NodeAddress(),
		Amount:    BlockReward,
		Fee:       0,
		PubKey:    nil,
		Signature: nil,
	}
}

func leadingZeros(hash string) uint64 {
	i := uint64(0)
	for _, c := range hash {
		if c == '0' {
			i++
		} else {
			break
		}
	}
	return i
}
