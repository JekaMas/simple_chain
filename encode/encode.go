package encode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

func Hash(b []byte) string {
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func Bytes(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func HashAlloc(alloc map[string]uint64) (string, error) {
	// sort Alloc keys (lexicographical order)
	keys := make([]string, 0, len(alloc))
	for k := range alloc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	concatStr := ""

	for _, key := range keys {
		concatStr += fmt.Sprintf("%v:%v,", key, alloc[key])
	}

	return Hash([]byte(concatStr)), nil
}
