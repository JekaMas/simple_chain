package bc

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log"
	"simple_chain/storage"
)

const MSGBusLen = 100

func NewNode(key ed25519.PrivateKey, genesis Genesis) (*Node, error) {
	address, err := PubKeyToAddress(key.Public())
	if err != nil {
		return nil, err
	}
	return &Node{
		key:             key,
		address:         address,
		genesis:         genesis,
		blocks:          []Block{genesis.ToBlock()},
		lastBlockNum:    0,
		peers:           make(map[string]connectedPeer, 0),
		state:           storage.NewMap(),
		transactionPool: make(map[string]Transaction),
	}, err
}

type Node struct {
	key          ed25519.PrivateKey
	address      string
	genesis      Genesis
	lastBlockNum uint64

	//state
	blocks []Block
	//peer address - > peer info
	peers map[string]connectedPeer
	//peer address -> fund
	state storage.Storage
	validators []ed25519.PublicKey

	//transaction hash - > transaction
	transactionPool map[string]Transaction
}

func (c *Node) NodeKey() crypto.PublicKey {
	return c.key.Public()
}

func (c *Node) Connection(address string, in chan Message) chan Message {
	out := make(chan Message, MSGBusLen)
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

	out := make(chan Message, MSGBusLen)
	in := peer.Connection(c.address, out)
	c.Connection(remoteAddress, in)
	return nil
}

func (c *Node) peerLoop(ctx context.Context, peer connectedPeer) {
	//todo handshake
	peer.Send(ctx, Message{
		From: c.address,
		Data: c.NodeInfo(),
	})

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-peer.In:
			err := c.processMessage(peer.Address, msg)
			if err != nil {
				log.Println("Process peer error", err)
				continue
			}

			//broadcast to connected peers
			c.Broadcast(ctx, msg)
		}
	}
}

func (c *Node) processMessage(address string, msg Message) error {
	switch m := msg.Data.(type) {
	// get info from another peer
	case NodeInfoResp:
		needSync := c.lastBlockNum < m.BlockNum
		fmt.Println(simplifyAddress(c.address), "connected to ", simplifyAddress(address), "need sync", needSync)

		if needSync {
			// todo
		}
	}
	return nil
}

func (c *Node) Broadcast(ctx context.Context, msg Message) {
	for _, v := range c.peers {
		if v.Address != c.address {
			v.Send(ctx, msg)
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

func (c *Node) AddTransaction(tr Transaction) error {
	if c.isValidator() {
		hash, err := tr.Hash()
		if err != nil {
			return err
		}
		c.transactionPool[hash] = tr
	}
	// todo broadcast transaction?
	return nil
}

func (c *Node) GetBlockByNumber(ID uint64) Block {
	// todo make check and other stuff
	return c.blocks[ID]
}

func (c *Node) NodeInfo() NodeInfoResp {
	lastBlockBytes, err := Bytes(c.blocks[len(c.blocks)-1])
	if err != nil {
		panic("can't convert block to bytes")
	}

	return NodeInfoResp{
		NodeName: c.address,
		BlockNum: c.lastBlockNum,
		LastBlockHash: Hash(lastBlockBytes),
		TotalDifficulty: 1, // todo totalDifficult
	}
}

func (c *Node) NodeAddress() string {
	return c.address
}

func (c *Node) SignTransaction(transaction Transaction) (Transaction, error) {
	b, err := transaction.Bytes()
	if err != nil {
		return Transaction{}, err
	}

	transaction.Signature = ed25519.Sign(c.key, b)
	return transaction, nil
}

/* -- Common -------------------------------------------------------------------------------------------------------- */

func simplifyAddress(address string) string {
	return address[:4]
}

func (c *Node) isValidator() bool {
	for _, key := range c.genesis.Validators {
		bts1, _ := Bytes(key)
		bts2, _ := Bytes(c.key.Public())
		if bytes.Equal(bts1, bts2) {
			return true
		}
	}
	return false
}
