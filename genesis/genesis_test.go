package genesis

import (
	"reflect"
	"testing"
	"time"
)

func TestGenesis_ToBlock(t *testing.T) {

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

func TestGenesis_BlockHashSame(t *testing.T) {
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

	hash1, _ := gen1.ToBlock().Hash()
	hash2, _ := gen2.ToBlock().Hash()

	if !reflect.DeepEqual(hash1, hash2) {
		t.Fatalf("genesis block hash difference: \n%v vs %v", hash1, hash2)
	}
}
