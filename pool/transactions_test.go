package pool

import (
	"simple_chain/msg"
	"sync"
	"testing"
	"time"
)

func TestTransactionPool_Pop(t *testing.T) {

	txPool := NewTransactionPool()
	_ = txPool.Insert(msg.Transaction{From: "one", To: "two", Amount: 1})
	_ = txPool.Insert(msg.Transaction{From: "one", To: "two", Amount: 2})
	_ = txPool.Insert(msg.Transaction{From: "one", To: "two", Amount: 3})
	_ = txPool.Insert(msg.Transaction{From: "one", To: "two", Amount: 4})
	_ = txPool.Insert(msg.Transaction{From: "one", To: "two", Amount: 5})

	txs := txPool.Peek(3)
	if len(txs) != 3 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 3, len(txs))
	}
	if len(txPool.alloc) != 5 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 5, len(txPool.alloc))
	}

	txPool.Delete(msg.Transaction{From: "one", To: "two", Amount: 1})
	txPool.Delete(msg.Transaction{From: "one", To: "two", Amount: 2})
	txPool.Delete(msg.Transaction{From: "one", To: "two", Amount: 3})

	txs = txPool.Peek(2)
	if len(txs) != 2 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 2, len(txs))
	}
	if len(txPool.alloc) != 2 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 2, len(txPool.alloc))
	}
}

func TestTransactionPool_lifeTime(t *testing.T) {

	txPool := NewTransactionPool()
	txPool.maxLifeTime = int64(time.Millisecond)

	_ = txPool.Insert(msg.Transaction{From: "one", To: "two", Amount: 1})
	time.Sleep(time.Millisecond)
	_ = txPool.Insert(msg.Transaction{From: "one", To: "two", Amount: 2})

	txs := txPool.Peek(2)
	if len(txs) != 1 {
		t.Fatalf("too many transactions: get=%v, want=%v", len(txs), 1)
	}
	if txs[0].Amount != 2 {
		t.Fatalf("wrong transaction was received: amount get=%v, want=%v", txs[0].Amount, 2)
	}
}

// run with -race
func TestTransactionPool_Concurrency(t *testing.T) {

	tp := NewTransactionPool()
	wg := sync.WaitGroup{}

	wg.Add(3)

	go func() {
		defer wg.Done()
		_ = tp.Insert(msg.Transaction{})
	}()

	go func() {
		defer wg.Done()
		tp.Delete(msg.Transaction{})
	}()

	go func() {
		defer wg.Done()
		txs := tp.Peek(1)
		t.Logf("peek transactions: %v", txs)
	}()

	wg.Wait()
}
