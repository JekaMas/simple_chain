package node

import (
	"crypto/ed25519"
	"simple_chain/genesis"
	"simple_chain/msg"
	"simple_chain/storage"
	"testing"
	"time"
)

func TestValidator(t *testing.T) {
	gen := genesis.New()
	// validator 1
	v1, err := NewValidatorFromGenesis(&gen)
	if err != nil {
		t.Fatal(err)
	}
	// validator 2
	v2, err := NewValidatorFromGenesis(&gen)
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
}

func TestValidator_WaitMissedValidator(t *testing.T) {

	gen := genesis.New()
	var vds []*Validator
	for i := 0; i < 3; i++ {
		pubKey, privKey, _ := ed25519.GenerateKey(nil)
		gen.Validators = append(gen.Validators, pubKey)
		vd, _ := NewValidator(privKey, &gen)
		vds = append(vds, vd)
	}

	for i, vd := range vds {
		vd.validators = gen.Validators
		t.Logf("validator1 %v len=%v", simplifyAddress(vd.NodeAddress()), len(vd.validators))

		for j, vdPeer := range vds {
			if i != j {
				_ = vd.AddPeer(vdPeer)
			}
		}
		// do nothing with second validator
		if i != 2 {
			go vd.startValidating()
		}
	}

	// wait validating
	time.Sleep(time.Millisecond * 10)

	// check states
	for i, vd := range vds {
		if len(vd.blocks) != 2 {
			t.Fatalf("wrong blocks number validator%v: want=%v, get=%v", i+1, 2, len(vd.blocks))
		}
	}
}

func TestValidator_Reward(t *testing.T) {
	/* TODO */
}

func TestValidator_Mining(t *testing.T) {
	/* TODO */
}

func TestValidator_VerifyBlockWithCoinbaseTransaction(t *testing.T) {
	vd := NewValidator()
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
