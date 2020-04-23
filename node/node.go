package node

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"simple_chain/genesis"
	"simple_chain/logger"
	"simple_chain/msg"
	"simple_chain/storage"
	"sync"
)

const (
	MessagesBusLen = 100
	TransactionFee = 10
)

type connectedPeer struct {
	Address string
	In      chan msg.Message
	Out     chan msg.Message
	cancel  context.CancelFunc
}

type Node struct {
	key          ed25519.PrivateKey
	address      string
	genesis      *genesis.Genesis
	lastBlockNum uint64

	//chain
	blocks []msg.Block
	//peer address -> peer info
	peers map[string]connectedPeer
	//peer address -> fund
	state      storage.Storage
	validators []crypto.PublicKey

	mxPeers  *sync.Mutex
	mxBlocks *sync.Mutex
	logger   logger.Logger
}

func NewNode(key ed25519.PrivateKey, genesis *genesis.Genesis) (*Node, error) {
	address, err := PubKeyToAddress(key.Public())
	if err != nil {
		return nil, err
	}

	state := storage.FromGenesis(genesis)

	return &Node{
		key:          key,
		address:      address,
		genesis:      genesis,
		blocks:       []msg.Block{genesis.ToBlock()},
		lastBlockNum: 0,
		peers:        make(map[string]connectedPeer, 0),
		state:        state,
		validators:   genesis.Validators,

		mxBlocks: &(sync.Mutex{}),
		mxPeers:  &(sync.Mutex{}),
		logger:   logger.New(logger.All),
	}, nil
}

/* --- Interface ---------------------------------------------------------------------------------------------------- */

func (c *Node) NodeKey() crypto.PublicKey {
	return c.key.Public()
}

func (c *Node) Connection(address string, in chan msg.Message, outs ...chan msg.Message) chan msg.Message {
	var out chan msg.Message
	if len(outs) == 0 {
		out = make(chan msg.Message, MessagesBusLen)
	} else {
		out = outs[0]
	}

	ctx, cancel := context.WithCancel(context.Background())

	c.mxPeers.Lock()
	defer c.mxPeers.Unlock()
	c.peers[address] = connectedPeer{
		Address: address,
		Out:     out,
		In:      in,
		cancel:  cancel,
	}

	go c.peerLoop(ctx, c.peers[address])
	return c.peers[address].Out
}

func (c *Node) AddPeer(peer Blockchain) error {
	remoteAddress, err := PubKeyToAddress(peer.NodeKey())
	if err != nil {
		return err
	}

	if c.address == remoteAddress {
		return errors.New("self connection")
	}

	if _, ok := c.peers[remoteAddress]; ok {
		return nil
	}

	if v, ok := peer.(*Validator); ok {
		if !c.containsValidator(v) {
			c.validators = append(c.validators, v.NodeKey())
		}
	}

	out := make(chan msg.Message, MessagesBusLen)
	in := peer.Connection(c.address, out)
	c.Connection(remoteAddress, in, out)
	return nil
}

func (c *Node) Broadcast(ctx context.Context, msg msg.Message) {
	c.mxPeers.Lock()
	defer c.mxPeers.Unlock()

	for _, v := range c.peers {
		if msg.From != v.Address && v.Address != c.address {
			c.SendMessageTo(v, ctx, msg)
		}
	}
}

func (c *Node) RemovePeer(peer Blockchain) error {
	delete(c.peers, peer.NodeAddress())
	return nil
}

func (c *Node) GetBalance(account string) (uint64, error) {
	fund, err := c.state.Get(account)
	if err != nil {
		return 0, err
	}
	return fund, nil
}

func (c *Node) AddTransaction(tr msg.Transaction) error {
	//nothing for node
	return nil
}

func (c *Node) GetBlockByNumber(ID uint64) msg.Block {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()

	return c.blocks[ID] // todo make check and other stuff
}

func (c *Node) NodeInfo() msg.NodeInfoResp {
	c.mxBlocks.Lock()
	lastBlock := c.blocks[len(c.blocks)-1]
	c.mxBlocks.Unlock()

	return msg.NodeInfoResp{
		NodeName:        c.address,
		BlockNum:        c.lastBlockNum,
		LastBlockHash:   lastBlock.BlockHash,
		TotalDifficulty: 1, // todo totalDifficult
	}
}

