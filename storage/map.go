package storage

import (
	"errors"
	"fmt"
	"simple_chain/encode"
	"simple_chain/genesis"
	"sync"
)

type operation struct {
	name   string
	key    string
	amount uint64
}

type MapStorage struct {
	Alloc   map[string]uint64
	History []operation
	mxAlloc sync.Mutex
}

func NewMap() *MapStorage {
	return &MapStorage{
		Alloc: make(map[string]uint64),
	}
}

func FromGenesis(genesis genesis.Genesis) *MapStorage {
	storage := NewMap()
	block := genesis.ToBlock()

	storage.PutBlockToHistory(0)
	for _, tx := range block.Transactions {
		_ = storage.Put(tx.To, tx.Amount)
	}

	return storage
}

func (m *MapStorage) Get(key string) (uint64, error) {
	data, ok := m.Alloc[key]
	if !ok {
		return 0, errors.New("account not found")
	}
	return data, nil
}

func (m *MapStorage) Copy() Storage {
	m.Lock()
	defer m.Unlock()

	alloc := make(map[string]uint64)
	for key, value := range m.Alloc {
		alloc[key] = value
	}

	history := make([]operation, len(m.History))
	copy(history, m.History)

	return &MapStorage{
		Alloc:   alloc,
		History: history,
	}
}

func (m *MapStorage) PutOrAdd(key string, data uint64) error {
	_, ok := m.Alloc[key]
	if ok {
		return m.Add(key, data)
	} else {
		return m.Put(key, data)
	}
}

func (m *MapStorage) Put(key string, data uint64) error {
	_, ok := m.Alloc[key]
	if ok {
		return fmt.Errorf("account '%v' already exists", key)
	}

	m.Alloc[key] = data
	m.History = append(m.History, operation{"Put", key, data})
	return nil
}

func (m *MapStorage) Add(key string, amount uint64) error {
	fund, ok := m.Alloc[key]
	if !ok {
		return fmt.Errorf("no such account: %v", key)
	}

	m.Alloc[key] = fund + amount
	m.History = append(m.History, operation{"Add", key, amount})
	return nil
}

func (m *MapStorage) Sub(key string, amount uint64) error {
	fund, ok := m.Alloc[key]
	if !ok {
		return errors.New("no such account")
	}
	if fund < amount {
		return errors.New("insufficient funds")
	}

	m.Alloc[key] = fund - amount
	m.History = append(m.History, operation{"Sub", key, amount})
	return nil
}

func (m *MapStorage) PutBlockToHistory(num uint64) {
	m.History = append(m.History, operation{"Block", "", num})
}

func (m *MapStorage) RevertBlock() {
	for {
		op := m.revertOperation()
		if op.name == "Block" {
			return
		}
	}
}

// revertOperation - delete and return one operation from history
func (m *MapStorage) revertOperation() operation {
	// pop
	op := m.History[len(m.History)-1]
	m.History = m.History[:len(m.History)-1]
	// revertOperation
	switch op.name {
	case "Put":
		delete(m.Alloc, op.key)
	case "Add":
		m.Alloc[op.key] -= op.amount
	case "Sub":
		m.Alloc[op.key] += op.amount
	}
	return op
}

func (m *MapStorage) Hash() (string, error) {
	return encode.HashAlloc(m.Alloc)
}

func (m *MapStorage) String() string {
	m.Lock()
	defer m.Unlock()

	return fmt.Sprint(m.Alloc)
}

func (m *MapStorage) Lock() {
	m.mxAlloc.Lock()
}

func (m *MapStorage) Unlock() {
	m.mxAlloc.Unlock()
}
