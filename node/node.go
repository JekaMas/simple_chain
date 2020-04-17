package node

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log"
	"simple_chain"
	"simple_chain/storage"
)

const (
	MessagesBusLen = 100
)

type Node struct {
	key          ed25519.PrivateKey
	address      string
	genesis      bc.Genesis
	lastBlockNum uint64

	//state
	blocks []bc.Block
	//peer address - > peer info
	peers map[string]connectedPeer
	//peer address -> fund
	state      storage.Storage
	validators []crypto.PublicKey
}

func NewNode(key ed25519.PrivateKey, genesis bc.Genesis) (*Node, error) {
	address, err := bc.PubKeyToAddress(key.Public())
	if err != nil {
		return nil, err
	}
	return &Node{
		key:          key,
		address:      address,
		genesis:      genesis,
		blocks:       []bc.Block{genesis.ToBlock()},
		lastBlockNum: 0,
		peers:        make(map[string]connectedPeer, 0),
		state:        storage.NewMap(),
		validators:   genesis.Validators,
	}, nil
}

/* --- Interface ---------------------------------------------------------------------------------------------------- */

type connectedPeer struct {
	Address string
	In      chan bc.Message
	Out     chan bc.Message
	cancel  context.CancelFunc
}

func (c *Node) NodeKey() crypto.PublicKey {
	return c.key.Public()
}

func (c *Node) Connection(address string, in chan bc.Message, outs ...chan bc.Message) chan bc.Message {
	var out chan bc.Message
	if len(outs) == 0 {
		out = make(chan bc.Message, MessagesBusLen)
	} else {
		out = outs[0]
	}

	ctx, cancel := context.WithCancel(context.Background())
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
	remoteAddress, err := bc.PubKeyToAddress(peer.NodeKey())
	if err != nil {
		return err
	}

	if c.address == remoteAddress {
		return errors.New("self connection")
	}

	if _, ok := c.peers[remoteAddress]; ok {
		return nil
	}

	out := make(chan bc.Message, MessagesBusLen)
	in := peer.Connection(c.address, out)
	c.Connection(remoteAddress, in, out)
	return nil
}

func (c *Node) Broadcast(ctx context.Context, msg bc.Message) {
	for _, v := range c.peers {
		if msg.From != v.Address && v.Address != c.address {
			c.SendTo(v, ctx, msg)
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

func (c *Node) AddTransaction(tr bc.Transaction) error {
	/* nothing */
	return nil
}

func (c *Node) GetBlockByNumber(ID uint64) bc.Block {
	// todo make check and other stuff
	return c.blocks[ID]
}

func (c *Node) NodeInfo() bc.NodeInfoResp {
	lastBlockBytes, err := bc.Bytes(c.blocks[len(c.blocks)-1])
	if err != nil {
		panic("can't convert block to bytes")
	}

	return bc.NodeInfoResp{
		NodeName:        c.address,
		BlockNum:        c.lastBlockNum,
		LastBlockHash:   bc.Hash(lastBlockBytes),
		TotalDifficulty: 1, // todo totalDifficult
	}
}

func (c *Node) NodeAddress() string {
	return c.address
}

func (c *Node) SignTransaction(transaction bc.Transaction) (bc.Transaction, error) {
	b, err := transaction.Bytes()
	if err != nil {
		return bc.Transaction{}, err
	}

	transaction.Signature = ed25519.Sign(c.key, b)
	return transaction, nil
}

func (c *Node) SendTo(cp connectedPeer, ctx context.Context, data interface{}) {
	// todo timeout using context + done check
	m := bc.Message{
		From: c.address,
		Data: data,
	}
	cp.Out <- m
}

/* --- Processes ---------------------------------------------------------------------------------------------------- */

func (c *Node) peerLoop(ctx context.Context, peer connectedPeer) {
	// handshake
	c.SendTo(peer, ctx, c.NodeInfo())

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-peer.In:
			err := c.processMessage(ctx, peer, msg)
			if err != nil {
				log.Println("Process peer error", err)
				continue
			}
			// broadcast to connected peers
			c.Broadcast(ctx, msg)
		}
	}
}

func (c *Node) processMessage(ctx context.Context, peer connectedPeer, msg bc.Message) error {
	switch m := msg.Data.(type) {
	// get transaction from another peer
	case bc.Transaction:
		// add all transaction to transaction pool (for validator)
		return c.AddTransaction(m)
	// received block
	case bc.Block:
		fmt.Println(bc.SimplifyAddress(c.address), "receive block from", bc.SimplifyAddress(peer.Address))
		block := msg.Data.(bc.Block)
		if c.checkBlock(block) {
			err := c.insertBlock(block)
			if err != nil {
				return err
			}
		}
	// send blocks to peer that requested
	case bc.BlocksRequest:
		req := msg.Data.(bc.BlocksRequest)
		for id := req.BlockNumFrom; id < req.BlockNumTo; id++ {
			fmt.Println(bc.SimplifyAddress(c.address), "send block [", id, "] to", bc.SimplifyAddress(peer.Address))
			c.SendTo(peer, ctx, c.GetBlockByNumber(id))
		}
	// get info from another peer
	case bc.NodeInfoResp:
		needSync := c.lastBlockNum < m.BlockNum
		fmt.Println(bc.SimplifyAddress(c.address), "connected to", bc.SimplifyAddress(peer.Address), "need sync", needSync)
		if needSync { // blocks request
			c.SendTo(peer, ctx, bc.BlocksRequest{
				BlockNumFrom: c.lastBlockNum,
				BlockNumTo:   m.BlockNum,
			})
		}
	}
	return nil
}

/* -- Common -------------------------------------------------------------------------------------------------------- */

/*
type Block struct {
	BlockNum      uint64
	Timestamp     int64
	Transactions  []Transaction
	BlockHash     string `json:"-"`
	PrevBlockHash string
	StateHash     string
	Signature     []byte `json:"-"`
}
*/
func (c *Node) checkBlock(block bc.Block) bool {
	return true
}

func (c *Node) validatorAddr(b bc.Block) (string, error) {
	validatorKey := c.validators[int(b.BlockNum%uint64(len(c.validators)))]
	return bc.PubKeyToAddress(validatorKey)
}

func (c *Node) insertBlock(b bc.Block) error {
	for _, v := range b.Transactions {
		c.state.Sub(v.From, v.Amount+v.Fee)
		c.state.Add(v.To, v.Amount)

		validatorAddr, err := c.validatorAddr(b)
		if err != nil {
			return err
		}
		c.state.Add(validatorAddr, v.Fee)
	}

	c.blocks = append(c.blocks, b)
	c.lastBlockNum += 1
	return nil
}
