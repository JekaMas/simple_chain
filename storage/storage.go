package storage

import (
	"errors"
	"simple_chain/encode"
	"simple_chain/genesis"
)

type Storage interface {
	Put(key string, data uint64) error
	PutOrAdd(key string, amount uint64) error
	Get(key string) (uint64, error)
	// Operations
	Sub(key string, amount uint64) error
	Add(key string, amount uint64) error
	Hash() (string, error)
	// Copy
	Copy() Storage
}

/* --- Implementation ----------------------------------------------------------------------------------------------- */

type MapStorage struct {
	alloc map[string]uint64
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

func (m MapStorage) Put(key string, data uint64) error {
	_, ok := m.alloc[key]
	if ok {
		return errors.New("account already exists")
	}

	m.alloc[key] = data
	return nil
}

func (m MapStorage) Get(key string) (uint64, error) {
	data, ok := m.alloc[key]
	if !ok {
		return 0, errors.New("not found")
	}
	return data, nil
}

func (m MapStorage) Copy() Storage {
	alloc := make(map[string]uint64)
	for key, value := range m.alloc {
		alloc[key] = value
	}
	return MapStorage{alloc: alloc}
}

/* --- Operations --------------------------------------------------------------------------------------------------- */

func (m MapStorage) PutOrAdd(key string, amount uint64) error {
	_, ok := m.alloc[key]
	if ok {
		return m.Add(key, amount)
	} else {
		return m.Put(key, amount)
	}
}

func (m MapStorage) Add(key string, amount uint64) error {
	fund, ok := m.alloc[key]
	if !ok {
		return errors.New("no such account")
	}
	m.alloc[key] = fund + amount
	return nil
}

func (m MapStorage) Sub(key string, amount uint64) error {
	fund, ok := m.alloc[key]
	if !ok {
		return errors.New("no such account")
	}
	if fund < amount {
		return errors.New("insufficient funds")
	}
	m.alloc[key] = fund - amount
	return nil
}

func (m MapStorage) Hash() (string, error) {
	return encode.HashAlloc(m.alloc)
}
