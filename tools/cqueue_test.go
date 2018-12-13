package tools

import (
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

var MockCQ *CycleQueue

var BlockCapacity = 3
var TxPoolCapacity = 10

func TestNewQueue(t *testing.T) {
	MockCQ = NewQueue(10, 3)
	assert.NotNil(t, MockCQ)
	assert.Equal(t, TxPoolCapacity, cap(MockCQ.cqueue))
}

func MockNewTrans() []*types.Transaction {
	txs := make([]*types.Transaction, 0, 11)
	var tx *types.Transaction
	for i := 0; i < 11; i++ {
		tx = &types.Transaction{
			Data: types.TxData{
				AccountNonce: uint64(i),
			},
		}
		txs = append(txs, tx)
	}
	return txs
}

var txs = MockNewTrans()

func TestCycleQueue_Producer(t *testing.T) {
	assert.NotNil(t, txs[0])
	MockCQ.Producer(txs[0])
	assert.Equal(t, uint64(1), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	MockCQ.Producer(txs[1])
	assert.Equal(t, uint64(2), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	MockCQ.Producer(txs[2])
	MockCQ.Producer(txs[3])
	assert.Equal(t, uint64(4), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
}

func TestCycleQueue_Consumer(t *testing.T) {
	res := MockCQ.Consumer()
	assert.NotNil(t, txs)
	assert.Equal(t, 3, len(res))
	// MockCQ.Producer(txs[2])
	assert.Equal(t, uint64(4), MockCQ.ppos)
	assert.Equal(t, uint64(3), MockCQ.gpos)
	res = MockCQ.Consumer()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, uint64(4), MockCQ.ppos)
	assert.Equal(t, uint64(4), MockCQ.gpos)
	MockCQ.Producer(txs[4])
	MockCQ.Producer(txs[5])
	MockCQ.Producer(txs[6])
	MockCQ.Producer(txs[7])
	MockCQ.Producer(txs[8])
	MockCQ.Producer(txs[9])
	assert.Equal(t, uint64(0), MockCQ.ppos)
	assert.Equal(t, uint64(4), MockCQ.gpos)
	MockCQ.Producer(txs[10])
	assert.Equal(t, uint64(1), MockCQ.ppos)
	assert.Equal(t, uint64(4), MockCQ.gpos)
	res = MockCQ.Consumer()
	assert.Equal(t, 3, len(res))
	assert.Equal(t, uint64(7), MockCQ.gpos)
	res = MockCQ.Consumer()
	assert.Equal(t, uint64(0), MockCQ.gpos)
}

func TestCycleQueue_ProducerAdndConsumer(t *testing.T) {
	MockCQ = NewQueue(10, 5)
	MockCQ.Producer(txs[0])
	assert.Equal(t, uint64(1), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	MockCQ.Producer(txs[1])
	MockCQ.Producer(txs[2])
	MockCQ.Producer(txs[3])
	MockCQ.Producer(txs[4])
	MockCQ.Producer(txs[5])
	MockCQ.Producer(txs[6])
	MockCQ.Producer(txs[7])
	MockCQ.Producer(txs[8])
	assert.Equal(t, uint64(9), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	assert.Equal(t, uint64(9), MockCQ.Count())
	MockCQ.Producer(txs[9])
	assert.Equal(t, uint64(0), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	assert.Equal(t, uint64(10), MockCQ.Count())
	MockCQ.Consumer()
	assert.Equal(t, uint64(5), MockCQ.gpos)
	MockCQ.Consumer()
	assert.Equal(t, uint64(0), MockCQ.gpos)
	MockCQ.Producer(txs[0])
	assert.Equal(t, uint64(1), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	MockCQ.Producer(txs[1])
	MockCQ.Producer(txs[2])
	MockCQ.Producer(txs[3])
	MockCQ.Producer(txs[4])
	MockCQ.Producer(txs[5])
	MockCQ.Producer(txs[6])
	MockCQ.Producer(txs[7])
	MockCQ.Producer(txs[8])
	assert.Equal(t, uint64(9), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	MockCQ.Producer(txs[9])
	assert.Equal(t, uint64(0), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	MockCQ.Consumer()
	assert.Equal(t, uint64(5), MockCQ.gpos)
	MockCQ.Consumer()
	assert.Equal(t, uint64(0), MockCQ.gpos)
}

func TestCycleQueue_ProducerAdndConsumer1(t *testing.T) {
	MockCQ := NewQueue(4, 3)
	MockCQ.Producer(txs[0])
	MockCQ.Producer(txs[1])
	MockCQ.Producer(txs[2])
	assert.Equal(t, uint64(3), MockCQ.ppos)
	assert.Equal(t, uint64(0), MockCQ.gpos)
	assert.Equal(t, uint64(3), MockCQ.Count())
	MockCQ.Consumer()
	assert.Equal(t, uint64(3), MockCQ.ppos)
	assert.Equal(t, uint64(3), MockCQ.gpos)
	assert.Equal(t, uint64(0), MockCQ.Count())

	MockCQ.Producer(txs[3])
	MockCQ.Producer(txs[4])
	txs := MockCQ.Consumer()
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, MockCQ.ppos, MockCQ.gpos)
}

func TestCycleQueue_SetDiscarding(t *testing.T) {
	MockCQ := NewQueue(4, 2)
	MockCQ.Producer(txs[0])
	MockCQ.Producer(txs[1])
	MockCQ.Producer(txs[2])
	MockCQ.SetDiscarding(common.TxHash(txs[0]))
	txs := MockCQ.Consumer()
	assert.Equal(t, 2, len(txs))
}
