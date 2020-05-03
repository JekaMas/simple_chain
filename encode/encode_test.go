package encode

import (
	"reflect"
	"testing"
)

func TestHashAllocSame(t *testing.T) {
	// todo can't test it - can't declare map in some order
	alloc1 := map[string]uint64{
		"one":   10,
		"two":   20,
		"three": 21,
	}

	alloc2 := map[string]uint64{
		"three": 21,
		"two":   20,
		"one":   10,
	}

	hash1, err := HashAlloc(alloc1)
	if err != nil {
		t.Fatalf("hash alloc error: %v", err)
	}

	hash2, err := HashAlloc(alloc2)
	if err != nil {
		t.Fatalf("hash alloc error: %v", err)
	}

	if !reflect.DeepEqual(hash1, hash2) {
		t.Fatalf("same alloc diffeent hashes: \n%v \nvs \n%v", hash1, hash2)
	}
}
