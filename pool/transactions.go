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

type TransactionPool map[string]pooledTransaction //transaction hash - > transaction

func NewTransactionPool() TransactionPool {
	return make(map[string]pooledTransaction)
}

func (p TransactionPool) Insert(tr msg.Transaction) error {
	hash, err := tr.Hash()
	if err != nil {
		return err
	}
	p[hash] = pooledTransaction{tr, time.Now().Unix()}
	return nil
}

// Pop - remove first 0-n transactions from transaction pool
func (p TransactionPool) Pop(maxCount uint64) []msg.Transaction {
	txs := make([]msg.Transaction, 0, maxCount)
	i := uint64(0)

	for hash, tr := range p {
		if i++; i > maxCount {
			break
		}
		if p.LifeTime(hash) < MaxLifeTime {
			txs = append(txs, tr.Transaction)
			p.Delete(hash)
		}
	}
	return txs
}

func (p TransactionPool) LifeTime(hash string) int64 {
	return time.Now().Unix() - p[hash].Timestamp
}

func (p TransactionPool) Delete(hash string) {
	delete(p, hash)
}
