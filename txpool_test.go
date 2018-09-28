package txpool

import (
	"fmt"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

// mock a config for txpool
func mock_txpool_config(slot uint64) TxPoolConfig {
	mock_config := TxPoolConfig{
		GlobalSlots: slot,
	}
	return mock_config
}

// mock a transaction
func mock_transactions(num int) []*types.Transaction {
	to := make([]types.Address, num)
	for m := 0; m < num; m++ {
		for j := 0; j < types.AddressLength; j++ {
			to[m][j] = byte(m)
		}
	}
	amount := new(big.Int)
	txList := make([]*types.Transaction, 0)
	for i := 0; i < num; i++ {
		tx := common.NewTransaction(uint64(i), to[i], amount, uint64(i), amount, nil, to[i])
		txList = append(txList, tx)
	}
	return txList
}

// Test new a txpool
func Test_NewTxPool(t *testing.T) {
	var slot uint64 = 64
	assert := assert.New(t)

	mock_config := mock_txpool_config(slot)
	txpool := NewTxPool(mock_config)
	assert.NotNil(txpool)
	instance := txpool.(*TxPool)
	assert.Equal(slot, instance.config.GlobalSlots, "they should be equal")
	assert.NotNil(instance.all)
	assert.NotNil(instance.process)
	assert.NotNil(instance.txsQueue)

	mock_config = mock_txpool_config(uint64(0))
	txpool = NewTxPool(mock_config)
	instance = txpool.(*TxPool)
	assert.Equal(uint64(4096), instance.config.GlobalSlots, "they should be equal")

}

// Test add a tx to txpool
func Test_AddTx(t *testing.T) {
	assert := assert.New(t)

	txList := mock_transactions(3)
	assert.NotNil(txList)

	var MockTxPoolConfig = TxPoolConfig{
		GlobalSlots:    2,
		MaxTrxPerBlock: 2,
	}

	txpool := NewTxPool(MockTxPoolConfig)
	assert.NotNil(txpool)
	instance := txpool.(*TxPool)
	assert.Equal(uint64(2), instance.config.GlobalSlots)
	assert.Equal(uint64(2), instance.config.MaxTrxPerBlock)

	err := txpool.AddTx(txList[0])
	assert.Nil(err)
	instance = txpool.(*TxPool)

	// add duplicate tx to txpool
	err = txpool.AddTx(txList[0])
	assert.NotNil(err)
	errs := fmt.Errorf("the tx [15 72 229 1 174 103 134 209 244 212 139 199 131 35 99 73 0 242 47 255 200 150 184 19 9 134 78 65 28 61 137 244] has exist")
	assert.Equal(errs, err)
	instance = txpool.(*TxPool)
	assert.Equal(1, instance.all.Count(), "they should be equal")

	err = txpool.AddTx(txList[1])
	assert.Nil(err)
	assert.Equal(int(2), instance.all.Count(), "they should be equal")
	err = txpool.AddTx(txList[2])
	assert.NotNil(err)
	instance = txpool.(*TxPool)
	excErr := fmt.Errorf("txpool has full")
	assert.Equal(excErr, err)

}

// Test Get a tx from txpool
func Test_GetTxs(t *testing.T) {
	assert := assert.New(t)
	tx := mock_transactions(1)[0]
	assert.NotNil(tx)

	txpool := NewTxPool(DefaultTxPoolConfig)
	assert.NotNil(txpool)

	pool := txpool.(*TxPool)
	assert.Equal(0, len(pool.process))

	txpool.AddTx(tx)
	assert.Equal(0, len(pool.process))
	assert.Equal(1, pool.all.Count())

	pool.GetTxs()
	assert.Equal(1, len(pool.process))
	assert.Equal(0, pool.all.Count())
}

// Test DelTxs txs from txpool
func Test_DelTxs(t *testing.T) {
	assert := assert.New(t)
	tx := mock_transactions(1)[0]
	assert.NotNil(tx)

	txpool := NewTxPool(DefaultTxPoolConfig)
	assert.NotNil(txpool)
	pool := txpool.(*TxPool)
	assert.Equal(0, len(pool.process))

	pool.AddTx(tx)
	assert.Equal(0, len(pool.process))
	assert.Equal(1, pool.all.Count())

	pool.GetTxs()
	assert.Equal(1, len(pool.process))
	assert.Equal(0, pool.all.Count())

	pool.DelTxs()
	assert.Equal(0, len(pool.process))
	assert.Equal(0, pool.all.Count())
}

func TestTxPool_GetTxByHash(t *testing.T) {
	assert := assert.New(t)
	tx := mock_transactions(1)[0]
	assert.NotNil(tx)
	txpool := NewTxPool(DefaultTxPoolConfig)
	assert.NotNil(txpool)
	pool := txpool.(*TxPool)

	// try to get exist tx
	err := pool.AddTx(tx)
	assert.Nil(err)
	exceptTx := pool.GetTxByHash(common.TxHash(tx))
	assert.Equal(common.TxHash(tx), common.TxHash(exceptTx))

	// try to get not exist tx
	mockHash := types.Hash{
		0xbd, 0x79, 0x1d, 0x4a, 0xf9, 0x64, 0x8f, 0xc3, 0x7f, 0x94, 0xeb, 0x36, 0x53, 0x19, 0xf6, 0xd0,
		0xa9, 0x78, 0x9f, 0x9c, 0x22, 0x47, 0x2c, 0xa7, 0xa6, 0x12, 0xa9, 0xca, 0x4, 0x13, 0xc1, 0x4,
	}
	assert.NotEqual(exceptTx, mockHash)
	exceptTx = pool.GetTxByHash(mockHash)
	assert.Nil(exceptTx)
}
