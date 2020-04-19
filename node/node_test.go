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

func TestSendTransactionSuccess(t *testing.T) {
	numOfValidators := 3
	numOfPeers := 5

	initialBalance := uint64(100000)
	peers := make([]Blockchain, numOfPeers)
	genesis := genesis.New()

	keys := make([]ed25519.PrivateKey, numOfPeers)
	for i := range keys {
		_, key, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		// initialize validators
		keys[i] = key
		if numOfValidators > 0 {
			genesis.Validators = append(genesis.Validators, key.Public())
			numOfValidators--
		}
		// initialize other nodes
		address, err := PubKeyToAddress(key.Public())
		if err != nil {
			t.Error(err)
		}
		genesis.Alloc[address] = initialBalance
	}

	var err error
	for i := 0; i < numOfPeers; i++ {
		peers[i], err = NewNode(keys[i], &genesis)
		if err != nil {
			t.Error(err)
		}
	}
	// add all peers to each other
	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			err = peers[i].AddPeer(peers[j])
			if err != nil {
				t.Error(err)
			}
		}
	}
	// initialize test transaction
	tr := msg.Transaction{
		From:   peers[3].NodeAddress(),
		To:     peers[4].NodeAddress(),
		Amount: 100,
		Fee:    10,
		PubKey: keys[3].Public().(ed25519.PublicKey),
	}

	tr, err = peers[3].SignTransaction(tr)
	if err != nil {
		t.Fatal(err)
	}

	err = peers[0].AddTransaction(tr)
	if err != nil {
		t.Fatal(err)
	}

	//wait transaction processing
	time.Sleep(time.Second * 5)

	//check "from" balance
	balance, err := peers[0].GetBalance(peers[3].NodeAddress())
	if err != nil {
		t.Fatal(err)
	}

	if balance != initialBalance-100-10 {
		t.Fatal("Incorrect from balance")
	}

	//check "to" balance
	balance, err = peers[0].GetBalance(peers[4].NodeAddress())
	if err != nil {
		t.Fatal(err)
	}

	if balance != initialBalance+100 {
		t.Fatal("Incorrect to balance")
	}

	//check validators balance
	for i := 0; i < 3; i++ {
		balance, err = peers[0].GetBalance(peers[i].NodeAddress())
		if err != nil {
			t.Error(err)
		}

		if balance > initialBalance {
			t.Error("Incorrect validator balance")
		}
	}
}

func TestNodeBlockProcessing(t *testing.T) {
	// generate validator key
	pubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	validatorAddr, err := PubKeyToAddress(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	// generate one node with one validator
	nd := &Node{validators: []crypto.PublicKey{pubKey}}
	// initial state
	nd.state = storage.NewMap()
	nd.state.Put("one", 200)
	nd.state.Put("two", 50)
	nd.state.Put(validatorAddr, 50)

	// insert one block with one transaction
	err = nd.insertBlock(msg.Block{
		BlockNum: 1,
		Transactions: []msg.Transaction{
			{
				From:   "one",
				To:     "two",
				Fee:    10,
				Amount: 100,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	// get current state
	one, _ := nd.state.Get("one")
	two, _ := nd.state.Get("two")
	val, _ := nd.state.Get(validatorAddr)

	// check current state
	if one != 90 {
		t.Errorf("first account wrong state %v != %v", one, 90)
	}
	if two != 150 {
		t.Errorf("seccond account wrong state %v != %v", two, 150)
	}
	if val != 60 {
		t.Errorf("validator account wrong state %v != %v", val, 60)
	}
}

func TestNodesSyncTwoNodes(t *testing.T) {

	// generate validator key
	pubKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	validatorAddr, err := PubKeyToAddress(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	genesis := genesis.Genesis{
		Alloc:      make(map[string]uint64),
		Validators: []crypto.PublicKey{pubKey},
	}

	NewTestNode := func() Node {
		// generate one node with one validator
		_, privateKey, _ := ed25519.GenerateKey(nil)
		nd, _ := NewNode(privateKey, &genesis)
		// generate node key
		// initial state
		nd.state = storage.NewMap()
		nd.state.Put("one", 200)
		nd.state.Put("two", 50)
		nd.state.Put(validatorAddr, 50)
		return *nd
	}

	// generate
	nd1 := NewTestNode()
	nd2 := NewTestNode()

	// add one block
	err = nd1.insertBlock(msg.Block{
		BlockNum: 1,
		Transactions: []msg.Transaction{
			{
				From:   "one",
				To:     "two",
				Fee:    10,
				Amount: 100,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	// synchronize second node
	err = nd1.AddPeer(&nd2)
	if err != nil {
		t.Errorf("add peer error: %v", err)
	}

	// wait processing
	time.Sleep(1 * time.Millisecond)
}

func TestVerifyBlockSuccess(t *testing.T) {

	gen := genesis.New()
	v, err := NewValidator(&gen)
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
	err = verifyTransaction(&state1, tr)
	if err != nil {
		t.Errorf("verify transaction: %v", err)
	}

	state2 := nd2.state.Copy()
	err = verifyTransaction(&state2, tr)
	if err != nil {
		t.Errorf("verify transaction: %v", err)
	}
}
