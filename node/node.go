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
	"simple_chain/log"
	"simple_chain/msg"
	"simple_chain/pool"
	"simple_chain/storage"
	"sync"
	"time"
)

const (
	MessagesBusLen                = 100
	TransactionFee                = 10
	TransactionSuccessBlocksDelta = 6
	MessageSendTimeout            = time.Second * 3
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
	genesis      genesis.Genesis
	lastBlockNum uint64

	//chain
	blocks    []msg.Block
	blockPool pool.BlockPool
	txsPool   pool.TransactionPool
	//peer address -> peer info
	peers map[string]connectedPeer
	//peer address -> fund
	state storage.Storage

	mxPeers  *sync.Mutex
	mxBlocks *sync.Mutex
	logger   log.Logger
}

func NewNode(genesis genesis.Genesis) (*Node, error) {
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return NewNodeWithKey(genesis, key)
}

func NewNodeWithKey(genesis genesis.Genesis, key ed25519.PrivateKey) (*Node, error) {
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
		txsPool:      pool.NewTransactionPool(),
		lastBlockNum: 0,
		peers:        make(map[string]connectedPeer, 0),
		state:        state,

		mxBlocks: &(sync.Mutex{}),
		mxPeers:  &(sync.Mutex{}),
		logger:   log.New(log.Debug + log.Chain),
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
		// There is no verification that the message does not belong
		// to the receiver, because the validator waits until the block
		// returns from the network before adding it to its chain and
		// changing the state.
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

// AddTransaction - add verified ! transaction to transaction pool
func (c *Node) AddTransaction(tr msg.Transaction) error {
	return c.txsPool.Insert(tr)
}

func (c *Node) getBlockByNumber(ID uint64) (msg.Block, error) {
	if ID >= uint64(len(c.blocks)) {
		return msg.Block{}, errors.New("block id is too big")
	}
	return c.blocks[ID], nil
}

func (c *Node) getBlockByHash(hash string) (msg.Block, error) {
	for _, block := range c.blocks {
		blockHash, err := block.Hash()
		if err != nil {
			return msg.Block{}, err
		}
		if hash == blockHash {
			return block, nil
		}
	}
	return msg.Block{}, fmt.Errorf("no blocks with hash: %v", hash)
}

func (c *Node) NodeInfo() msg.NodeInfoResp {
	return msg.NodeInfoResp{
		NodeName:        c.address,
		BlockNum:        c.lastBlockNum,
		TotalDifficulty: c.totalDifficulty(),
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
		From: c.NodeAddress(),
		Data: data,
	}
	c.SendMessageTo(cp, ctx, m)
}

func (c *Node) SendMessageTo(cp connectedPeer, ctx context.Context, msg msg.Message) {
	select {
	case cp.Out <- msg:
	case <-ctx.Done():
	case <-time.After(MessageSendTimeout):
	}
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
			broadcast, err := c.processMessage(ctx, peer, message)
			if err != nil {
				c.logger.Errorf("%v process peer error: %v", log.Simplify(c.address), err)
				continue
			}
			if broadcast {
				c.Broadcast(ctx, message)
			}
		}
	}
}

