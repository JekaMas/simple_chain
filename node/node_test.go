// +build !race

package node

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"simple_chain/genesis"
	"simple_chain/msg"
	"sync"
	"testing"
	"time"
)

/* --- BAD ---------------------------------------------------------------------------------------------------------- */

// !!!
//func TestSyncBlockSuccess(t *testing.T) {
//	numOfValidators := 3
//	numOfPeers := 5
//
//	initialBalance := uint64(100000)
//	peers := make([]*Node, numOfPeers)
//	gen := genesis.New()
//
//	keys := make([]ed25519.PrivateKey, numOfPeers)
//	for i := range keys {
//		_, key, err := ed25519.GenerateKey(nil)
//		if err != nil {
//			t.Fatal(err)
//		}
//		// initialize validators
//		keys[i] = key
//		if numOfValidators > 0 {
//			gen.Validators = append(gen.Validators, key.Public())
//			numOfValidators--
//		}
//		// initialize other nodes
//		address, err := PubKeyToAddress(key.Public())
//		if err != nil {
//			t.Error(err)
//		}
//		gen.Alloc[address] = initialBalance
//	}
//
//	var err error
//	for i := 0; i < numOfPeers; i++ {
//		peers[i], err = NewNode(keys[i], &gen)
//		if err != nil {
//			t.Error(err)
//		}
//	}
//	// add all peers to each other
//	for i := 0; i < len(peers); i++ {
//		for j := i + 1; j < len(peers); j++ {
//			err = peers[i].AddPeer(peers[j])
//			if err != nil {
//				t.Error(err)
//			}
//		}
//	}
//	// initialize test transaction
//	tr := msg.Transaction{
//		From:   peers[3].NodeAddress(),
//		To:     peers[4].NodeAddress(),
//		Amount: 100,
//		Fee:    10,
//		PubKey: keys[3].Public().(ed25519.PublicKey),
//	}
//
//	tr, err = peers[3].SignTransaction(tr)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	err = applyTransaction(peers[0].state, peers[0].NodeAddress(), tr)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	//wait transaction processing
//	time.Sleep(time.Millisecond * 1)
//
//	//check "from" balance
//	balance, err := peers[0].GetBalance(peers[3].NodeAddress())
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if balance != initialBalance-100-10 {
//		t.Fatalf("Incorrect from balance: %v vs %v", balance, initialBalance-100-10)
//	}
//
//	//check "to" balance
//	balance, err = peers[0].GetBalance(peers[4].NodeAddress())
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if balance != initialBalance+100 {
//		t.Fatal("Incorrect to balance")
//	}
//
//	//check validators balance
//	for i := 0; i < 3; i++ {
//		balance, err = peers[0].GetBalance(peers[i].NodeAddress())
//		if err != nil {
//			t.Error(err)
//		}
//
//		if balance > initialBalance {
//			t.Error("Incorrect validator balance")
//		}
//	}
//}

func TestRace(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			fmt.Println(i) // Not the 'i' you are looking for.
			wg.Done()
		}()
	}
	wg.Wait()
	t.Logf("success")
}

// !!!
func TestApplyTransactionSuccess(t *testing.T) {
	// todo
}

// !!!
func TestNodeInsertBlockSuccess(t *testing.T) {
	gen := genesis.New()
	v, err := NewValidatorFromGenesis(0, &gen)
	if err != nil {
		t.Errorf("new validator: %v", err)
	}

	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Errorf("generate key: %v", err)
	}

	n, err := NewNode(key, &gen)
	if err != nil {
		t.Errorf("new node: %v", err)
	}

	// t.Log("validator state: ", v.state)
	// t.Log("node state:      ", n.state)

	block, err := v.newBlock()
	if err != nil {
		t.Errorf("new block error: %v", err)
	}

	// t.Log("new block: ", block)
	time.Sleep(1 * time.Millisecond)

	if err := n.verifyBlock(block); err != nil {
		t.Fatalf("verify block: %v", err)
	}
}

// !!! todo sigfault
//func TestNodeBlockProcessing(t *testing.T) {
//	// generate validator key
//	pubKey, _, err := ed25519.GenerateKey(nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	validatorAddr, err := PubKeyToAddress(pubKey)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	// generate one node with one validator
//	nd := &Node{validators: []crypto.PublicKey{pubKey}}
//	// initial state
//	nd.state = storage.NewMap()
//	nd.state.Put("one", 200)
//	nd.state.Put("two", 50)
//	nd.state.Put(validatorAddr, 50)
//
//	// insert one block with one transaction
//	err = nd.insertBlock(msg.Block{
//		BlockNum: 1,
//		Transactions: []msg.Transaction{
//			{
//				From:   "one",
//				To:     "two",
//				Fee:    10,
//				Amount: 100,
//			},
//		},
//	})
//
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	// get current state
//	one, _ := nd.state.Get("one")
//	two, _ := nd.state.Get("two")
//	val, _ := nd.state.Get(validatorAddr)
//
//	// check current state
//	if one != 90 {
//		t.Errorf("first account wrong state %v != %v", one, 90)
//	}
//	if two != 150 {
//		t.Errorf("seccond account wrong state %v != %v", two, 150)
//	}
//	if val != 60 {
//		t.Errorf("validator account wrong state %v != %v", val, 60)
//	}
//}

