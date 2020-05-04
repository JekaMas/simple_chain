package pool

import (
	"errors"
	"fmt"
	"simple_chain/msg"
	"sync"
)

type BlockPool struct {
	alloc map[uint64][]msg.Block
	mx    *sync.Mutex
}

func NewBlockPool() BlockPool {
	return BlockPool{
		alloc: make(map[uint64][]msg.Block),
		mx:    &sync.Mutex{},
	}
}

func (p *BlockPool) Insert(b msg.Block) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	pool := p.alloc[b.BlockNum]
	blockHash, err := b.Hash()
	if err != nil {
		return fmt.Errorf("incorrect block: %v", err)
	}

	for _, pooled := range pool {
		hash, err := pooled.Hash()
		if err != nil {
			return fmt.Errorf("incorrect block in the pool: %v", err)
		}
		if hash == blockHash {
			return errors.New("block already in pool")
		}
	}

	pool = append(pool, b)
	p.alloc[b.BlockNum] = pool
	return nil
}

func (p *BlockPool) Pop(blockNum uint64) (msg.Block, error) {
	p.mx.Lock()
	defer p.mx.Unlock()

	pool, ok := p.alloc[blockNum]
	if !ok {
		return msg.Block{}, errors.New("no such block")
	}
	if len(pool) == 0 {
		return msg.Block{}, errors.New("no blocks")
	}

	block := pool[len(pool)-1] // fixme get last one
	p.alloc[blockNum] = pool[:len(pool)-1]
	return block, nil
}

func (p *BlockPool) HasBlockNum(blockNum uint64) bool {
	p.mx.Lock()
	defer p.mx.Unlock()

	pool, ok := p.alloc[blockNum]
	return ok && len(pool) > 0
}
