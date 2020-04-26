package node

import (
	"crypto"
	"crypto/ed25519"
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

func TestWaitMissedValidator(t *testing.T) {

	pubKey1, privKey1, _ := ed25519.GenerateKey(nil)
	pubKey2, privKey2, _ := ed25519.GenerateKey(nil)
	pubKey3, privKey3, _ := ed25519.GenerateKey(nil)

	gen := genesis.New()
	gen.Validators = []crypto.PublicKey{pubKey1, pubKey2, pubKey3}

	vd1, _ := NewValidator(privKey1, &gen)
	vd2, _ := NewValidator(privKey2, &gen)
	vd3, _ := NewValidator(privKey3, &gen)

	vd1.validators = gen.Validators
	vd2.validators = gen.Validators
	vd3.validators = gen.Validators

	t.Logf("validator1 %v len=%v", simplifyAddress(vd1.NodeAddress()), len(vd1.validators))
	t.Logf("validator2 %v len=%v", simplifyAddress(vd2.NodeAddress()), len(vd2.validators))
	t.Logf("validator3 %v len=%v", simplifyAddress(vd3.NodeAddress()), len(vd3.validators))

	// do nothing with second validator
	go vd1.startValidating()
	go vd3.startValidating()

	// wait validating
	time.Sleep(time.Millisecond * 10)

	// check states
	for i, vd := range []*Validator{vd1, vd2, vd3} {
		if len(vd.blocks) != 2 {
			t.Fatalf("wrong blocks number validator%v: want=%v, get=%v", i, 2, len(vd1.blocks))
		}
	}
}
