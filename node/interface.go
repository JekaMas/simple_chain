package node

import (
	"crypto"

	"../msg"
)

type Blockchain interface {
	NodeKey() crypto.PublicKey
	NodeAddress() string
	Connection(address string, in chan msg.Message, out ...chan msg.Message) chan msg.Message
	PublicAPI
}

type PublicAPI interface {
	// network
	AddPeer(Blockchain) error
	RemovePeer(Blockchain) error

	// for clients
	GetBalance(account string) (uint64, error)
	// add to transaction pool
	AddTransaction(transaction msg.Transaction) error
	SignTransaction(transaction msg.Transaction) (msg.Transaction, error)

	// sync
	// todo: в чем смысл приватных методов в интерфейсе?
	//getBlockByNumber(ID uint64) msg.Block
	NodeInfo() msg.NodeInfoResp
}
