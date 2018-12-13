// TxPool contains all currently known transactions.
// Transactions enter the pool when they are received from the network or submitted locally.
// They exit the pool when they are included in the blockchain.
// The pool seperate processable transactions (which can be applied to the current state) and future transactions.
// Transactions move between those two states over time as they are received and processed.

package txpool

import (
	"bytes"
	"fmt"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/monitor"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"github.com/DSiSc/txpool/tools"
	"sync"
)

type TxsPool interface {
	// AddTx add a transaction to the txpool.
	AddTx(tx *types.Transaction) error

	// DelTxs delete the transactions which in processing queue.
	// Once a block was committed, transaction contained in the block can be removed.
	DelTxs(txs []*types.Transaction)

	// GetTxs gets the transactions which in pending status.
	GetTxs() []*types.Transaction
}

type TxPool struct {
	config   TxPoolConfig
	all      *txLookup
	process  map[types.Address]*accountLookup
	txsQueue *tools.CycleQueue
	mu       sync.RWMutex
}

// struct for tx lookup.
type txLookup struct {
	all  map[types.Hash]*types.Transaction
	lock sync.RWMutex
}

// newTxLookup returns a new txLookup structure.
func newTxLookup() *txLookup {
	return &txLookup{
		all: make(map[types.Hash]*types.Transaction),
	}
}

// struct for lookup by account
type accountLookup struct {
	hashMap  map[types.Hash]*types.Transaction
	maxNonce uint64
	lock     sync.RWMutex
}

// newTxLookup returns a new txLookup structure.
func newAccountLookup() *accountLookup {
	return &accountLookup{
		hashMap:  make(map[types.Hash]*types.Transaction),
		maxNonce: 0,
	}
}

func (t *accountLookup) Add(tx *types.Transaction, hash types.Hash) {
	t.lock.Lock()
	if _, ok := t.hashMap[hash]; !ok {
		if tx.Data.AccountNonce > t.maxNonce {
			t.maxNonce = tx.Data.AccountNonce
		}
		t.hashMap[hash] = tx
	} else {
		log.Warn("tx %x has exist in pool process.", hash)
	}
	t.lock.Unlock()
}

func (t *accountLookup) Delete(hash types.Hash) {
	t.lock.Lock()
	if _, ok := t.hashMap[hash]; ok {
		tx := t.hashMap[hash]
		delete(t.hashMap, hash)
		if tx.Data.AccountNonce >= t.maxNonce {
			t.maxNonce = 0
			for _, value := range t.hashMap {
				if value.Data.AccountNonce > t.maxNonce {
					t.maxNonce = value.Data.AccountNonce
				}
			}
		}
	}
	log.Warn("tx %x not exist in process.", hash)
	t.lock.Unlock()
}

// TxPoolConfig are the configuration parameters of the transaction pool.
type TxPoolConfig struct {
	GlobalSlots    uint64 // Maximum number of executable transaction slots for txpool
	MaxTrsPerBlock uint64 // Maximum num of transactions a block
}

var DefaultTxPoolConfig = TxPoolConfig{
	GlobalSlots:    40960,
	MaxTrsPerBlock: 20480,
}

var GlobalTxsPool *TxPool

// sanitize checks the provided user configurations and changes anything that's  unreasonable or unworkable.
func (config *TxPoolConfig) sanitize() {
	if config.GlobalSlots < 1 || config.GlobalSlots > DefaultTxPoolConfig.GlobalSlots {
		log.Warn("Sanitizing invalid txs pool global slots %d.", config.GlobalSlots)
		config.GlobalSlots = DefaultTxPoolConfig.GlobalSlots
	}
	if config.MaxTrsPerBlock < 1 || config.MaxTrsPerBlock > DefaultTxPoolConfig.MaxTrsPerBlock {
		log.Warn("Sanitizing invalid txs pool max num of transactions a block %d.", config.MaxTrsPerBlock)
		config.MaxTrsPerBlock = DefaultTxPoolConfig.MaxTrsPerBlock
	}
}

// NewTxPool creates a new transaction pool to gather, sort and filter inbound transactions from the network and local.
func NewTxPool(config TxPoolConfig) TxsPool {
	config.sanitize()
	// Create the transaction pool with its initial settings
	pool := &TxPool{
		config:   config,
		all:      newTxLookup(),
		txsQueue: tools.NewQueue(config.GlobalSlots, config.MaxTrsPerBlock),
		process:  make(map[types.Address]*accountLookup),
	}
	GlobalTxsPool = pool
	return pool
}

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) Get(hash types.Hash) *types.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.all[hash]
}

// Count returns the current number of items in the lookup.
func (t *txLookup) Count() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return len(t.all)
}

// Add adds a transaction to the lookup.
func (t *txLookup) Add(tx *types.Transaction) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.all[common.TxHash(tx)] = tx
}

