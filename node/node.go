package node

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"simple_chain/encode"
	"simple_chain/genesis"
	"simple_chain/msg"
	"simple_chain/storage"
)

const (
	MessagesBusLen = 100
)

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
}

func NewNode(key ed25519.PrivateKey, genesis *genesis.Genesis) (*Node, error) {
	address, err := PubKeyToAddress(key.Public())
	if err != nil {
		return nil, err
	}
	return &Node{
		key:          key,
		address:      address,
		genesis:      genesis,
		blocks:       []msg.Block{genesis.ToBlock()},
		lastBlockNum: 0,
		peers:        make(map[string]connectedPeer, 0),
		state:        storage.NewMap(),
		validators:   genesis.Validators,
	}, nil
}

/* --- Interface ---------------------------------------------------------------------------------------------------- */

type connectedPeer struct {
	Address string
	In      chan msg.Message
	Out     chan msg.Message
	cancel  context.CancelFunc
}

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

	//if v, ok := peer.(Node); ok {
	//
	//}

	out := make(chan msg.Message, MessagesBusLen)
	in := peer.Connection(c.address, out)
	c.Connection(remoteAddress, in, out)
	return nil
}

func (c *Node) Broadcast(ctx context.Context, msg msg.Message) {
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

func (c *Node) AddTransaction(tr msg.Transaction) error {
	/* nothing */
	return nil
}

func (c *Node) GetBlockByNumber(ID uint64) msg.Block {
	// todo make check and other stuff
	return c.blocks[ID]
}

func (c *Node) NodeInfo() msg.NodeInfoResp {
	lastBlockBytes, err := encode.Bytes(c.blocks[len(c.blocks)-1])
	if err != nil {
		panic("can't convert block to bytes")
	}

	return msg.NodeInfoResp{
		NodeName:        c.address,
		BlockNum:        c.lastBlockNum,
		LastBlockHash:   encode.Hash(lastBlockBytes),
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
	// todo timeout using context + done check
	m := msg.Message{
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

func (c *Node) processMessage(ctx context.Context, peer connectedPeer, message msg.Message) error {
	switch m := message.Data.(type) {
	// get transaction from another peer
	case msg.Transaction:
		// add all transaction to transaction pool (for validator)
		return c.AddTransaction(m)
	// received block
	case msg.Block:
		fmt.Println(simplifyAddress(c.address), "receive block from", simplifyAddress(peer.Address))
		block := message.Data.(msg.Block)
		if c.checkBlock(block) {
			err := c.insertBlock(block)
			if err != nil {
				return err
			}
		}
	// send blocks to peer that requested
	case msg.BlocksRequest:
		req := message.Data.(msg.BlocksRequest)
		for id := req.BlockNumFrom; id < req.BlockNumTo; id++ {
			fmt.Println(simplifyAddress(c.address), "send block [", id, "] to", simplifyAddress(peer.Address))
			c.SendTo(peer, ctx, c.GetBlockByNumber(id))
		}
	// get info from another peer
	case msg.NodeInfoResp:
		needSync := c.lastBlockNum < m.BlockNum
		fmt.Println(simplifyAddress(c.address), "connected to", simplifyAddress(peer.Address), "need sync", needSync)
		if needSync { // blocks request
			c.SendTo(peer, ctx, msg.BlocksRequest{
				BlockNumFrom: c.lastBlockNum,
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
	return address[:4]
}

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
func (c *Node) checkBlock(block msg.Block) bool {
	return true
}

func (c *Node) validatorAddr(b msg.Block) (string, error) {
	validatorKey := c.validators[int(b.BlockNum%uint64(len(c.validators)))]
	return PubKeyToAddress(validatorKey)
}

func (c *Node) insertBlock(b msg.Block) error {
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
