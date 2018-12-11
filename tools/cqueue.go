package tools

import (
	"sync"
	`github.com/DSiSc/craft/types`
)

type CycleQueue struct {
	c         *sync.Cond
	cqueue    []*types.Transaction
	ppos      uint64 // index that put a item
	gpos      uint64 // index that get a item
	total     uint64 // total length of the queue
	maxPerGet uint64 // max num of item a get
	full      bool
}

func NewQueue(quesuSize uint64, maxItemPerGet uint64) *CycleQueue {
	return &CycleQueue{
		c:         sync.NewCond(&sync.Mutex{}),
		cqueue:    make([]*types.Transaction, quesuSize),
		total:     quesuSize,
		maxPerGet: maxItemPerGet,
		full:      false,
	}
}

func (cq *CycleQueue) Producer(value *types.Transaction) {
	cq.c.L.Lock()
	// roll back
	if cq.ppos+1 == cq.total {
		cq.cqueue[cq.ppos] = value
		cq.ppos = 0
		cq.full = true
	} else {
		cq.cqueue[cq.ppos] = value
		cq.ppos += 1
	}
	cq.c.L.Unlock()

}

func (cq *CycleQueue) Consumer() []*types.Transaction {
	var count uint64
	var txs = make([]*types.Transaction, 0, cq.maxPerGet)
	var overflow = false
	for {
		cq.c.L.Lock()
		if cq.gpos == cq.ppos {
			if !cq.full {
				cq.c.L.Unlock()
				return txs
			} else {
				cq.full = false
				if overflow {
					cq.c.L.Unlock()
					return txs
				}
			}
		}

		if cq.gpos > cq.ppos {
			overflow = true
		}

		if count >= cq.maxPerGet {
			cq.c.L.Unlock()
			return txs
		}
		tx := cq.cqueue[cq.gpos]
		cq.gpos += 1
		if cq.gpos == cq.total {
			cq.gpos = 0
		}
		txs = append(txs, tx)
		count = count + 1
		cq.c.L.Unlock()
	}
}

func (cq *CycleQueue) Count() uint64 {
	cq.c.L.Lock()
	defer cq.c.L.Unlock()
	if cq.gpos < cq.ppos {
		return (cq.ppos - cq.gpos)
	}
	if cq.gpos > cq.ppos {
		return (cq.total - cq.gpos + cq.ppos)
	}

	if cq.full {
		return cq.total
	}
	return (cq.gpos - cq.ppos)
}

func (cq *CycleQueue) GetGpos() uint64 {
	cq.c.L.Lock()
	defer cq.c.L.Unlock()
	return cq.gpos
}

func (cq *CycleQueue) GetPpos() uint64 {
	cq.c.L.Lock()
	defer cq.c.L.Unlock()
	return cq.ppos
}
