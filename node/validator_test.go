package node

import (
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

	go v1.startValidating()
	go v2.startValidating()

	time.Sleep(time.Second * 5)
}
