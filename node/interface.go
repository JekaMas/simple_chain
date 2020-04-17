package node

import (
	"crypto"
	"simple_chain"
)

type Blockchain interface {
	NodeKey() crypto.PublicKey
	NodeAddress() string
	Connection(address string, in chan bc.Message, out ...chan bc.Message) chan bc.Message
	PublicAPI
}

type PublicAPI interface {
	//network
	AddPeer(Blockchain) error
	RemovePeer(Blockchain) error

	//for clients
	GetBalance(account string) (uint64, error)
	//add to transaction pool
	AddTransaction(transaction bc.Transaction) error
	SignTransaction(transaction bc.Transaction) (bc.Transaction, error)

	//sync
	GetBlockByNumber(ID uint64) bc.Block
	NodeInfo() bc.NodeInfoResp
}
