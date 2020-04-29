package pool

import (
	"errors"
	"fmt"
	"simple_chain/msg"
)

type BlockPool map[uint64][]msg.Block

func NewBlockPool() BlockPool {
	return make(map[uint64][]msg.Block)
}

func (p BlockPool) Insert(b msg.Block) error {
	pool := p[b.BlockNum]
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
	p[b.BlockNum] = pool
	return nil
}

func (p BlockPool) Pop(blockNum uint64) (msg.Block, error) {
	pool, ok := p[blockNum]
	if !ok {
		return msg.Block{}, errors.New("no such block")
	}
	if len(pool) == 0 {
		return msg.Block{}, errors.New("no blocks")
	}

	block := pool[len(pool)-1] // fixme get last one
	p[blockNum] = pool[:len(pool)-1]
	return block, nil
}

func (p BlockPool) HasBlockNum(blockNum uint64) bool {
	pool, ok := p[blockNum]
	return ok && len(pool) > 0
}
