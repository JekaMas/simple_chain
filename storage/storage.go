package storage

import (
	"errors"
	"fmt"
	"simple_chain/encode"
	"simple_chain/genesis"
	"sync"
)

type Storage interface {
	Put(key string, data uint64) error
	PutMap(map[string]uint64) error
	PutOrAdd(key string, amount uint64) error
	Get(key string) (uint64, error)
	// Operations
	Sub(key string, amount uint64) error
	Add(key string, amount uint64) error
	Hash() (string, error)
	// Copy
	Copy() Storage
	// Concurrency
	sync.Locker
	// String
	fmt.Stringer
}

/* --- Implementation ----------------------------------------------------------------------------------------------- */

type MapStorage struct {
	alloc   map[string]uint64
	mxAlloc sync.Mutex
}

func NewMap() *MapStorage {
	return &MapStorage{
		alloc: make(map[string]uint64),
	}
}

func FromGenesis(genesis *genesis.Genesis) *MapStorage {
	storage := NewMap()
	block := genesis.ToBlock()

	for _, tx := range block.Transactions {
		_ = storage.Put(tx.To, tx.Amount)
	}

	return storage
}

func (m *MapStorage) Put(key string, data uint64) error {
	_, ok := m.alloc[key]
	if ok {
		return errors.New("account already exists")
	}

	m.alloc[key] = data
	return nil
}

func (m *MapStorage) Get(key string) (uint64, error) {
	data, ok := m.alloc[key]
	if !ok {
		return 0, errors.New("not found")
	}
	return data, nil
}

func (m *MapStorage) Copy() Storage {
	m.Lock()
	defer m.Unlock()

	alloc := make(map[string]uint64)
	for key, value := range m.alloc {
		alloc[key] = value
	}
	return &MapStorage{
		alloc: alloc,
	}
}

/* --- Operations --------------------------------------------------------------------------------------------------- */

func (m *MapStorage) PutOrAdd(key string, amount uint64) error {
	_, ok := m.alloc[key]
	if ok {
		return m.Add(key, amount)
	} else {
		return m.Put(key, amount)
	}
}

func (m *MapStorage) PutMap(alloc map[string]uint64) error {
	for addr, fund := range alloc {
		if err := m.Put(addr, fund); err != nil {
			return err
		}
	}
	return nil
}

func (m *MapStorage) Add(key string, amount uint64) error {
	fund, ok := m.alloc[key]
	if !ok {
		return errors.New("no such account")
	}
	m.alloc[key] = fund + amount
	return nil
}

func (m *MapStorage) Sub(key string, amount uint64) error {
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

func (m *MapStorage) Hash() (string, error) {
	return encode.HashAlloc(m.alloc)
}

func (m *MapStorage) String() string {
	m.Lock()
	defer m.Unlock()

	return fmt.Sprint(m.alloc)
}

/* --- Concurrency -------------------------------------------------------------------------------------------------- */

func (m *MapStorage) Lock() {
	m.mxAlloc.Lock()
}

func (m *MapStorage) Unlock() {
	m.mxAlloc.Unlock()
}
