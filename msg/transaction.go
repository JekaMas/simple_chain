package msg

import (
	"bytes"
	"crypto/ed25519"
	"encoding/gob"
	"simple_chain/encode"
)

type Transaction struct {
	From      string
	To        string
	Amount    uint64
	Fee       uint64
	PubKey    ed25519.PublicKey
	Signature []byte `json:"-"`
}

func (t Transaction) Hash() (string, error) {
	b, err := t.Bytes()
	if err != nil {
		return "", err
	}
	return encode.Hash(b), nil
}

func (t Transaction) Bytes() ([]byte, error) {
	b := bytes.NewBuffer(nil)
	err := gob.NewEncoder(b).Encode(t)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
