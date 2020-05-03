package pool

import (
	"reflect"
	"simple_chain/msg"
	"sync"
	"testing"
)

func TestBlockPool_InsertDifferent(t *testing.T) {

	block1 := msg.Block{BlockNum: 1, Nonce: 1}
	block2 := msg.Block{BlockNum: 1, Nonce: 2}

	bp := NewBlockPool()

	err := bp.Insert(block1)
	if err != nil {
		t.Fatalf("error insert1: %v", err)
	}

	err = bp.Insert(block2)
	if err != nil {
		t.Fatalf("error insert2: %v", err)
	}

	if len(bp.alloc[1]) != 2 {
		t.Fatalf("blocks was not inserted: get=%v, want=%v", len(bp.alloc[1]), 2)
	}
}

func TestBlockPool_InsertSame(t *testing.T) {

	block1 := msg.Block{BlockNum: 1}
	block2 := msg.Block{BlockNum: 1}

	bp := NewBlockPool()

	_ = bp.Insert(block1)
	err := bp.Insert(block2)

	if err == nil {
		t.Errorf("same block insert no error")
	}

	if len(bp.alloc[1]) != 1 {
		t.Fatalf("same block was inserted: get=%v, want=%v", len(bp.alloc[1]), 1)
	}
}

func TestBlockPool_Pop(t *testing.T) {

	block1 := msg.Block{BlockNum: 1, Nonce: 1}
	block2 := msg.Block{BlockNum: 1, Nonce: 2}

	bp := NewBlockPool()

	_ = bp.Insert(block1)
	_ = bp.Insert(block2)

	// 1
	b, err := bp.Pop(1)
	if err != nil {
		t.Fatalf("error while pop: %v", err)
	}
	if !reflect.DeepEqual(b, block2) {
		t.Fatalf("wrong block order")
	}
	if len(bp.alloc[1]) != 1 {
		t.Fatalf("block was not removed")
	}

	// 2
	b, err = bp.Pop(1)
	if err != nil {
		t.Fatalf("error while pop: %v", err)
	}
	if !reflect.DeepEqual(b, block1) {
		t.Fatalf("wrong block order")
	}
	if len(bp.alloc[1]) != 0 {
		t.Fatalf("block was not removed")
	}
}

// run with -race
func TestBlockPool_Concurrency(t *testing.T) {

	bp := NewBlockPool()
	wg := sync.WaitGroup{}

	wg.Add(3)

	go func() {
		defer wg.Done()
		_ = bp.Insert(msg.Block{BlockNum: 1})
	}()

	go func() {
		defer wg.Done()
		block, err := bp.Pop(1)
		t.Logf("err=%v, block=%v", err, block)
	}()

	go func() {
		defer wg.Done()
		t.Logf("has block 1: %v", bp.HasBlockNum(1))
	}()

	wg.Wait()
}
