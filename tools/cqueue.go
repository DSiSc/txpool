package tools

import (
	"sync"
)

type CycleQueue struct {
	c         *sync.Cond
	cqueue    []interface{}
	ppos      uint64 // index that put a item
	gpos      uint64 // index that get a item
	total     uint64 // total length of the queue
	maxPerGet uint64 // max num of item a get
	full      bool
}

func NewQueue(quesuSize uint64, maxItemPerGet uint64) *CycleQueue {
	return &CycleQueue{
		c:         sync.NewCond(&sync.Mutex{}),
		cqueue:    make([]interface{}, quesuSize),
		total:     quesuSize,
		maxPerGet: maxItemPerGet,
		full:      false,
	}
}

/*
func pirntInfo(value interface{}, put bool, c *CycleQueue) {
	tx := value.(*types.Transaction)
	if put {
		log.Debug("put item[%d]: %d and hash is %x.",
			c.ppos, tx.Data.AccountNonce, common.TxHash(tx))
	} else {
		log.Debug("get item[%d]: %d and hash is %x.",
			c.gpos, tx.Data.AccountNonce, common.TxHash(tx))
	}
}
*/
func (cq *CycleQueue) Producer(value interface{}) {
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
	// pirntInfo(value, true, cq)
	cq.c.L.Unlock()

}

func (cq *CycleQueue) Consumer() []interface{} {
	var count uint64
	var txs = make([]interface{}, 0, cq.maxPerGet)
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
		// pirntInfo(tx, false, cq)
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
