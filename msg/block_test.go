package msg

import (
	"reflect"
	"testing"
)

func TestBlock_HashSame(t *testing.T) {

	tr1 := Transaction{"one", "two", 20, 30, []byte("abc"), []byte("cde")}
	tr2 := Transaction{"thr", "two", 30, 40, []byte("bde"), []byte("fgh")}

	block1 := Block{1, 20, 111, []Transaction{tr1, tr2}, "abc", "abc", "fgh", []byte("abba"), []byte("baa")}
	block2 := Block{1, 20, 111, []Transaction{tr1, tr2}, "abc", "abc", "fgh", []byte("abba"), []byte("baa")}

	hash1, err := block1.Hash()
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}

	hash2, err := block2.Hash()
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}

	if !reflect.DeepEqual(hash1, hash2) {
		t.Fatalf("same block hash difference: \n%v vs %v", hash1, hash2)
	}
}
