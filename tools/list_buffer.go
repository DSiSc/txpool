package tools

import (
	"container/list"
	"errors"
	"github.com/DSiSc/craft/types"
	"time"
)

var (
	DuplicateError    = errors.New("duplicate insert")
	BufferIsFullError = errors.New("buffer is full")
)

// TimedTransaction contains a transaction with the time added to buffer
type TimedTransaction struct {
	Tx        *types.Transaction
	TimeStamp time.Time
}

// ListBuffer is a Tx list buffer implementation.
type ListBuffer struct {
	limit         uint64
	maxCacheTime  uint64
	len           int
	txs           map[types.Hash]*types.Transaction
	timedTxGroups map[types.Address]*list.List
}

// NewListBuffer create a Tx list buffer instance
func NewListBuffer(limit uint64, maxCacheTime uint64) *ListBuffer {
	return &ListBuffer{
		limit:         limit,
		maxCacheTime:  maxCacheTime,
		len:           0,
		timedTxGroups: make(map[types.Address]*list.List),
		txs:           make(map[types.Hash]*types.Transaction),
	}
}

// AddTx add an element to list buffer
func (self *ListBuffer) AddTx(tx *types.Transaction) error {
	hash := tx.Hash.Load().(types.Hash)
	if self.txs[hash] != nil {
		return DuplicateError
	} else {
		self.txs[hash] = tx
	}

	// insert timedTx into the correct index in self.timedTxGroups
	if self.timedTxGroups[*tx.Data.From] == nil {
		self.timedTxGroups[*tx.Data.From] = list.New()
	}
	sameFromTxs := self.timedTxGroups[*tx.Data.From]
	if self.insertOrReplace(sameFromTxs, tx) {
		return nil
	}

	// check limit
	if uint64(self.len) <= self.limit {
		return nil
	}

	// check timeout tx in self group
	if self.removeTimeOutTx(sameFromTxs) {
		return nil
	}

	// delete timeout tx in other group
	if self.RemoveTimeOutTx() {
		return nil
	}

	// remove last tx
	backTimedTx := sameFromTxs.Back().Value.(*TimedTransaction)
	self.RemoveTx(backTimedTx.Tx.Hash.Load().(types.Hash))
	if backTimedTx.Tx.Data.AccountNonce == tx.Data.AccountNonce {
		return BufferIsFullError
	}
	return nil
}

// GetTx get an element from list buffer
func (self *ListBuffer) GetTx(hash types.Hash) *types.Transaction {
	return self.txs[hash]
}

// RemoveElement remove an element from list buffer
func (self *ListBuffer) RemoveTx(hash types.Hash) {
	if elem := self.txs[hash]; elem != nil {
		delete(self.txs, hash)
		self.deleteTx(*elem.Data.From, elem.Data.AccountNonce)
		self.decLen()
	}
}

// RemoveTimeOutTx remove an timeout Tx from list buffer, return true if exists timeout Tx.
func (self *ListBuffer) RemoveTimeOutTx() bool {
	for _, timedTxGroup := range self.timedTxGroups {
		if self.removeTimeOutTx(timedTxGroup) {
			return true
		}
	}
	return false
}

// TimedTxGroups returns the tx groups of ListBuffer.
func (self *ListBuffer) TimedTxGroups() map[types.Address]*list.List {
	return self.timedTxGroups
}

// NonceInBuffer returns the nonce in buffer.
func (self *ListBuffer) NonceInBuffer(from types.Address) uint64 {
	if timedTxGroup := self.timedTxGroups[from]; timedTxGroup != nil {
		timedTx := timedTxGroup.Back().Value.(*TimedTransaction)
		return timedTx.Tx.Data.AccountNonce
	}
	return 0
}

// Len returns the number of txs of ListBuffer.
func (self *ListBuffer) Len() int {
	return self.len
}

// insert into group if not exist same nonce tx, else update the exist tx. return true if exists same nonce tx.
func (self *ListBuffer) insertOrReplace(sameFromTxs *list.List, tx *types.Transaction) bool {
	timedTx := &TimedTransaction{
		Tx:        tx,
		TimeStamp: time.Now(),
	}

	for e := sameFromTxs.Back(); e != nil; e = e.Prev() {
		eTx := e.Value.(*TimedTransaction)
		if eTx.Tx.Data.AccountNonce == tx.Data.AccountNonce {
			e.Value = timedTx
			return true
		}
		if eTx.Tx.Data.AccountNonce < tx.Data.AccountNonce {
			sameFromTxs.InsertAfter(timedTx, e)
			self.incLen()
			return false
		}
	}

	sameFromTxs.PushFront(timedTx)
	self.incLen()
	return false
}

// remove an timeout Tx from list buffer, return true if exists timeout Tx.
func (self *ListBuffer) removeTimeOutTx(timedTxGroup *list.List) bool {
	fontTx := timedTxGroup.Front().Value.(*TimedTransaction)
	if time.Now().After(fontTx.TimeStamp.Add(time.Duration(self.maxCacheTime) * time.Second)) {
		lastTx := timedTxGroup.Back().Value.(*TimedTransaction)
		self.RemoveTx(lastTx.Tx.Hash.Load().(types.Hash))
		return true
	} else {
		return false
	}
}

// delete Tx from self.timedTxGroups
func (self *ListBuffer) deleteTx(addr types.Address, nonce uint64) {
	if l := self.timedTxGroups[addr]; l != nil {
		firstE := l.Front()
		for ; firstE != nil; firstE = firstE.Next() {
			firstTx := firstE.Value.(*TimedTransaction)
			if firstTx.Tx.Data.AccountNonce > nonce {
				return
			}
			if firstTx.Tx.Data.AccountNonce == nonce {
				l.Remove(firstE)
				break
			}
		}
		if l.Len() <= 0 {
			delete(self.timedTxGroups, addr)
		}
	}
}

// increase length of buffer
func (self *ListBuffer) incLen() {
	self.len++
}

// decrease length of buffer
func (self *ListBuffer) decLen() {
	self.len--
}
