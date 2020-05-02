package node

import (
	"context"
	"reflect"
	"simple_chain/genesis"
	"simple_chain/msg"
	"testing"
	"time"
)

/* --- BAD ---------------------------------------------------------------------------------------------------------- */

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

/* --- GOOD --------------------------------------------------------------------------------------------------------- */

func TestNodeInsertBlockSuccess(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	vd, _ := NewValidator(gen)
	nd, _ := NewNode(gen)
	_ = vd.AddTransaction(msg.Transaction{
		From:   "one",
		To:     "two",
		Amount: 10,
		Fee:    1,
	})

	block, err := vd.newBlock()
	if err != nil {
		t.Errorf("new block error: %v", err)
	}

	if err := nd.insertBlock(block); err != nil {
		t.Fatalf("insert block: %v", err)
	}

	if v, _ := nd.state.Get("one"); v != 9 {
		t.Fatalf("wrong 'one' state: get=%v want=%v'", v, 9)
	}

	if v, _ := nd.state.Get("two"); v != 40 {
		t.Fatalf("wrong 'two' state: get=%v want=%v'", v, 40)
	}

	if v, _ := nd.state.Get(vd.NodeAddress()); v != BlockReward+1 {
		t.Fatalf("wrong 'val' state: get=%v want=%v'", v, BlockReward+1)
	}
}

func TestApplyTransactionSuccess(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	nd, _ := NewNode(gen)
	vd, _ := NewNode(gen)

	tr := msg.Transaction{
		From:   "one",
		To:     "two",
		Amount: 10,
		Fee:    1,
	}

	if err := applyTransaction(nd.state, vd.NodeAddress(), tr); err != nil {
		t.Errorf("apply transaction error: %v", err)
	}

	if v, _ := nd.state.Get("one"); v != 9 {
		t.Fatalf("wrong 'one' state: get=%v want=%v'", v, 9)
	}

	if v, _ := nd.state.Get("two"); v != 40 {
		t.Fatalf("wrong 'two' state: get=%v want=%v'", v, 40)
	}

	if v, _ := nd.state.Get(vd.NodeAddress()); v != 1 {
		t.Fatalf("wrong 'val' state: get=%v want=%v'", v, 1)
	}
}

func TestVerifyBlockSuccess(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	vd, _ := NewValidator(gen)
	nd, _ := NewNode(gen)

	err := vd.AddTransaction(msg.Transaction{
		From:   "two",
		To:     "one",
		Amount: 10,
		Fee:    20,
	})
	if err != nil {
		t.Errorf("add transaction error: %v", err)
	}

	block, err := vd.newBlock()
	if err != nil {
		t.Errorf("new block error: %v", err)
	}

	if err := nd.verifyBlock(block); err != nil {
		t.Fatalf("verify block: %v", err)
	}
}

func TestVerifyTransactionSuccess(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	nd1, _ := NewNode(gen)
	nd2, _ := NewNode(gen)

	tr := msg.Transaction{
		From:   "one",
		To:     "two",
		Amount: 19,
		Fee:    1,
	}
	tr, _ = nd1.SignTransaction(tr)

	if err := verifyTransaction(nd1.state.Copy(), tr); err != nil {
		t.Errorf("verify transaction error: %v", err)
	}

	if err := verifyTransaction(nd2.state.Copy(), tr); err != nil {
		t.Errorf("verify transaction error: %v", err)
	}
}

func TestVerifySameBlockFailure(t *testing.T) {
	gen := genesis.New()
	vd, _ := NewValidator(gen)
	nd, _ := NewNode(gen)

	block, _ := vd.newBlock()

	err := nd.insertBlock(block)
	if err != nil {
		t.Errorf("verify block: %v", err)
	}

	err = nd.verifyBlock(block)
	if err == nil {
		t.Error("same block verified")
	} else {
		t.Log(err)
	}
}

