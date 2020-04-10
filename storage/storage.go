package storage

import "errors"

type Storage interface {
	Put(key string, data uint64)
	Get(key string) (uint64, error)
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

