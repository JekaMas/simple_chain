package node

import (
	"errors"
	"simple_chain/msg"
)

type BlockPool map[uint64][]msg.Block

func (p BlockPool) AddBlock(b msg.Block) error {
	pool := p[b.BlockNum]
	pool = append(pool, b)
	p[b.BlockNum] = pool
	return nil
}

func (p BlockPool) HasBlockNum(blockNum uint64) bool {
	pool, ok := p[blockNum]
	return ok && len(pool) > 0
}

func (p BlockPool) PopBlock(blockNum uint64) (msg.Block, error) {
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
