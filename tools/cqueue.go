package tools

import (
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"github.com/DSiSc/craft/log"
	"sync"
)

type CycleQueue struct {
	c         *sync.Cond
	cqueue    []*types.Transaction
	ppos      uint64 // index that put a item
	gpos      uint64 // index that get a item
	total     uint64 // total length of the queue
	maxPerGet uint64 // max num of item a get
	full      bool
	data      map[types.Hash]bool
}

func NewQueue(quesuSize uint64, maxItemPerGet uint64) *CycleQueue {
	return &CycleQueue{
		c:         sync.NewCond(&sync.Mutex{}),
		cqueue:    make([]*types.Transaction, quesuSize),
		total:     quesuSize,
		maxPerGet: maxItemPerGet,
		full:      false,
		data:      make(map[types.Hash]bool),
	}
}

func (cq *CycleQueue) Producer(tx *types.Transaction) {
	cq.c.L.Lock()
	// roll back
	cq.cqueue[cq.ppos] = tx
	cq.data[common.TxHash(tx)] = true
	if cq.ppos+1 == cq.total {
		cq.ppos = 0
		cq.full = true
	} else {
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
		txHash := common.TxHash(tx)
		cq.gpos += 1
		if cq.gpos == cq.total {
			cq.gpos = 0
		}
		if !cq.data[txHash] {
			log.Debug("tx %x has been committed to block.", txHash)
		} else {
			txs = append(txs, tx)
			count = count + 1
		}
		delete(cq.data, txHash)
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

func (cq *CycleQueue) SetDiscarding(tx types.Hash) {
	cq.c.L.Lock()
	cq.data[tx] = false
	cq.c.L.Unlock()
}
