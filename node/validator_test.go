package node

import (
	"context"
	"simple_chain/genesis"
	"simple_chain/msg"
	"testing"
	"time"
)

func TestValidator_Mining(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one": 20,
		"two": 30,
	}

	vd, _ := NewValidator(gen)
	nd, _ := NewNode(gen)
	_ = vd.AddTransaction(msg.Transaction{
		From:   "two",
		To:     "one",
		Amount: 10,
		Fee:    20,
	})

	err := vd.AddPeer(nd)
	if err != nil {
		t.Fatalf("can't add peer: %v", err)
	}

	func() {
		go vd.startValidating()
		select {
		case <-time.After(time.Second * 1):
			return
		}
	}()

	vd.mxBlocks.Lock()
	if len(vd.blocks) < 2 {
		t.Fatalf("no blocks was validated")
	}
	if leadingZeros(vd.blocks[vd.lastBlockNum].BlockHash) != BlockDifficulty {
		t.Fatalf("wrong block hash difficulty")
	}
}

func TestValidator_RewardAfterBlockReturns(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	vd, _ := NewValidator(gen)
	nd, _ := NewNode(gen)
	_ = vd.AddTransaction(msg.Transaction{
		From:   "two",
		To:     "one",
		Amount: 10,
		Fee:    20,
	})

	block, err := vd.newBlock()
	if err != nil {
		t.Errorf("new block error: %v", err)
	}

	v, _ := vd.GetBalance(vd.NodeAddress())
	if v != 0 {
		t.Errorf("early reward: get=%v, want=%v", v, 0)
	}

	err = vd.AddPeer(nd)
	if err != nil {
		t.Fatalf("add peer error: %v", err)
	}
	vd.Broadcast(context.Background(), msg.Message{
		From: vd.NodeAddress(),
		Data: msg.BlockMessage{
			Block:           block,
			TotalDifficulty: vd.totalDifficulty() + 1,
		},
	})

	time.Sleep(time.Millisecond * 1)

	if len(vd.blocks) != 2 {
		t.Fatalf("block was not received: len=%v, want=%v", len(vd.blocks), 2)
	}
	v, _ = vd.GetBalance(vd.NodeAddress())
	if v != BlockReward+20 {
		t.Errorf("wrong balance: get=%v, want=%v", v, BlockReward+20)
	}
}

func TestValidator_CoinbaseTransactionCorrect(t *testing.T) {
	gen := genesis.New()
	gen.Alloc = map[string]uint64{
		"one":   20,
		"two":   30,
		"three": 40,
	}

	vd, _ := NewValidator(gen)
	_ = vd.AddTransaction(msg.Transaction{
		From:   "two",
		To:     "one",
		Amount: 10,
		Fee:    20,
	})

	block, err := vd.newBlock()
	if err != nil {
		t.Errorf("new block error: %v", err)
	}

	if len(block.Transactions) != 2 {
		t.Fatalf("wrong transactions count: get=%v want=%v", len(block.Transactions), 2)
	}

	coinbase := block.Transactions[0]
	if coinbase.From != "" {
		t.Fatalf("coinbase can't have sender: From='%v'", coinbase.From)
	}
	if coinbase.To != vd.NodeAddress() {
		t.Fatalf("coinbase wrong receiver address: To='%v'", coinbase.To)
	}
	if coinbase.Amount != BlockReward {
		t.Fatalf("coinbase wrong amount: Amount='%v' want=%v", coinbase.Amount, BlockReward)
	}
	if coinbase.Fee != 0 {
		t.Fatalf("coinbase wrong fee amount: Feee='%v' want=%v", coinbase.Fee, 0)
	}
}

func TestValidator_LeadingZeros(t *testing.T) {

	type test struct {
		hash      string
		wantCount uint64
	}

	tests := []test{
		{"000aaa", 3},
		{"aaa", 0},
		{"00a0aa", 2},
		{"00000a0a0000a0000", 5},
	}

	for _, tt := range tests {
		t.Run(tt.hash, func(t *testing.T) {
			if leadingZeros(tt.hash) != tt.wantCount {
				t.Fatalf("wrong count: get=%v, want=%v", leadingZeros(tt.hash), tt.wantCount)
			}
		})
	}
}
