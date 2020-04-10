package bc

import (
	"crypto"
	"crypto/ed25519"
	"testing"
	"time"
)

func TestSendTransactionSuccess(t *testing.T) {
	numOfValidators := 3
	numOfPeers := 5

	initialBalance := uint64(100000)
	peers := make([]Blockchain, numOfPeers)

	genesis := Genesis{
		make(map[string]uint64),
		make([]crypto.PublicKey, 0, numOfValidators),
	}

	keys := make([]ed25519.PrivateKey, numOfPeers)
	for i := range keys {
		_, key, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		// initialize validators
		keys[i] = key
		if numOfValidators > 0 {
			genesis.Validators = append(genesis.Validators, key.Public())
			numOfValidators--
		}
		// initialize other nodes
		address, err := PubKeyToAddress(key.Public())
		if err != nil {
			t.Error(err)
		}
		genesis.Alloc[address] = initialBalance
	}

	var err error
	for i := 0; i < numOfPeers; i++ {
		peers[i], err = NewNode(keys[i], genesis)
		if err != nil {
			t.Error(err)
		}
	}
	// add all peers to each other
	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			err = peers[i].AddPeer(peers[j])
			if err != nil {
				t.Error(err)
			}
		}
	}
	// initialize test transaction
	tr := Transaction{
		From:   peers[3].NodeAddress(),
		To:     peers[4].NodeAddress(),
		Amount: 100,
		Fee:    10,
		PubKey: keys[3].Public().(ed25519.PublicKey),
	}

	tr, err = peers[3].SignTransaction(tr)
	if err != nil {
		t.Fatal(err)
	}

	err = peers[0].AddTransaction(tr)
	if err != nil {
		t.Fatal(err)
	}

	//wait transaction processing
	time.Sleep(time.Second * 5)

	//check "from" balance
	balance, err := peers[0].GetBalance(peers[3].NodeAddress())
	if err != nil {
		t.Fatal(err)
	}

	if balance != initialBalance-100-10 {
		t.Fatal("Incorrect from balance")
	}

	//check "to" balance
	balance, err = peers[0].GetBalance(peers[4].NodeAddress())
	if err != nil {
		t.Fatal(err)
	}

	if balance != initialBalance+100 {
		t.Fatal("Incorrect to balance")
	}

	//check validators balance
	for i := 0; i < 3; i++ {
		balance, err = peers[0].GetBalance(peers[i].NodeAddress())
		if err != nil {
			t.Error(err)
		}

		if balance > initialBalance {
			t.Error("Incorrect validator balance")
		}
	}
}