package storage

import (
	"fmt"
	"sync"
)

type Storage interface {
	Put(key string, data uint64) error
	PutMap(map[string]uint64) error
	Get(key string) (uint64, error)

	// Operations
	Sub(key string, amount uint64) error
	Add(key string, amount uint64) error
	Hash() (string, error)

	// Revert operations
	Revert(trCount int) error

	// Copy
	Copy() Storage

	// Concurrency
	sync.Locker

	// String
	fmt.Stringer
}