func TestNodesSyncBlockStopBroadcasting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	gen := genesis.New()
	vd, _ := NewValidator(gen)
	nd, _ := NewNode(gen)

	in := make(chan msg.Message, MessagesBusLen)
	out := make(chan msg.Message, MessagesBusLen)

	block, _ := vd.newBlock()

	peer := connectedPeer{
		Address: "abc",
		In:      in,
		Out:     out,
		cancel:  cancel,
	}

	go nd.peerLoop(ctx, peer)

	// broadcast after receiving block
	in <- msg.Message{From: "abc", Data: block}
	<-out

	// send same block
	in <- msg.Message{From: "abc", Data: block}
	select {
	case b := <-out:
		if reflect.DeepEqual(b, block) {
			t.Fatalf("endless broadcasting")
		}
		t.Fatalf("received phantom message: %v", b)
	case <-time.After(time.Millisecond):
		// no messages - test passed
		return
	}
}

func TestNode_TotalDifficultyValue(t *testing.T) {
	gen := genesis.New()
	nd1, _ := NewNode(gen)
	nd2, _ := NewNode(gen)

	// chain 1
	vd1, _ := NewValidator(gen)
	for i := 0; i < 3; i++ {
		block, _ := vd1.newBlock()
		_ = nd1.insertBlock(block)
	}

	if nd1.totalDifficulty() != 4 {
		t.Fatalf("wrong total difficulty: get=%v, want=%v", nd1.totalDifficulty(), 4)
	}

	// chain 2
	vd2, _ := NewValidator(gen)
	for i := 0; i < 5; i++ {
		block, _ := vd2.newBlock()
		_ = nd2.insertBlock(block)
	}

	if nd2.totalDifficulty() != 6 {
		t.Fatalf("wrong total difficulty: get=%v, want=%v", nd2.totalDifficulty(), 6)
	}
}

func TestNode_IsTransactionSuccess(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	nd, _ := NewNode(gen)
	vd, _ := NewValidator(gen)

	tr := msg.Transaction{
		From:   "two",
		To:     "one",
		Amount: 10,
		Fee:    1,
	}
	_ = vd.AddTransaction(tr)

	block, err := vd.newBlock()
	if err != nil {
		t.Fatalf("new block error: %v", err)
	}
	_ = nd.insertBlock(block)

	if nd.IsTransactionSuccess(tr) {
		t.Fatalf("early transaction success")
	}

	// six more blocks
	for i := 0; i < 6; i++ {
		block, _ := vd.newBlock()
		_ = nd.insertBlock(block)
	}

	if !nd.IsTransactionSuccess(tr) {
		t.Fatalf("transaction must be success")
	}
}

func TestNodesSyncTwoNodes(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one": 200,
		"two": 50,
	}

	nd1, _ := NewNode(gen)
	nd2, _ := NewNode(gen)

	vd, _ := NewValidator(gen)
	err := vd.AddTransaction(msg.Transaction{
		From:   "one",
		To:     "two",
		Amount: 100,
		Fee:    10,
	})
	if err != nil {
		t.Errorf("add transaction error: %v", err)
	}

	block, _ := vd.newBlock()

	if err := nd1.insertBlock(block); err != nil {
		t.Fatalf("insert block err: %v", err)
	}

	if err := nd1.AddPeer(nd2); err != nil {
		t.Fatalf("add peer err: %v", err)
	}

	time.Sleep(1 * time.Millisecond)

	if len(nd2.blocks) != 2 {
		t.Fatalf("no block was synced")
	}

	if !reflect.DeepEqual(nd1.state, nd2.state) {
		t.Fatalf("wrong synced state")
	}
}

func TestNode_SyncDifferentTotalDifficulty(t *testing.T) {
	gen := genesis.New()
	nd1, _ := NewNode(gen)
	nd2, _ := NewNode(gen)

	// chain 1
	vd1, _ := NewValidator(gen)
	for i := 0; i < 3; i++ {
		block, _ := vd1.newBlock()
		_ = nd1.insertBlock(block)
	}
	// chain 2
	vd2, _ := NewValidator(gen)
	for i := 0; i < 5; i++ {
		block, _ := vd2.newBlock()
		_ = nd2.insertBlock(block)
	}

	_ = nd1.AddPeer(nd2)
	time.Sleep(time.Millisecond * 100)

	if nd1.totalDifficulty() != 6 {
		t.Errorf("wrong total difficulty: get=%v, want=%v", nd1.totalDifficulty(), 6)
	}
	if !reflect.DeepEqual(nd1.blocks, nd2.blocks) {
		t.Fatalf("nodes not synchronized")
	}
}
