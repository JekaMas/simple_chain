package node

import (
	"context"
	"crypto/ed25519"
	"reflect"
	"simple_chain/genesis"
	"simple_chain/log"
	"simple_chain/msg"
	"testing"
	"time"
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

/* --- Network ------------------------------------------------------------------------------------------------------ */

func TestNode_SyncBlockStopBroadcasting(t *testing.T) {
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

func TestNode_SyncTwoNodes(t *testing.T) {
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

	time.Sleep(100 * time.Millisecond)

	if len(nd2.blocks) != 2 {
		t.Fatalf("no block was synced")
	}

	if !reflect.DeepEqual(nd1.state, nd2.state) {
		t.Fatalf("wrong synced state")
	}
}

func TestNode_SyncTwoNodesWithDifferentTotalDifficulty(t *testing.T) {
	gen := genesis.New()
	nd1, _ := NewNode(gen)
	nd2, _ := NewNode(gen)

	t.Logf("genesis block [%v]", log.Simplify(gen.ToBlock().BlockHash))

	// chain 1
	vd1, _ := NewValidator(gen)
	for i := 0; i < 3; i++ {
		block, _ := vd1.newBlock()
		_ = vd1.insertBlock(block)
		_ = nd1.insertBlock(block)
		t.Logf("nd1 insert block [%v]", log.Simplify(block.BlockHash))
	}

	t.Logf("-->")

	// chain 2
	vd2, _ := NewValidator(gen)
	for i := 0; i < 5; i++ {
		block, _ := vd2.newBlock()
		_ = vd2.insertBlock(block)
		_ = nd2.insertBlock(block)
		t.Logf("nd2 insert block [%v]", log.Simplify(block.BlockHash))
	}

	_ = nd1.AddPeer(nd2)
	time.Sleep(time.Millisecond * 100)

	if nd1.totalDifficulty() != 6 {
		t.Errorf("wrong total difficulty: get=%v, want=%v", nd1.totalDifficulty(), 6)
	}
	if !reflect.DeepEqual(nd1.blocks, nd2.blocks) {
		t.Fatalf("nodes not synchronized")
	}
	if !reflect.DeepEqual(nd1.state, nd2.state) {
		t.Fatalf("states are not equal")
	}
}

func TestNode_SyncOneNodeOneValidator(t *testing.T) {
	gen := genesis.New()

	nd, _ := NewNode(gen)
	vd, _ := NewValidator(gen)

	_ = nd.AddPeer(vd)

	vd.startValidating()
	time.Sleep(time.Millisecond * 100)

	_ = vd.stopValidating()
	time.Sleep(time.Millisecond * 300)

	if !reflect.DeepEqual(nd.state, vd.state) {
		t.Fatalf("%v and %v state difference: \n%v vs \n%v",
			log.Simplify(nd.NodeAddress()), log.Simplify(vd.NodeAddress()), nd.state, vd.state)
	}
}

func TestNode_SyncTwoNodesOneValidator(t *testing.T) {
	peers, validators, _ := makeSomePeers(2, 1, uint64(100000))

	// fully connected
	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			if err := peers[i].AddPeer(peers[j]); err != nil {
				t.Error(err)
			}
		}
	}

	for _, val := range validators {
		val.startValidating()
	}

	time.Sleep(time.Millisecond * 100)

	for _, val := range validators {
		_ = val.stopValidating()
	}

	time.Sleep(time.Millisecond * 300)

	for i, peer1 := range peers {
		for j, peer2 := range peers {
			if i != j && !reflect.DeepEqual(peer1.state, peer2.state) {
				t.Fatalf("%v and %v state difference: \n%v vs \n%v",
					log.Simplify(peer1.NodeAddress()), log.Simplify(peer2.NodeAddress()), peer1.state, peer2.state)
			}
		}
	}
}

func TestNode_SyncTwoNodesTwoValidators(t *testing.T) {
	peers, validators, _ := makeSomePeers(2, 2, uint64(100000))

	// fully connected
	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			if err := peers[i].AddPeer(peers[j]); err != nil {
				t.Error(err)
			}
		}
	}

	for _, val := range validators {
		val.startValidating()
	}

	time.Sleep(time.Millisecond * 100)

	for _, val := range validators {
		_ = val.stopValidating()
	}

	time.Sleep(time.Millisecond * 300)

	for i, peer1 := range peers {
		for j, peer2 := range peers {
			if i != j && !reflect.DeepEqual(peer1.state, peer2.state) {
				t.Logf("%v and %v state difference: \n%v vs \n%v",
					log.Simplify(peer1.NodeAddress()), log.Simplify(peer2.NodeAddress()), peer1.state, peer2.state)
				if len(peer1.blocks) != len(peer2.blocks) {
					t.Fatalf("state and blocks len difference")
				}
			}
		}
	}
}

