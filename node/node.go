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
	"simple_chain/pool"
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
	blocks    []msg.Block
	blockPool pool.BlockPool
	//peer address -> peer info
	peers map[string]connectedPeer
	//peer address -> fund
	state storage.Storage

	mxPeers  *sync.Mutex
	mxBlocks *sync.Mutex
	logger   logger.Logger
}

func NewNode(genesis *genesis.Genesis) (*Node, error) {
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return NewNodeWithKey(genesis, key)
}

func NewNodeWithKey(genesis *genesis.Genesis, key ed25519.PrivateKey) (*Node, error) {
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
		blockPool:    pool.NewBlockPool(),
		lastBlockNum: 0,
		peers:        make(map[string]connectedPeer, 0),
		state:        state,

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

	out := make(chan msg.Message, MessagesBusLen)
	in := peer.Connection(c.address, out)
	c.Connection(remoteAddress, in, out)
	return nil
}

func (c *Node) Broadcast(ctx context.Context, msg msg.Message) {
	c.mxPeers.Lock()
	defer c.mxPeers.Unlock()

	for _, v := range c.peers {
		if v.Address != c.address {
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

// AddTransaction - nothing for simple node
func (c *Node) AddTransaction(tr msg.Transaction) error {
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
	case msg.Transaction:
		return c.processTransaction(peer, m)
	case msg.Block:
		return c.processBlockMessage(ctx, peer, m)
	case msg.BlocksRequest:
		return c.processBlockRequest(ctx, peer, m)
	case msg.NodeInfoResp:
		return c.processNodeInfo(ctx, peer, m)
	}
	return nil
}

// processTransaction - received transaction
func (c *Node) processTransaction(peer connectedPeer, tr msg.Transaction) error {
	// check public key
	addr, err := PubKeyToAddress(tr.PubKey)
	if err != nil {
		return fmt.Errorf("can't convert peer adress to key: %v", err)
	}
	if addr != peer.Address {
		return errors.New("transaction key not belong to address")
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

	return c.AddTransaction(tr)
}

// processBlock - received block
func (c *Node) processBlockMessage(ctx context.Context, peer connectedPeer, block msg.Block) error {
	c.logger.Infof("%v receive block [%v] from %v",
		simplifyAddress(c.address), simplifyAddress(block.BlockHash), simplifyAddress(peer.Address))

	// if the block is out of turn
	if block.BlockNum > c.lastBlockNum+1 {
		return c.blockPool.AddBlock(block)
	}

	// process block from message
	if err := c.processBlock(block); err != nil {
		return fmt.Errorf("can't process message: %v", err)
	}

	// check block pool for blocks
	if c.blockPool.HasBlockNum(c.lastBlockNum + 1) {
		block, err := c.blockPool.PopBlock(c.lastBlockNum + 1)
		if err != nil {
			return fmt.Errorf("can't process block pool: %v", err)
		}
		return c.processBlock(block)
	}

	return nil
}

func (c *Node) processBlock(block msg.Block) error {
	if err := c.verifyBlock(block); err != nil {
		return fmt.Errorf("can't process block: %v", err)
	}

	if err := c.insertBlock(block); err != nil {
		return fmt.Errorf("can't process block: %v", err)
	}

	c.logger.Infof("%v insert new block [%v]", simplifyAddress(c.address), simplifyAddress(block.BlockHash))
	return nil
}

// send blocks to peer that requested
func (c *Node) processBlockRequest(ctx context.Context, peer connectedPeer, req msg.BlocksRequest) error {
	// if my request
	if req.From == c.NodeAddress() {
		for id := req.BlockNumFrom; id <= req.BlockNumTo; id++ {
			c.logger.Infof("%v send block [%v] to %v",
				simplifyAddress(c.address), simplifyAddress(c.blocks[id].BlockHash), simplifyAddress(peer.Address))
			c.SendTo(peer, ctx, c.GetBlockByNumber(id))
		}
	}
	return nil
}

// get info from another peer
func (c *Node) processNodeInfo(ctx context.Context, peer connectedPeer, res msg.NodeInfoResp) error {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()
	// blocks request
	if c.lastBlockNum < res.BlockNum {
		c.logger.Infof("%v connect to %v need sync", simplifyAddress(c.address), simplifyAddress(peer.Address))
		c.SendTo(peer, ctx, msg.BlocksRequest{
			From:         res.NodeName,
			To:           c.NodeAddress(),
			BlockNumFrom: c.lastBlockNum + 1,
			BlockNumTo:   res.BlockNum,
		})
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
	if block.BlockNum < 0 {
		return errors.New("incorrect block num")
	}
	if block.BlockNum <= c.lastBlockNum {
		return fmt.Errorf("already have block [%v <= %v]", block.BlockNum, c.lastBlockNum)
	}
	if len(block.Transactions) == 0 {
		return errors.New("no coinbase transaction")
	}

	validatorAddr, err := PubKeyToAddress(block.PubKey)
	if err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}

	// verify transactions
	stateCopy := c.state.Copy()
	for _, tr := range block.Transactions[1:] {
		if err := verifyTransaction(stateCopy, tr); err != nil {
			return fmt.Errorf("can't verify block: %v", err)
		}
		if err := applyTransaction(stateCopy, validatorAddr, tr); err != nil {
			return fmt.Errorf("can't verify block: %v", err)
		}
	}
	// verify coinbase transaction
	coinbase := block.Transactions[0]
	if coinbase.From != "" || coinbase.Amount != BlockReward {
		return errors.New("wrong coinbase transaction")
	}
	if err := applyCoinbaseTransaction(stateCopy, coinbase); err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}

	// verify state hash
	stateHash, err := stateCopy.Hash()
	if err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}
	if stateHash != block.StateHash {
		return errors.New("state hash is incorrect")
	}

	// check signature
	sig := block.Signature
	key := block.PubKey

	block.Signature = nil
	block.PubKey = nil

	bts, err := block.Bytes()
	if err != nil {
		return fmt.Errorf("can't verify block: %v", err)
	}
	if !ed25519.Verify(key, bts, sig) {
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
	return nil
}

func (c *Node) insertBlock(b msg.Block) error {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()

	validatorAddr, err := PubKeyToAddress(b.PubKey)
	if err != nil {
		return err
	}

	for _, tr := range b.Transactions[1:] {
		err := applyTransaction(c.state, validatorAddr, tr)
		if err != nil {
			return err
		}
	}

	err = applyCoinbaseTransaction(c.state, b.Transactions[0])
	if err != nil {
		return err
	}

	c.blocks = append(c.blocks, b)
	c.lastBlockNum += 1
	return nil
}

func applyCoinbaseTransaction(state storage.Storage, tr msg.Transaction) error {
	state.Lock()
	defer state.Unlock()

	err := state.Add(tr.To, tr.Amount)
	if err != nil {
		return err
	}

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