func (c *Node) NodeAddress() string {
	return c.address
}

func (c *Node) SignTransaction(transaction msg.Transaction) (msg.Transaction, error) {
	b, err := transaction.Bytes()
	if err != nil {
		return msg.Transaction{}, err
	}

	transaction.Signature = ed25519.Sign(c.key, b)
	return transaction, nil
}

func (c *Node) SendTo(cp connectedPeer, ctx context.Context, data interface{}) {
	m := msg.Message{
		From: c.address,
		Data: data,
	}
	c.SendMessageTo(cp, ctx, m)
}

func (c *Node) SendMessageTo(cp connectedPeer, ctx context.Context, msg msg.Message) {
	//todo timeout using context + done check
	cp.Out <- msg
}

/* --- Processes ---------------------------------------------------------------------------------------------------- */

func (c *Node) peerLoop(ctx context.Context, peer connectedPeer) {
	//handshake
	c.SendTo(peer, ctx, c.NodeInfo())

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-peer.In:
			err := c.processMessage(ctx, peer, message)
			if err != nil {
				c.logger.Errorf("%v process peer error: %v", simplifyAddress(c.address), err)
				continue
			}
			//broadcast to connected peers
			c.Broadcast(ctx, message)
		}
	}
}

func (c *Node) processMessage(ctx context.Context, peer connectedPeer, message msg.Message) error {
	switch m := message.Data.(type) {
	// get transaction from another peer
	case msg.Transaction:
		// add all transaction to transaction pool (for validator)
		return c.AddTransaction(m)
	// received block
	case msg.Block:
		c.logger.Infof("%v receive block [%v] from %v",
			simplifyAddress(c.address), simplifyAddress(m.BlockHash), simplifyAddress(peer.Address))
		if err := c.verifyBlock(m); err != nil {
			return fmt.Errorf("can't process message: %v", err)
		}
		if err := c.insertBlock(m); err != nil {
			return fmt.Errorf("can't process message: %v", err)
		}
		c.logger.Infof("%v insert new block [%v]", simplifyAddress(c.address), simplifyAddress(m.BlockHash))
	// send blocks to peer that requested
	case msg.BlocksRequest:
		// if my request
		if m.From == c.NodeAddress() {
			for id := m.BlockNumFrom; id <= m.BlockNumTo; id++ {
				c.logger.Infof("%v send block [%v] to %v",
					simplifyAddress(c.address), simplifyAddress(c.blocks[id].BlockHash), simplifyAddress(peer.Address))
				c.SendTo(peer, ctx, c.GetBlockByNumber(id))
			}
		}
	// get info from another peer
	case msg.NodeInfoResp:
		c.mxBlocks.Lock()
		defer c.mxBlocks.Unlock()
		// blocks request
		if c.lastBlockNum < m.BlockNum {
			c.logger.Infof("%v connect to %v need sync", simplifyAddress(c.address), simplifyAddress(peer.Address))
			c.SendTo(peer, ctx, msg.BlocksRequest{
				From:         m.NodeName,
				To:           c.NodeAddress(),
				BlockNumFrom: c.lastBlockNum + 1,
				BlockNumTo:   m.BlockNum,
			})
		}
	}
	return nil
}

/* -- Common -------------------------------------------------------------------------------------------------------- */

func PubKeyToAddress(key crypto.PublicKey) (string, error) {
	if v, ok := key.(ed25519.PublicKey); ok {
		b := sha256.Sum256(v)
		return hex.EncodeToString(b[:]), nil
	}
	return "", errors.New("incorrect key")
}

func simplifyAddress(address string) string {
	if len(address) < 4 {
		return address
	}
	return address[:4]
}

func (c *Node) newTransaction(toAddress string, amount uint64) (msg.Transaction, error) {
	tr := msg.Transaction{
		From:   c.address,
		To:     toAddress,
		Amount: amount,
		Fee:    TransactionFee,
		PubKey: c.key.Public().(ed25519.PublicKey),
	}
	return c.SignTransaction(tr)
}