// !!!
func TestVerifyBlockSuccess(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	v, err := NewValidatorFromGenesis(0, &gen)
	if err != nil {
		t.Errorf("new validator: %v", err)
	}

	err = v.AddTransaction(msg.Transaction{
		From:      "two",
		To:        "one",
		Amount:    10,
		Fee:       20,
		PubKey:    nil,
		Signature: nil,
	})
	if err != nil {
		t.Errorf("new validator: %v", err)
	}

	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Errorf("generate key: %v", err)
	}

	n, err := NewNode(key, &gen)
	if err != nil {
		t.Errorf("new node: %v", err)
	}

	// t.Log("validator state: ", v.state)
	// t.Log("node state:      ", n.state)

	block, err := v.newBlock()
	if err != nil {
		t.Errorf("new block error: %v", err)
	}

	// t.Log("new block: ", block)
	time.Sleep(1 * time.Millisecond)

	if err := n.verifyBlock(block); err != nil {
		t.Fatalf("verify block: %v", err)
	}
}

// !!!
func TestVerifyTransactionSuccess(t *testing.T) {

	gen := genesis.New()
	pubKey, privKey, _ := ed25519.GenerateKey(nil)
	addr, _ := PubKeyToAddress(pubKey)
	gen.Alloc[addr] = 100

	nd1, _ := NewNode(privKey, &gen)
	_, privKey, _ = ed25519.GenerateKey(nil)
	nd2, _ := NewNode(privKey, &gen)

	tr, err := nd1.newTransaction(nd2.NodeAddress(), 10)
	if err != nil {
		t.Errorf("new transaction error: %v", err)
	}

	state1 := nd1.state.Copy()
	err = verifyTransaction(state1, tr)
	if err != nil {
		t.Errorf("verify transaction: %v", err)
	}

	state2 := nd2.state.Copy()
	err = verifyTransaction(state2, tr)
	if err != nil {
		t.Errorf("verify transaction: %v", err)
	}
}

/* --- GOOD --------------------------------------------------------------------------------------------------------- */

func TestNodesSyncTwoNodes(t *testing.T) {
	// init nodes
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one": 200,
		"two": 50,
	}

	pubKey, privateKey, _ := ed25519.GenerateKey(nil)
	valAddr, _ := PubKeyToAddress(pubKey)
	gen.Alloc[valAddr] = 70

	validator, _ := NewValidator(privateKey, 0, &gen)
	validator.validators = []crypto.PublicKey{validator.NodeKey()}

	NewTestNode := func() *Node {
		// generate one node with one validator
		_, privateKey, _ := ed25519.GenerateKey(nil)
		nd, _ := NewNode(privateKey, &gen)
		nd.validators = []crypto.PublicKey{validator.NodeKey()}
		return nd
	}

	nd1 := NewTestNode()
	nd2 := NewTestNode()

	t.Logf("genesis block: [%v]", simplifyAddress(nd1.blocks[0].BlockHash))

	tr := msg.Transaction{
		From:   "one",
		To:     "two",
		Fee:    10,
		Amount: 100,
		PubKey: nd1.NodeKey().(ed25519.PublicKey),
	}

	tr, err := nd1.SignTransaction(tr)
	if err != nil {
		t.Fatalf("sign transaction: %v", err)
	}

	err = validator.AddTransaction(tr)
	if err != nil {
		t.Fatalf("add transaction: %v", err)
	}

	_, err = validator.newBlock()
	if err != nil {
		t.Fatalf("new block: %v", err)
	}

	// add one block
	//err = nd1.insertBlock(block)
	//if err != nil {
	//	t.Fatalf("insert block: %v", err)
	//}

	t.Logf("insert block [%v] to %v",
		simplifyAddress(validator.blocks[1].BlockHash), simplifyAddress(nd1.NodeAddress()))

	t.Logf("node1 %v", simplifyAddress(nd1.NodeAddress()))
	// t.Logf("node2 %v", simplifyAddress(nd2.NodeAddress()))
	t.Logf("valid %v", simplifyAddress(validator.NodeAddress()))

	// connect peers
	peers := []*Node{
		&validator.Node,
		nd1,
		nd2,
	}
	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			if err := peers[i].AddPeer(peers[j]); err != nil {
				t.Error(err)
			}
		}
	}

	// wait processing
	time.Sleep(5 * time.Millisecond)

	t.Logf("node1 %v state: %v", simplifyAddress(nd1.NodeAddress()), nd1.state.String())
	t.Logf("node2 [%v] state: %v", simplifyAddress(nd2.NodeAddress()), nd2.state.String())
	t.Logf("valid %v state: %v", simplifyAddress(validator.NodeAddress()), validator.state.String())

	// check states
	for i := 0; i < len(peers); i++ {
		// get balances
		one, err := peers[i].GetBalance("one")
		if err != nil {
			t.Error(err)
		}
		two, err := peers[i].GetBalance("two")
		if err != nil {
			t.Error(err)
		}
		valBalance, err := peers[i].GetBalance(validator.NodeAddress())
		if err != nil {
			t.Error(err)
		}

		// check balances
		if valBalance != 70+10 {
			t.Fatalf("wrong second validator's state: get=%v, want=%v ",
				valBalance, 70+10)
		}
		if one != 200-(100+10) {
			t.Fatalf("wrong first node's state: get=%v, want=%v ",
				one, 200-(100+10))
		}
		if two != 50+100 {
			t.Fatalf("wrong second node's state: get=%v, want=%v ",
				two, 50+100)
		}
	}
}

func TestVerifySameBlockFailure(t *testing.T) {
	gen := genesis.New()
	v, _ := NewValidatorFromGenesis(0, &gen)
	block, _ := v.newBlock()

	_, privKey, _ := ed25519.GenerateKey(nil)
	nd2, _ := NewNode(privKey, &gen)

	err := nd2.insertBlock(block)
	if err != nil {
		t.Errorf("verify block: %v", err)
	}

	err = nd2.verifyBlock(block)
	if err == nil {
		t.Error("same block inserted twice")
	} else {
		t.Log(err)
	}
}
