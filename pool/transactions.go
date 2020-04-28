package pool

import (
	"simple_chain/msg"
	"time"
)

const (
	MaxLifeTime = 1 * 60 * 1000 // ms
)

type pooledTransaction struct {
	msg.Transaction
	Timestamp int64
}

type TransactionPool struct {
	alloc       map[string]pooledTransaction //transaction hash - > transaction
	maxLifeTime int64
}

func NewTransactionPool() TransactionPool {
	return TransactionPool{
		alloc:       make(map[string]pooledTransaction),
		maxLifeTime: MaxLifeTime,
	}
}

func (p TransactionPool) Insert(tr msg.Transaction) error {
	hash, err := tr.Hash()
	if err != nil {
		return err
	}
	p.alloc[hash] = pooledTransaction{tr, time.Now().UnixNano()}
	return nil
}

// Peek - get and not delete first 0-n transactions from transaction pool
func (p TransactionPool) Peek(maxCount uint64) []msg.Transaction {
	txs := make([]msg.Transaction, 0, maxCount)
	i := uint64(0)

	for hash, tr := range p.alloc {
		if i++; i > maxCount {
			break
		}
		if p.LifeTime(hash) < p.maxLifeTime {
			txs = append(txs, tr.Transaction)
		}
	}
	return txs
}

func (p TransactionPool) LifeTime(hash string) int64 {
	return time.Now().UnixNano() - p.alloc[hash].Timestamp
}

func (p TransactionPool) Delete(tr msg.Transaction) {
	hash, _ := tr.Hash()
	delete(p.alloc, hash)
}
