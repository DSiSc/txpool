package tools

import (
	"errors"
	"fmt"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"sync"
)

var BlockCapacity = 3
var TxPoolCapacity = 10

type CycleQueue struct {
	c      *sync.Cond
	cqueue []interface{}
	ppos   int // 存放元素的位置
	gpos   int // 获取元素的位置
	total  int // 总的元素个数
}

func NewQueue() *CycleQueue {
	return &CycleQueue{
		c:      sync.NewCond(&sync.Mutex{}),
		cqueue: make([]interface{}, TxPoolCapacity),
		total:  TxPoolCapacity,
	}
}

func pirntS(value interface{}, put bool, c *CycleQueue) {
	switch value.(type) {
	case *types.Transaction:
		if put {
			fmt.Printf("put item[%d]: %d and hash is %x.\n",
				c.ppos, value.(*types.Transaction).Data.AccountNonce, common.TxHash((value.(*types.Transaction))))
		} else {
			fmt.Printf("get item[%d]: %d and hash is %x.\n",
				c.gpos, (value.(*types.Transaction).Data.AccountNonce), common.TxHash((value.(*types.Transaction))))
		}
	default:
		panic(errors.New("Unsupport type"))
	}
}

// 生产者
func (cq *CycleQueue) Producer(value interface{}) {
	cq.c.L.Lock()

	cq.ppos += 1
	// roll back
	if cq.ppos == cq.total {
		cq.ppos = 0
	}
	cq.cqueue[cq.ppos] = value
	pirntS(value, true, cq)
	cq.c.L.Unlock()

	//c.Signal()
}

// 消费者
func (cq *CycleQueue) Consumer() []interface{} {
	var count = 0
	var txs = make([]interface{}, 0, BlockCapacity)
	for {
		cq.c.L.Lock()
		for cq.gpos == cq.ppos {
			cq.c.L.Unlock()
			return txs
		}

		if count >= BlockCapacity {
			cq.c.L.Unlock()
			return txs
		}

		cq.gpos += 1
		if cq.gpos == cq.total { //roll back
			cq.gpos = 0
		}

		tx := cq.cqueue[cq.gpos]
		txs = append(txs, tx)
		pirntS(tx, false, cq)
		count = count + 1
		cq.c.L.Unlock()
	}
}
