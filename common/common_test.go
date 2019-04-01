package common

import (
	"fmt"
	"github.com/DSiSc/craft/types"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

var emptyTx *types.Transaction

func TestNewTransaction(t *testing.T) {
	assert := assert.New(t)
	b := types.Address{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
	}
	emptyTx = NewTransaction(
		0,
		b,
		big.NewInt(0),
		0,
		big.NewInt(0),
		b[:10],
		b,
	)
	assert.NotNil(emptyTx)
	assert.Equal(emptyTx.Data.From, &b)
	assert.Equal(emptyTx.Data.Recipient, &b)
	assert.Equal(emptyTx.Data.AccountNonce, uint64(0))
	assert.Equal(emptyTx.Data.GasLimit, uint64(0))
	assert.Equal(emptyTx.Data.Price, big.NewInt(0))
}

func TestCopyBytes(t *testing.T) {
	s := CopyBytes(nil)
	assert.Nil(t, s)
	b := []byte{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
	}
	c := CopyBytes(b)
	assert.Equal(t, b, c)
}

func TestTxHash(t *testing.T) {
	assert := assert.New(t)
	b := types.Address{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
	}
	emptyTx = NewTransaction(
		0,
		b,
		big.NewInt(0),
		0,
		big.NewInt(0),
		b[:10],
		b,
	)
	exceptHash := types.Hash{
		0x23, 0xd1, 0xda, 0xdc, 0xaf, 0xbf, 0x97, 0xaf, 0xb1, 0x79, 0xf, 0xfd, 0xb8, 0xc7, 0xb7, 0x51,
		0xc0, 0x81, 0xa6, 0x43, 0x9c, 0xd2, 0x1d, 0xaf, 0x89, 0x2a, 0x60, 0x63, 0x89, 0x42, 0x38, 0x29,
	}
	txHash := TxHash(emptyTx)
	assert.Equal(exceptHash, txHash)

	exceptHash1 := TxHash(emptyTx)
	assert.Equal(exceptHash, exceptHash1)
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
		tx := NewTransaction(uint64(i), to[i], amount, uint64(i), amount, nil, to[i])
		txList = append(txList, tx)
	}
	return txList
}

func TestHash(t *testing.T) {
	assert := assert.New(t)
	//tx := mock_transactions(1)
	hash := TxHash(mock_transactions(1)[0])
	fmt.Printf("%x\n", hash)
	assert.NotNil(hash)
}

func TestHexToAddress(t *testing.T) {
	addHex := "333c3310824b7c685133f2bedb2ca4b8b4df633d"
	address := HexToAddress(addHex)
	b := types.Address{
		0x33, 0x3c, 0x33, 0x10, 0x82, 0x4b, 0x7c, 0x68, 0x51, 0x33,
		0xf2, 0xbe, 0xdb, 0x2c, 0xa4, 0xb8, 0xb4, 0xdf, 0x63, 0x3d,
	}
	assert.Equal(t, b, address)
}
