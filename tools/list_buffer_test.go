package tools

import (
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

var (
	mockHash  = common.HexToHash("0x776a2bbddcb56d8bc5a97ca8058a76fa5bb27b2a589c80cf508b86d083bdd191")
	mockHash1 = common.HexToHash("0x776a2bbddcb56d8bc5a97ca8058a76fa5bb27b2a589c80cf508b86d083bdd190")
	mockHash2 = common.HexToHash("0x776a2bbddcb56d8bc5a97ca8058a76fa5bb27b2a589c80cf508b86d083bdd189")
	mockAddr  = common.HexToAddress("0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b")
)

func mockTransaction() *types.Transaction {
	return mockTransaction1(mockHash, mockAddr)
}

func mockTransaction1(hash types.Hash, from types.Address) *types.Transaction {
	tx := &types.Transaction{
		Data: types.TxData{
			From: &from,
		},
	}
	tx.Hash.Store(hash)
	return tx
}

func TestNewListBuffer(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer(100, 100)
	assert.NotNil(lb)
	assert.Equal(0, lb.Len())
}

func TestListBuffer_AddElement(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer(100, 100)
	assert.NotNil(lb)
	tx := mockTransaction()
	assert.Nil(lb.AddTx(tx))
	assert.NotNil(lb.AddTx(tx))
}

func TestListBuffer_GetElement(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer(100, 100)
	assert.NotNil(lb)
	tx := mockTransaction()
	assert.Nil(lb.AddTx(tx))
	assert.NotNil(lb.GetTx(tx.Hash.Load().(types.Hash)))
}

func TestListBuffer_Front(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer(100, 100)
	assert.NotNil(lb)
	assert.Nil(lb.AddTx(mockTransaction1(common.HexToHash("0x776a2bbddcb56d8bc5a97ca8058a76fa5bb27b2a589c80cf508b86d083bdd191"), common.HexToAddress("0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b"))))
	assert.Nil(lb.AddTx(mockTransaction1(common.HexToHash("0x676a2bbddcb56d8bc5a97ca8058a76fa5bb27b2a589c80cf508b86d083bdd191"), common.HexToAddress("0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b"))))
	assert.Nil(lb.AddTx(mockTransaction1(common.HexToHash("0x576a2bbddcb56d8bc5a97ca8058a76fa5bb27b2a589c80cf508b86d083bdd191"), common.HexToAddress("0xb94f5374fce5edbc8e2a8697c15331677e6ebf0b"))))
	e := lb.TimedTxGroups()
	assert.Equal(2, len(e))
}

func TestListBuffer_Len(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer(100, 100)
	assert.NotNil(lb)
	tx := mockTransaction()
	assert.Nil(lb.AddTx(tx))
	assert.Equal(1, lb.Len())
}

func TestListBuffer_AddTx(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer(100, 100)
	assert.NotNil(lb)
	tx := mockTransaction()
	assert.Nil(lb.AddTx(tx))
	assert.Equal(1, lb.Len())
	assert.Equal(1, len(lb.txs))

	tx1 := mockTransaction1(mockHash1, *tx.Data.From)
	tx1.Data.Price = big.NewInt(2)
	assert.Nil(lb.AddTx(tx1))
	assert.Equal(1, lb.Len())
	assert.Equal(1, len(lb.txs))
}

func TestListBuffer_RemoveOlderTx(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer(100, 100)
	assert.NotNil(lb)
	tx := mockTransaction1(mockHash, mockAddr)
	assert.Nil(lb.AddTx(tx))

	tx = mockTransaction1(mockHash1, mockAddr)
	tx.Data.AccountNonce = 1
	assert.Nil(lb.AddTx(tx))

	tx = mockTransaction1(mockHash2, mockAddr)
	tx.Data.AccountNonce = 2
	assert.Nil(lb.AddTx(tx))

	lb.RemoveOlderTx(mockAddr, 1)
	assert.Equal(1, lb.Len())
	assert.Equal(1, len(lb.txs))
	assert.Equal(1, lb.timedTxGroups[mockAddr].Len())
}
