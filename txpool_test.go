package txpool

import (
	"github.com/DSiSc/craft/types"
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
		tx := types.NewTransaction(uint64(i), to[i], amount, uint64(i), amount, nil)
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
	switch instance := txpool.(type) {
	case *TxPool:
		//vte.config.GlobalSlots
		assert.Equal(slot, instance.config.GlobalSlots, "they should be equal")
	}
}

// Test add a tx to txpool
func Test_AddTx(t *testing.T) {
	assert := assert.New(t)

	txList := mock_transactions(2)
	assert.NotNil(txList)

	txpool := NewTxPool(DefaultTxPoolConfig)
	assert.NotNil(txpool)

	// add the specified tx to txpool
	txpool.AddTx(txList[0])
	switch instance := txpool.(type) {
	case *TxPool:
		//vte.config.GlobalSlots
		assert.Equal(1, instance.all.Count(), "they should be equal")
	}

	// add deplicate tx to txpool
	txpool.AddTx(txList[0])
	switch instance := txpool.(type) {
	case *TxPool:
		//vte.config.GlobalSlots
		assert.Equal(1, instance.all.Count(), "they should be equal")
	}

}

// Test Get a tx from txpool
func Test_GetTxs(t *testing.T) {
	assert := assert.New(t)
	tx := mock_transactions(1)[0]
	assert.NotNil(tx)

	txpool := NewTxPool(DefaultTxPoolConfig)
	assert.NotNil(txpool)

	txpool.AddTx(tx)
	txList := txpool.GetTxs()
	assert.Equal(1, len(txList), "they should be equal")
	assert.Equal(txList[0].Hash(), tx.Hash(), "they should be equal")
}

// Test DelTxs txs from txpool
func Test_DelTxs(t *testing.T) {
	assert := assert.New(t)

	txpool := NewTxPool(DefaultTxPoolConfig)
	assert.NotNil(txpool)

	err := txpool.DelTxs()
	assert.Nil(err)
}
