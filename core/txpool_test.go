package core

import (
	"testing"
)

// Test new a txpool
func Test_NewTxPool(t *testing.T){
	var slot uint64 = 64
	mock_config := TxPoolConfig{
		GlobalSlots: slot,
	}

	txpool := NewTxPool(mock_config)
	if nil != txpool{
		t.Log("PASS txpool")
	}else{
		t.Error("NO PASS")
	}

	if txpool.config.GlobalSlots == slot {
		t.Log("PASS GlobalSlots")
	}else{
		t.Error("NO PASS GlobalSlots")
	}
}

