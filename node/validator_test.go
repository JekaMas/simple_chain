package node

import (
	"simple_chain/genesis"
	"simple_chain/msg"
	"simple_chain/storage"
	"testing"
	"time"
)

func TestValidator(t *testing.T) {
	gen := genesis.New()
	// validator 1
	v1, err := NewValidatorFromGenesis(&gen)
	if err != nil {
		t.Fatal(err)
	}
	// validator 2
	v2, err := NewValidatorFromGenesis(&gen)
	if err != nil {
		t.Fatal(err)
	}

	err = v1.AddPeer(v2)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 10)

	go v1.startValidating()
	go v2.startValidating()

	time.Sleep(time.Millisecond * 100)
}

func TestValidator_popTransactions(t *testing.T) {
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
