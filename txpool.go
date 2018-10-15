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
	process  map[types.Address][]*types.Transaction
	txsQueue *tools.CycleQueue
	mu       sync.RWMutex
}

// structure for tx lookup.
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

// TxPoolConfig are the configuration parameters of the transaction pool.
type TxPoolConfig struct {
	GlobalSlots    uint64 // Maximum number of executable transaction slots for txpool
	MaxTrsPerBlock uint64 // Maximum num of transactions a block
}

var DefaultTxPoolConfig = TxPoolConfig{
	GlobalSlots:    4096,
	MaxTrsPerBlock: 512,
}

var GlobalTxsPool *TxPool

// sanitize checks the provided user configurations and changes anything that's  unreasonable or unworkable.
func (config *TxPoolConfig) sanitize() {
	if config.GlobalSlots < 1 || config.GlobalSlots > DefaultTxPoolConfig.GlobalSlots {
		log.Warn("Sanitizing invalid txpool global slots %d.", config.GlobalSlots)
		config.GlobalSlots = DefaultTxPoolConfig.GlobalSlots
	}
	if config.MaxTrsPerBlock < 1 || config.MaxTrsPerBlock > DefaultTxPoolConfig.MaxTrsPerBlock {
		log.Warn("Sanitizing invalid txpool max num of transactions a block %d.", config.MaxTrsPerBlock)
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
		process:  make(map[types.Address][]*types.Transaction, 0),
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
	txList := make([]*types.Transaction, 0)
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	txs := pool.txsQueue.Consumer()
	for _, value := range txs {
		tx := value.(*types.Transaction)
		log.Info("Get tx %x form txpool.", common.TxHash(tx))
		txList = append(txList, tx)
		pool.process[*tx.Data.From] = sortTxsByNonce(pool.process[*tx.Data.From], tx)
		pool.all.Remove(common.TxHash(tx))
	}
	log.Info("Get txs %d form txpool.", len(txList))
	monitor.JTMetrics.TxpoolOutgoingTx.Add(float64(len(txList)))
	return txList
}

// Update processing queue, clean txs from process and all queue.
func (pool *TxPool) DelTxs(txs []*types.Transaction) {
	log.Info("Update txpool after the txs has been applied by producer.")
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for i := 0; i < len(txs); i++ {
		txList := pool.process[*txs[i].Data.From]
		txHash := common.TxHash(txs[i])
		exist := false
		for j := 0; j < len(txList); j++ {
			hash := common.TxHash(txList[j])
			if bytes.Equal(txHash[:], hash[:]) {
				pool.process[*txs[i].Data.From] = append(
					pool.process[*txs[i].Data.From][:j], pool.process[*txs[i].Data.From][j+1:]...)
				exist = true
				break
			}
		}
		if !exist {
			log.Error("tx %x not exist in process, please confirm.", txHash)
		}
	}
}

// Adding transaction to the txpool
func (pool *TxPool) AddTx(tx *types.Transaction) error {
	hash := common.TxHash(tx)
	pool.mu.Lock()
	defer pool.mu.Unlock()
	monitor.JTMetrics.TxpoolIngressTx.Add(float64(1))
	if uint64(pool.all.Count()) >= pool.config.GlobalSlots {
		log.Error("Txpool has full.")
		monitor.JTMetrics.TxpoolDiscardedTx.Add(float64(1))
		return fmt.Errorf("txpool has full")
	}
	if nil != pool.all.Get(hash) {
		monitor.JTMetrics.TxpoolDuplacatedTx.Add(float64(1))
		log.Error("The tx %x has exist, please confirm.", hash)
		return fmt.Errorf("the tx %x has exist", hash)
	}
	pool.addTx(tx)
	monitor.JTMetrics.TxpoolPooledTx.Add(float64(1))
	return nil
}

func (pool *TxPool) addTx(tx *types.Transaction) {
	// Add to queue
	pool.txsQueue.Producer(tx)
	// Add to all
	pool.all.Add(tx)
}

func (pool *TxPool) GetTxByHash(hash types.Hash) *types.Transaction {
	txs := pool.all.Get(hash)
	if nil == txs {
		log.Warn("Tx %x not exist in pool.", hash)
		return nil
	}
	return txs
}

func sortTxsByNonce(txs []*types.Transaction, tx *types.Transaction) []*types.Transaction {
	// simple sort
	var index int
	newNonce := tx.Data.AccountNonce
	txsCount := len(txs)
	for index = 0; index < txsCount; index++ {
		if newNonce > txs[index].Data.AccountNonce {
			break
		}
	}
	temp := append([]*types.Transaction{}, txs[index:]...)
	txs = append(txs[:index], tx)
	txs = append(txs, temp...)
	return txs
}

func GetTxByHash(hash types.Hash) *types.Transaction {
	txs := GlobalTxsPool.all.Get(hash)
	if nil == txs {
		log.Warn("Tx %x not exist in pool.", hash)
		return nil
	}
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
	txs := GlobalTxsPool.process[address]
	if len(txs) > 0 && txs[0].Data.AccountNonce > defaultNonce {
		defaultNonce = txs[0].Data.AccountNonce
	}
	return defaultNonce
}