// Remove removes a transaction from the lookup.
func (t *txLookup) Remove(hash types.Hash) {
	t.lock.Lock()
	defer t.lock.Unlock()
	delete(t.all, hash)
}

// Get pending txs from txpool.
func (pool *TxPool) GetTxs() []*types.Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	txList := pool.txsQueue.Consumer()
	log.Info("get txs from pool is %d.", len(txList))
	for _, tx := range txList {
		if nil == pool.process[*tx.Data.From] {
			pool.process[*tx.Data.From] = newAccountLookup()
		}
		pool.process[*tx.Data.From].Add(tx, common.TxHash(tx))
		pool.all.Remove(common.TxHash(tx))
		log.Debug("Get tx %x form tx pool.", common.TxHash(tx))
	}
	log.Info("fetch %d txs, now pool reside %d txs with ppos:%d and gpos:%d.",
		len(txList), pool.txsQueue.Count(), pool.txsQueue.GetPpos(), pool.txsQueue.GetGpos())
	monitor.JTMetrics.TxpoolOutgoingTx.Add(float64(len(txList)))
	return txList
}

// Update processing queue, clean txs from process and all queue.
func (pool *TxPool) DelTxs(txs []*types.Transaction) {
	for i := 0; i < len(txs); i++ {
		txHash := common.TxHash(txs[i])
		pool.mu.Lock()
		if accountTxsMap, ok := pool.process[*txs[i].Data.From]; !ok {
			log.Warn("txs %x not exist in process queue.", txHash)
		} else {
			accountTxsMap.lock.Lock()
			if _, ok := accountTxsMap.hashMap[txHash]; ok {
				log.Debug("Delete tx %x from pool after commit block.", txHash)
				delete(accountTxsMap.hashMap, txHash)
			} else {
				log.Warn("tx %x not exist in process hashMap, please confirm.", txHash)
			}
			accountTxsMap.lock.Unlock()
		}
		pool.all.Remove(txHash)
		pool.txsQueue.SetDiscarding(txHash)
		pool.mu.Unlock()
	}
}

// Adding transaction to the txpool
func (pool *TxPool) AddTx(tx *types.Transaction) error {
	hash := common.TxHash(tx)
	monitor.JTMetrics.TxpoolIngressTx.Add(float64(1))
	if uint64(pool.all.Count()) >= pool.config.GlobalSlots {
		log.Error("Tx pool has full, which defined %d, but have %d.",
			pool.config.GlobalSlots, uint64(pool.all.Count()))
		monitor.JTMetrics.TxpoolDiscardedTx.Add(float64(1))
		return fmt.Errorf("txpool has full")
	}
	if nil != pool.all.Get(hash) {
		monitor.JTMetrics.TxpoolDuplacatedTx.Add(float64(1))
		log.Debug("The tx %x has exist, please confirm.", hash)
		return fmt.Errorf("the tx %x has exist", hash)
	}
	pool.addTx(tx)
	log.Debug("now txpool count is %d txs and ppos:%d and gpos:%d.",
		pool.txsQueue.Count(), pool.txsQueue.GetPpos(), pool.txsQueue.GetGpos())
	monitor.JTMetrics.TxpoolPooledTx.Add(float64(1))
	return nil
}

func (pool *TxPool) addTx(tx *types.Transaction) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	// Add to queue
	pool.txsQueue.Producer(tx)
	// Add to all
	pool.all.Add(tx)
}

func GetTxByHash(hash types.Hash) *types.Transaction {
	txs := GlobalTxsPool.all.Get(hash)
	if nil == txs {
		log.Warn("Tx %x not exist in all queue.", hash)
		for _, value := range GlobalTxsPool.process {
			value.lock.RLock()
			if nil != value {
				if tx, ok := value.hashMap[hash]; ok {
					value.lock.RUnlock()
					return tx
				}
			}
			value.lock.RUnlock()
		}
		log.Warn("Tx %x not exist in process queue.", hash)
	}
	log.Warn("Tx %x not exist in txpool.", hash)
	return txs
}

func GetPoolNonce(address types.Address) uint64 {
	defaultNonce := uint64(0)
	GlobalTxsPool.all.lock.RLock()
	for _, tx := range GlobalTxsPool.all.all {
		txFrom := *tx.Data.From
		if bytes.Equal(address[:], txFrom[:]) && tx.Data.AccountNonce > defaultNonce {
			defaultNonce = tx.Data.AccountNonce
		}
	}
	GlobalTxsPool.all.lock.RUnlock()
	if txs, ok := GlobalTxsPool.process[address]; ok {
		txs.lock.RLock()
		if txs.maxNonce > defaultNonce {
			defaultNonce = txs.maxNonce
		}
		txs.lock.RUnlock()
	}
	return defaultNonce
}
