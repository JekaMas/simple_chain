package msg

import "simple_chain/encode"

type Block struct {
	BlockNum      uint64
	Timestamp     int64
	Transactions  []Transaction
	BlockHash     string `json:"-"`
	PrevBlockHash string
	StateHash     string
	Signature     []byte `json:"-"`
}

func (bl Block) Hash() (string, error) {
	b, err := encode.Bytes(bl)
	if err != nil {
		return "", err
	}
	return encode.Hash(b), nil
}
