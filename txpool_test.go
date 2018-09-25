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
	errs := fmt.Errorf("the tx [207 172 68 74 133 202 86 65 81 91 116 2 175 105 105 194 110 169 221 233 129 234 137 126 59 56 204 62 206 84 211 13] has exist")
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

	// try to et exist tx
	err := pool.AddTx(tx)
	assert.Nil(err)
	exceptTx := pool.GetTxByHash(common.TxHash(tx))
	assert.Equal(common.TxHash(tx), common.TxHash(exceptTx))

	// try to get not exist tx
	tx.Data.AccountNonce = uint64(10)
	exceptTx = pool.GetTxByHash(common.TxHash(tx))
	assert.Nil(exceptTx)
}