/*
type Block struct {
	Timestamp     int64 ???
}
*/
func (c *Node) verifyBlock(block msg.Block) error {
	// block num
	if block.BlockNum < 0 {
		return errors.New("incorrect block num")
	}
	if block.BlockNum <= c.lastBlockNum {
		return fmt.Errorf("already have block [%v <= %v]", block.BlockNum, c.lastBlockNum)
	}

	validatorAddr, err := c.validatorAddr(block)
	if err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}

	// verify transactions
	stateCopy := c.state.Copy()
	for _, tr := range block.Transactions {
		if err := verifyTransaction(stateCopy, tr); err != nil {
			return fmt.Errorf("can't verify block: %v", err)
		}
		if err := applyTransaction(stateCopy, validatorAddr, tr); err != nil {
			return fmt.Errorf("can't verify block: %v", err)
		}
	}

	// verify state hash
	stateHash, err := stateCopy.Hash()
	if err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}
	if stateHash != block.StateHash {
		return errors.New("state hash is incorrect")
	}

	// get validator public key
	validatorNum := block.BlockNum % uint64(len(c.validators))
	validatorKey, ok := c.validators[validatorNum].(ed25519.PublicKey)
	if !ok {
		return errors.New("can't convert public key to ed25519")
	}

	// check signature
	sig := block.Signature
	block.Signature = nil
	bts, err := block.Bytes()
	if err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}
	if !ed25519.Verify(validatorKey, bts, sig) {
		return errors.New("block signature is incorrect")
	}

	// check block hash
	getBlockHash := block.BlockHash
	block.BlockHash = ""
	wantBlockHash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}
	if getBlockHash != wantBlockHash {
		return errors.New("block hash is incorrect")
	}

	// check parent hash
	prevBlockHash := c.GetBlockByNumber(block.BlockNum - 1).BlockHash
	if prevBlockHash != block.PrevBlockHash {
		return errors.New("parent hash is incorrect")
	}
	return nil
}

func verifyTransaction(state storage.Storage, tr msg.Transaction) error {
	// check state funds
	fund, err := state.Get(tr.From)
	if err != nil {
		return fmt.Errorf("can't verify transaction: %v", err)
	}
	if fund < tr.Amount+tr.Fee {
		return fmt.Errorf("insufficient funds: %v < %v", fund, tr.Amount+tr.Fee)
	}
	// check signature
	sig := tr.Signature
	tr.Signature = nil
	bts, err := tr.Bytes()
	if err != nil {
		return fmt.Errorf("can't convert transaction to bytes: %v", err)
	}
	if len(tr.PubKey) == 0 {
		return errors.New("transaction key is empty")
	}
	if !ed25519.Verify(tr.PubKey, bts, sig) {
		return errors.New("transaction signature is incorrect")
	}
	return nil
}

func (c *Node) validatorAddr(b msg.Block) (string, error) {
	validatorKey := c.validators[int(b.BlockNum%uint64(len(c.validators)))]
	return PubKeyToAddress(validatorKey)
}

func (c *Node) containsValidator(v *Validator) bool {
	for _, pubKey := range c.validators {
		addr1, _ := PubKeyToAddress(pubKey)
		addr2, _ := PubKeyToAddress(v.NodeKey())
		if addr1 == addr2 {
			return true
		}
	}
	return false
}

func (c *Node) insertBlock(b msg.Block) error {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()

	for _, tr := range b.Transactions {
		validatorAddr, err := c.validatorAddr(b)
		if err != nil {
			return err
		}

		err = applyTransaction(c.state, validatorAddr, tr)
		if err != nil {
			return err
		}
	}

	c.blocks = append(c.blocks, b)
	c.lastBlockNum += 1
	return nil
}

func applyTransaction(state storage.Storage, validatorAddress string, tr msg.Transaction) error {
	state.Lock()
	defer state.Unlock()

	err := state.Sub(tr.From, tr.Amount+tr.Fee)
	if err != nil {
		return err
	}
	err = state.Add(tr.To, tr.Amount)
	if err != nil {
		return err
	}
	err = state.Add(validatorAddress, tr.Fee)
	if err != nil {
		return err
	}
	return nil
}
