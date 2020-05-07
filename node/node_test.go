package node

import (
	"reflect"
	"simple_chain/genesis"
	"simple_chain/msg"
	"testing"
)

func TestNode_InsertBlockSuccess(t *testing.T) {
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

func TestNode_ApplyTransactionSuccess(t *testing.T) {
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

func TestNode_VerifyBlockSuccess(t *testing.T) {
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

func TestNode_VerifyTransactionSuccess(t *testing.T) {
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

func TestNode_VerifySameBlockFailure(t *testing.T) {
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

func TestNode_RevertBlock(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one": 200,
		"two": 50,
	}

	nd, _ := NewNode(gen)
	vd, _ := NewValidator(gen)
	_ = vd.AddTransaction(msg.Transaction{
		From:   "one",
		To:     "two",
		Amount: 100,
		Fee:    10,
	})

	block, _ := vd.newBlock()
	stateBefore := nd.state.Copy()

	_ = nd.insertBlock(block)
	insertedBlockHash := nd.lastBlockHash()

	if reflect.DeepEqual(stateBefore, nd.state) {
		t.Fatalf("state was not changed: check insert block function")
	}

	err := nd.revertLastBlock()
	if err != nil {
		t.Fatalf("error while reverting last block: %v", err)
	}
	revertedBlockHash := nd.lastBlockHash()

	if insertedBlockHash == revertedBlockHash {
		t.Fatalf("same hashes for inserted and reverted blocks")
	}
	if nd.lastBlockNum != 0 {
		t.Fatalf("last block num was not changed")
	}
	if len(nd.blocks) != 1 {
		t.Fatalf("block len not correct: get=%v, want=%v", len(nd.blocks), 1)
	}
	if !reflect.DeepEqual(nd.state, stateBefore) {
		t.Fatalf("block was not reverted: \n get state: %v, \n before state: %v", nd.state, stateBefore)
	}
}
