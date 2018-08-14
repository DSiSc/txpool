package core

import (
	"math/big"
	"testing"
	"github.com/DSiSc/txpool/common"
	"github.com/DSiSc/txpool/core/types"
)

// mock a config for txpool
func mock_txpool_config(slot uint64) TxPoolConfig {
	mock_config := TxPoolConfig{
		GlobalSlots: slot,
	}
	return mock_config
}

// mock a transaction
func mock_transactions(num int) []*types.Transaction{
	to := make([]common.Address, num)
	for m := 0; m < num; m ++{
		for j := 0; j < common.AddressLength; j++ {
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
func Test_NewTxPool(t *testing.T){
	var slot uint64 = 64
	mock_config := mock_txpool_config(slot)
	txpool := NewTxPool(mock_config)
	if nil != txpool{
		t.Log("PASS: success to create a txpool.")
	}else{
		t.Error("UNPASS: failed to create a txpool")
	}

	if txpool.config.GlobalSlots == slot {
		t.Log("PASS: getting global slots suceess.")
	}else{
		t.Error("NOPASS: getting global slots failed")
	}
}

// Test add a tx to txpool
func Test_AddTx(t *testing.T){
	txList := mock_transactions(2)
	if nil == txList{
		t.Error("UNPASS: mokc a transaction failed.")
	}
	txpool := NewTxPool(DefaultTxPoolConfig)
	if nil == txpool{
		t.Error("UNPASS: failed to create a txpool")
	}
    // add the specified tx to txpool
	txpool.AddTx(txList[0])
	if 1 != txpool.all.Count(){
		t.Error("UNPASS: failed to add a tx")
	}else{
		t.Log("PASS: success to add a tx to txpool.")
	}
	// add deplicate tx to txpool
	txpool.AddTx(txList[0])
	if 1 != txpool.all.Count(){
		t.Error("UNPASS: failed to add a duplicate tx")
	}else{
		t.Log("PASS: success to add a duplicate tx.")
	}

}

// Test Get a tx from txpool
func Test_GetTxs(t *testing.T){
	tx := mock_transactions(1)[0]
	if nil == tx{
		t.Error("UNPASS: mokc a transaction failed.")
	}

	txpool := NewTxPool(DefaultTxPoolConfig)
	if nil == txpool{
		t.Error("UNPASS: failed to create a txpool")
	}

	txpool.AddTx(tx)
	txList := txpool.GetTxs()
	if 1 == len(txList) && txList[0].Hash() == tx.Hash(){
		t.Log("PASS: success to get tx from txpool.")
	}else{
		t.Error("UNPASS: failed to get tx from txpool")
	}
}

// Test DelTxs txs from txpool
func Test_DelTxs(t *testing.T){
	txpool := NewTxPool(DefaultTxPoolConfig)
	if nil == txpool{
		t.Error("UNPASS: failed to create a txpool")
	}

	txpool.DelTxs()
	t.Log("PASS: success to del tx from txpool.")
}


