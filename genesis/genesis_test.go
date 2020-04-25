package genesis

import (
	"reflect"
	"testing"
	"time"
)

func TestGenesis_BlockSameWithTimeDelay(t *testing.T) {

	gen := New()
	block1 := gen.ToBlock()

	time.Sleep(time.Millisecond * 10)

	block2 := gen.ToBlock()

	if !reflect.DeepEqual(block1, block2) {
		t.Fatalf("genesis blocks not equal")
	}

	t.Logf("hash1: %v", block1.BlockHash)
	t.Logf("hash2: %v", block2.BlockHash)
}

func TestGenesis_BlockSameWithDifferentTransactionsOrder(t *testing.T) {
	gen1 := New()
	gen1.Alloc = map[string]uint64{
		"one": 200,
		"two": 50,
	}

	gen2 := New()
	gen2.Alloc = map[string]uint64{
		"two": 50,
		"one": 200,
	}

	block1 := gen1.ToBlock()
	block2 := gen2.ToBlock()

	if !reflect.DeepEqual(block1, block2) {
		t.Fatalf("genesis blocks difference: \n%v vs %v", block1, block2)
	}
}
