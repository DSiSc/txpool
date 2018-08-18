package common

import (
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"reflect"
)

// Lengths of hashes and addresses in bytes.
const (
	HashLength    = 32
	AddressLength = 20
)

// Address represents the 20 byte address of an Ethereum account.
type Address [AddressLength]byte

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

var (
	hashT    = reflect.TypeOf(Hash{})
	addressT = reflect.TypeOf(Address{})
)

func RlpHash(x interface{}) (h Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// CopyBytes returns an exact copy of the provided bytes.
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

func (u *Hash) Serialize(w io.Writer) error {
	_, err := w.Write(u[:])
	return err
}

func (u *Hash) ToArray() []byte {
	x := make([]byte, HashLength)
	for i := 0; i < HashLength; i++ {
		x[i] = byte(u[i])
	}

	return x
}
