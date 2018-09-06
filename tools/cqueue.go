package tools

import (
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"sync"
)

type CycleQueue struct {
	c         *sync.Cond
	cqueue    []interface{}
	ppos      uint64 // index that put a item
	gpos      uint64 // index that get a item
	total     uint64 // total length of the queue
	maxPerGet uint64 // max num of item a get
}

func NewQueue(quesuSize uint64, maxItemPerGet uint64) *CycleQueue {
	return &CycleQueue{
		c:         sync.NewCond(&sync.Mutex{}),
		cqueue:    make([]interface{}, quesuSize),
		total:     quesuSize,
		maxPerGet: maxItemPerGet,
	}
}

func pirntInfo(value interface{}, put bool, c *CycleQueue) {
	tx := value.(*types.Transaction)
	if put {
		log.Info("put item[%d]: %d and hash is %x.\n",
			c.ppos, tx.Data.AccountNonce, common.TxHash(tx))
	} else {
		log.Info("get item[%d]: %d and hash is %x.\n",
			c.gpos, tx.Data.AccountNonce, common.TxHash(tx))
	}
}

func (cq *CycleQueue) Producer(value interface{}) {
	cq.c.L.Lock()

	cq.ppos += 1
	// roll back
	if cq.ppos == cq.total {
		cq.ppos = 0
	}
	cq.cqueue[cq.ppos] = value
	pirntInfo(value, true, cq)
	cq.c.L.Unlock()

}

func (cq *CycleQueue) Consumer() []interface{} {
	var count uint64
	var txs = make([]interface{}, 0, cq.maxPerGet)
	for {
		cq.c.L.Lock()
		for cq.gpos == cq.ppos {
			cq.c.L.Unlock()
			return txs
		}

		if count >= cq.maxPerGet {
			cq.c.L.Unlock()
			return txs
		}

		cq.gpos += 1
		if cq.gpos == cq.total {
			cq.gpos = 0
		}

		tx := cq.cqueue[cq.gpos]
		txs = append(txs, tx)
		pirntInfo(tx, false, cq)
		count = count + 1
		cq.c.L.Unlock()
	}
}