func TestNode_SyncFullyConnected(t *testing.T) {
	peers, validators, _ := makeSomePeers(5, 3, uint64(100000))

	// fully connected
	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			if err := peers[i].AddPeer(peers[j]); err != nil {
				t.Error(err)
			}
		}
	}

	for _, val := range validators {
		val.startValidating()
	}

	time.Sleep(time.Millisecond * 100)

	for _, val := range validators {
		_ = val.stopValidating()
	}

	time.Sleep(time.Millisecond * 300)

	for i, peer1 := range peers {
		for j, peer2 := range peers {
			if i != j && !reflect.DeepEqual(peer1.state, peer2.state) {
				t.Logf("%v and %v state difference: \n%v vs \n%v",
					log.Simplify(peer1.NodeAddress()), log.Simplify(peer2.NodeAddress()), peer1.state, peer2.state)
				if len(peer1.blocks) != len(peer2.blocks) {
					t.Fatalf("state and blocks len difference")
				}
			}
		}
	}
}

func TestNode_SyncLinear(t *testing.T) {
	peers, validators, _ := makeSomePeers(5, 3, uint64(100000))

	// linear
	for i := 1; i < len(peers); i++ {
		if err := peers[i-1].AddPeer(peers[i]); err != nil {
			t.Error(err)
		}
	}

	for _, val := range validators {
		val.startValidating()
	}

	time.Sleep(time.Millisecond * 100)

	for _, val := range validators {
		_ = val.stopValidating()
	}

	time.Sleep(time.Millisecond * 300)

	for i, peer1 := range peers {
		for j, peer2 := range peers {
			if i != j && !reflect.DeepEqual(peer1.state, peer2.state) {
				t.Logf("%v and %v state difference: \n%v vs \n%v",
					log.Simplify(peer1.NodeAddress()), log.Simplify(peer2.NodeAddress()), peer1.state, peer2.state)
				if len(peer1.blocks) != len(peer2.blocks) {
					t.Fatalf("state and blocks len difference")
				}
			}
		}
	}
}

func TestNode_SyncRing(t *testing.T) {
	peers, validators, _ := makeSomePeers(5, 3, uint64(100000))

	// ring
	for i := 1; i < len(peers); i++ {
		if err := peers[i-1].AddPeer(peers[i]); err != nil {
			t.Error(err)
		}
	}
	// close ring
	if err := peers[0].AddPeer(peers[len(peers)-1]); err != nil {
		t.Error(err)
	}

	for _, val := range validators {
		val.startValidating()
	}

	time.Sleep(time.Millisecond * 100)

	for _, val := range validators {
		_ = val.stopValidating()
	}

	time.Sleep(time.Millisecond * 300)

	for i, peer1 := range peers {
		for j, peer2 := range peers {
			if i != j && !reflect.DeepEqual(peer1.state, peer2.state) {
				t.Logf("%v and %v state difference: \n%v vs \n%v",
					log.Simplify(peer1.NodeAddress()), log.Simplify(peer2.NodeAddress()), peer1.state, peer2.state)
				if len(peer1.blocks) != len(peer2.blocks) {
					t.Fatalf("state and blocks len difference")
				}
			}
		}
	}
}

func TestNode_SyncStar(t *testing.T) {
	peers, validators, _ := makeSomePeers(5, 3, uint64(100000))

	// star
	centerIndex := 2
	for i := 0; i < len(peers); i++ {
		if i != centerIndex {
			if err := peers[centerIndex].AddPeer(peers[i]); err != nil {
				t.Error(err)
			}
		}
	}

	for _, val := range validators {
		val.startValidating()
	}

	time.Sleep(time.Millisecond * 100)

	for _, val := range validators {
		_ = val.stopValidating()
	}

	time.Sleep(time.Millisecond * 300)

	for i, peer1 := range peers {
		for j, peer2 := range peers {
			if i != j && !reflect.DeepEqual(peer1.state, peer2.state) {
				t.Logf("%v and %v state difference: \n%v vs \n%v",
					log.Simplify(peer1.NodeAddress()), log.Simplify(peer2.NodeAddress()), peer1.state, peer2.state)
				if len(peer1.blocks) != len(peer2.blocks) {
					t.Fatalf("state and blocks len difference")
				}
			}
		}
	}
}

/* --- Utils -------------------------------------------------------------------------------------------------------- */

// makeSomePeers - return (peers, validators, nodes), where peers = validators + nodes
func makeSomePeers(nNodes uint64, nVals uint64, initBalance uint64) ([]*Node, []*Validator, []*Node) {
	peers := make([]*Node, nNodes+nVals)
	nodes := make([]*Node, nNodes)
	vals := make([]*Validator, nVals)

	keys := make([]ed25519.PrivateKey, nNodes+nVals)
	gen := genesis.New()

	for i := range keys {
		_, keys[i], _ = ed25519.GenerateKey(nil)
		address, _ := PubKeyToAddress(keys[i].Public())
		gen.Alloc[address] = initBalance
	}

	for i, key := range keys {
		if nVals > 0 {
			val, _ := NewValidatorWithKey(gen, key)
			peers[i] = &val.Node
			vals[nVals-1] = val
			nVals--
		} else if nNodes > 0 {
			node, _ := NewNodeWithKey(gen, key)
			peers[i] = node
			nodes[nNodes-1] = node
			nNodes--
		}
	}

	return peers, vals, nodes
}
