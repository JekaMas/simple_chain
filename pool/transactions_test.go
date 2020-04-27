package pool

import (
	"simple_chain/msg"
	"simple_chain/storage"
	"testing"
)

func TestTransactionPool_Pop(t *testing.T) {
	vd := Validator{
		transactionPool: map[string]msg.Transaction{
			"aaa": {From: "one", To: "two", Amount: 1},
			"bbb": {From: "one", To: "two", Amount: 2},
			"ccc": {From: "one", To: "two", Amount: 3},
			"ddd": {From: "one", To: "two", Amount: 4},
			"eee": {From: "one", To: "two", Amount: 5},
		},
		Node: Node{
			state: &storage.MapStorage{
				Alloc: map[string]uint64{
					"one": 200,
					"two": 50,
				},
			},
		},
	}

	txs := vd.popTransactions(3)
	if len(txs) != 3 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 3, len(txs))
	}
	if len(vd.transactionPool) != 2 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 2, len(vd.transactionPool))
	}

	txs = vd.popTransactions(3)
	if len(txs) != 2 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 2, len(txs))
	}
	if len(vd.transactionPool) != 0 {
		t.Fatalf("wrong transactions count: want=%v get=%v", 0, len(vd.transactionPool))
	}
}
