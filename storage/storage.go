package storage

import (
	"errors"
	"simple_chain/encode"
	"simple_chain/genesis"
)

type Storage interface {
	Put(key string, data uint64)
	Get(key string) (uint64, error)
	// Operations
	Sub(key string, amount uint64)
	Add(key string, amount uint64)
	Hash() (string, error)
}

/* --- Implementation ----------------------------------------------------------------------------------------------- */

type MapStorage struct {
	storage map[string]uint64
}

func NewMap() MapStorage {
	return MapStorage{
		make(map[string]uint64),
	}
}

func FromGenesis(genesis *genesis.Genesis) MapStorage {
	storage := NewMap()
	block := genesis.ToBlock()

	for _, tx := range block.Transactions {
		storage.Put(tx.To, tx.Amount)
	}

	return storage
}

func (m MapStorage) Put(key string, data uint64) {
	m.storage[key] = data
}

func (m MapStorage) Get(key string) (uint64, error) {
	data, ok := m.storage[key]
	if !ok {
		return 0, errors.New("not found")
	}
	return data, nil
}

/* --- Operations --------------------------------------------------------------------------------------------------- */

func (m MapStorage) Add(key string, amount uint64) {
	// todo error
	m.storage[key] = m.storage[key] + amount
}

func (m MapStorage) Sub(key string, amount uint64) {
	// todo error
	m.storage[key] = m.storage[key] - amount
}

func (m MapStorage) Hash() (string, error) {
	return encode.HashAlloc(m.storage)
}
