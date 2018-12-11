package common

import (
	"encoding/json"
	gconf "github.com/DSiSc/craft/config"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/crypto-suite/crypto/sha3"
	"math/big"
)

// Sum returns the first 32 bytes of hash of the bz.
func Sum(bz []byte) []byte {
	var alg string
	if value, ok := gconf.GlobalConfig.Load(gconf.HashAlgName); ok {
		alg = value.(string)
	} else {
		alg = "SHA256"
	}
	hasher := sha3.NewHashByAlgName(alg)
	hasher.Write(bz)
	hash := hasher.Sum(nil)
	return hash[:types.HashLength]
}

func TxHash(tx *types.Transaction) types.Hash {
	if hash := tx.Hash.Load(); hash != nil {
		return hash.(types.Hash)
	}
	hashData := types.TxData{
		AccountNonce: tx.Data.AccountNonce,
		Price:        tx.Data.Price,
		GasLimit:     tx.Data.GasLimit,
		Recipient:    tx.Data.Recipient,
		Amount:       tx.Data.Amount,
		Payload:      tx.Data.Payload,
		V:            tx.Data.V,
		R:            tx.Data.R,
		S:            tx.Data.S,
	}
	jsonByte, _ := json.Marshal(hashData)
	sumByte := Sum(jsonByte)
	var temp types.Hash
	copy(temp[:], sumByte)
	tx.Hash.Store(temp)
	return temp
}

func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

// New a transaction
func newTransaction(nonce uint64, to *types.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, from *types.Address) *types.Transaction {
	if len(data) > 0 {
		data = CopyBytes(data)
	}
	d := types.TxData{
		AccountNonce: nonce,
		Recipient:    to,
		From:         from,
		Payload:      data,
		Amount:       new(big.Int),
		GasLimit:     gasLimit,
		Price:        new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &types.Transaction{Data: d}
}

func NewTransaction(nonce uint64, to types.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, from types.Address) *types.Transaction {
	return newTransaction(nonce, &to, amount, gasLimit, gasPrice, data, &from)
}