func (c *Node) processMessage(ctx context.Context, peer connectedPeer, message msg.Message) (bool, error) {
	switch m := message.Data.(type) {
	case msg.Transaction:
		return !c.txsPool.Has(m), c.processTransaction(peer, m)
	case msg.BlockMessage:
		return !c.hasBlock(m.Block), c.processBlockMessage(ctx, peer, m)
	case msg.BlocksRequest:
		return m.To != c.NodeAddress(), c.processBlocksRequest(ctx, peer, m)
	case msg.BlocksResponse:
		return m.To != c.NodeAddress(), c.processBlocksResponse(ctx, peer, m)
	case msg.NodeInfoResp:
		return false, c.processNodeInfo(ctx, peer, m)
	}
	return false, nil
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
func (c *Node) processBlockMessage(ctx context.Context, peer connectedPeer, blockMsg msg.BlockMessage) error {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()

	c.logger.Infof("%v receive block [%v] from %v",
		log.Simplify(c.address), log.Simplify(blockMsg.BlockHash), log.Simplify(peer.Address))

	// check if block num is too high
	if blockMsg.BlockNum > c.lastBlockNum+1 {
		return c.blockPool.Insert(blockMsg.Block)
	}

	// check for reorgs and greater total difficulty
	if c.isReorg(blockMsg) {
		c.logger.Debugf("%v reorg was found with block [%v]",
			log.Simplify(c.NodeAddress()), log.Simplify(blockMsg.BlockHash))
		// revert `incorrect` block
		if err := c.revertLastBlock(); err != nil {
			c.logger.Errorf("can't revert block: %v", err)
		}
		// if td - request some blocks
		c.SendTo(peer, ctx, msg.BlocksRequest{
			To:            peer.Address,
			LastBlockHash: c.lastBlockHash(),
		})
		return nil
	}

	// process block from message
	if err := c.processBlock(blockMsg.Block); err != nil {
		c.logger.Errorf("%v can't process block [%v] from %v: %v",
			log.Simplify(c.address), log.Simplify(blockMsg.BlockHash), log.Simplify(peer.Address), err)
		return fmt.Errorf("can't process message: %v", err)
	}

	return c.processBlockPool()
}

func (c *Node) processBlock(block msg.Block) error {
	if err := c.verifyBlock(block); err != nil {
		return fmt.Errorf("can't process block: %v", err)
	}

	if err := c.insertBlock(block); err != nil {
		return fmt.Errorf("can't process block: %v", err)
	}

	c.logger.Infof("%v insert new block [%v]", log.Simplify(c.address), log.Simplify(block.BlockHash))
	return nil
}

func (c *Node) processBlockPool() error {
	index := c.lastBlockNum
	for {
		// next index
		index++
		inserted := false
		// get block slice if pool contains
		blocks, err := c.blockPool.GetBlocks(index)
		if err != nil {
			// no blocks - nothing to do
			return nil
		}
		// for each block try to insert
		for _, block := range blocks {
			if err := c.processBlock(block); err == nil {
				inserted = true
				break
			}
		}
		// if nothing inserted - return
		if !inserted {
			return nil
		}
	}
}

func (c *Node) processBlocksResponse(ctx context.Context, peer connectedPeer, m msg.BlocksResponse) error {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()

	if c.NodeAddress() == m.To && m.Error != nil {
		c.logger.Infof("%v has no block with hash [%v]", log.Simplify(peer.Address), log.Simplify(m.BlockHash))
		c.logger.Infof("%v revert block [%v]", log.Simplify(c.NodeAddress()), log.Simplify(c.lastBlockHash()))

		if err := c.revertLastBlock(); err != nil {
			c.logger.Errorf("can't revert block: %v", err)
		}

		c.SendTo(peer, ctx, msg.BlocksRequest{
			To:            peer.Address,
			LastBlockHash: c.lastBlockHash(),
		})
	}
	return nil
}

// send blocks to peer that requested
func (c *Node) processBlocksRequest(ctx context.Context, peer connectedPeer, req msg.BlocksRequest) error {
	c.logger.Infof("%v blocks request from %v, from block [%v]",
		log.Simplify(c.address), log.Simplify(peer.Address), log.Simplify(req.LastBlockHash))

	if c.NodeAddress() == req.To {
		fromBlock, err := c.getBlockByHash(req.LastBlockHash)
		if err != nil {
			c.SendTo(peer, ctx, msg.BlocksResponse{
				To:        peer.Address,
				BlockHash: req.LastBlockHash,
				Error:     err,
			})
			return nil
		}

		for id := fromBlock.BlockNum + 1; id <= c.lastBlockNum; id++ {
			c.logger.Infof("%v send block [%v] to %v",
				log.Simplify(c.address), log.Simplify(c.blocks[id].BlockHash), log.Simplify(peer.Address))

			block, err := c.getBlockByNumber(id)
			if err != nil {
				return fmt.Errorf("can't get block by ID: %v", err)
			}

			c.SendTo(peer, ctx, msg.BlockMessage{
				Block:           block,
				TotalDifficulty: c.totalDifficulty(),
			})
		}
	}
	return nil
}

// get info from another peer
func (c *Node) processNodeInfo(ctx context.Context, peer connectedPeer, res msg.NodeInfoResp) error {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()

	if c.totalDifficulty() < res.TotalDifficulty {
		c.logger.Infof("%v connect to %v need sync", log.Simplify(c.address), log.Simplify(peer.Address))
		c.SendTo(peer, ctx, msg.BlocksRequest{
			To:            peer.Address,
			LastBlockHash: c.lastBlockHash(),
		})
	}
	return nil
}

func (c *Node) isReorg(msg msg.BlockMessage) bool {

	prevBlock, err := c.getBlockByNumber(c.lastBlockNum - 1)

	diffPrev := (msg.BlockNum == c.lastBlockNum-1) && err == nil && prevBlock.BlockHash != msg.BlockHash
	diffLast := (msg.BlockNum == c.lastBlockNum) && (c.lastBlockHash() != msg.BlockHash)
	diffNext := (msg.BlockNum == c.lastBlockNum+1) && (c.lastBlockHash() != msg.PrevBlockHash)

	diffTd := c.totalDifficulty() < msg.TotalDifficulty

	return diffTd && (diffPrev || diffLast || diffNext)
}

/* -- Common -------------------------------------------------------------------------------------------------------- */

func PubKeyToAddress(key crypto.PublicKey) (string, error) {
	if v, ok := key.(ed25519.PublicKey); ok {
		b := sha256.Sum256(v)
		return hex.EncodeToString(b[:]), nil
	}
	return "", errors.New("incorrect key")
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
	// check block num
	if block.BlockNum < 0 {
		return errors.New("incorrect block num")
	}
	if block.BlockNum <= c.lastBlockNum {
		return fmt.Errorf("already has block [%v <= %v]", block.BlockNum, c.lastBlockNum)
	}

	// check parent hash
	prevBlock, err := c.getBlockByNumber(block.BlockNum - 1)
	if err != nil {
		return fmt.Errorf("no previous block")
	}
	prevBlockHash := prevBlock.BlockHash
	if prevBlockHash != block.PrevBlockHash {
		return errors.New("parent hash is incorrect")
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
	// and changes node state
	c.state.Lock()
	defer c.state.Unlock()

	validatorAddr, err := PubKeyToAddress(b.PubKey)
	if err != nil {
		return err
	}

	c.state.PutBlockToHistory(b.BlockNum)
	if len(b.Transactions) > 1 {
		for _, tr := range b.Transactions[1:] {
			err := applyTransaction(c.state, validatorAddr, tr)
			if err != nil {
				return err
			}
		}
	}

	err = applyCoinbaseTransaction(c.state, b.Transactions[0])
	if err != nil {
		return err
	}

	c.blocks = append(c.blocks, b)
	c.lastBlockNum += 1

	c.logger.Chain(c.NodeAddress(), c.blocks)
	return nil
}

func (c *Node) IsTransactionSuccess(tr msg.Transaction) bool {
	hash, err := tr.Hash()
	if err != nil {
		return false
	}

	for _, block := range c.blocks {
		for _, blockTr := range block.Transactions {
			blockTrHash, err := blockTr.Hash()
			if err != nil {
				continue
			}
			if hash == blockTrHash {
				return c.lastBlockNum-block.BlockNum >= TransactionSuccessBlocksDelta
			}
		}
	}

	return false
}

func (c *Node) totalDifficulty() uint64 {
	return uint64(len(c.blocks))
}

func (c *Node) lastBlockHash() string {
	lastBlock := c.blocks[len(c.blocks)-1]
	return lastBlock.BlockHash
}

func (c *Node) revertLastBlock() error {
	if len(c.blocks) == 0 {
		return errors.New("nothing to revert")
	}
	if len(c.blocks) == 1 {
		return errors.New("genesis reverting")
	}

	c.state.RevertBlock()
	c.blocks = c.blocks[:len(c.blocks)-1]
	c.lastBlockNum--

	c.logger.Chain(c.NodeAddress(), c.blocks)
	return nil
}

func (c *Node) hasBlock(m msg.Block) bool {
	c.mxBlocks.Lock()
	defer c.mxBlocks.Unlock()

	hash, err := m.Hash()
	if err != nil {
		return false
	}

	for _, block := range c.blocks {
		blockHash, err := block.Hash()
		if err != nil {
			continue
		}
		if hash == blockHash {
			return true
		}
	}
	return false
}

func applyCoinbaseTransaction(state storage.Storage, tr msg.Transaction) error {
	err := state.PutOrAdd(tr.To, tr.Amount)
	if err != nil {
		return err
	}
	return nil
}

func applyTransaction(state storage.Storage, validatorAddress string, tr msg.Transaction) error {
	err := state.Sub(tr.From, tr.Amount+tr.Fee)
	if err != nil {
		return err
	}
	err = state.Add(tr.To, tr.Amount)
	if err != nil {
		return err
	}
	err = state.PutOrAdd(validatorAddress, tr.Fee)
	if err != nil {
		return err
	}
	return nil
}
