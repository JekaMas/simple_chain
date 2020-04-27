package node

import (
	"reflect"
	"simple_chain/genesis"
	"testing"
	"time"
)

func TestValidator(t *testing.T) {
	gen := genesis.New()
	// validator 1
	v1, err := NewValidator(&gen)
	if err != nil {
		t.Fatal(err)
	}
	// validator 2
	v2, err := NewValidator(&gen)
	if err != nil {
		t.Fatal(err)
	}

	err = v1.AddPeer(v2)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 10)

	go v1.startValidating()
	go v2.startValidating()

	time.Sleep(time.Millisecond * 100)

	if len(v1.blocks) <= 1 || len(v2.blocks) <= 1 {
		t.Fatalf("no blocks validated")
	}

	if !reflect.DeepEqual(v1.blocks, v2.blocks) {
		t.Fatalf("validators not synced: \n%v vs \n%v",
			v1.blocks, v2.blocks)
	}
}

func TestValidator_Reward(t *testing.T) {
	/* TODO */
}

func TestValidator_Mining(t *testing.T) {
	/* TODO */
}

func TestValidator_VerifyBlockWithCoinbaseTransaction(t *testing.T) {
	// vd := NewValidator()
}

/* --- GOOD --------------------------------------------------------------------------------------------------------- */

func TestValidator_leadingZeros(t *testing.T) {

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
