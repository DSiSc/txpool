// TxPool contains all currently known transactions.
// Transactions enter the pool when they are received from the network or submitted locally.
// They exit the pool when they are included in the blockchain.
// The pool seperate processable transactions (which can be applied to the current state) and future transactions.
// Transactions move between those two states over time as they are received and processed.

package txpool

import (
	"fmt"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"github.com/DSiSc/txpool/tools"
)

type TxsPool interface {
	// AddTx add a transaction to the txpool.
	AddTx(tx *types.Transaction) error

	// DelTxs delete the transactions which in processing queue.
	// Once a block was commited, transaction contained in the block can be removed.
	DelTxs() error

	// GetTxs gets the transactons which in pending status.
	GetTxs() []*types.Transaction
}

type TxPool struct {
	config TxPoolConfig
	all    *txLookup
	// TODO: signature block that the transactions belong
	process  []types.Hash
	txsQueue *tools.CycleQueue
}

// structure for tx lookup.
type txLookup struct {
	all map[types.Hash]*types.Transaction
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
	MaxTrxPerBlock uint64 // Maximum num of transactions a block
}

var DefaultTxPoolConfig = TxPoolConfig{
	GlobalSlots:    4096,
	MaxTrxPerBlock: 512,
}

// sanitize checks the provided user configurations and changes anything that's  unreasonable or unworkable.
func (config *TxPoolConfig) sanitize() TxPoolConfig {
	conf := *config
	if conf.GlobalSlots < 1 || conf.GlobalSlots > DefaultTxPoolConfig.GlobalSlots {
		log.Warn("Sanitizing invalid txpool global slots %d.", conf.GlobalSlots)
		conf.GlobalSlots = DefaultTxPoolConfig.GlobalSlots
	}
	if conf.MaxTrxPerBlock < 1 || conf.MaxTrxPerBlock > DefaultTxPoolConfig.MaxTrxPerBlock {
		log.Warn("Sanitizing invalid txpool max num of transactions a block %d.", conf.MaxTrxPerBlock)
		conf.MaxTrxPerBlock = DefaultTxPoolConfig.MaxTrxPerBlock
	}
	return conf
}

// NewTxPool creates a new transaction pool to gather, sort and filter inbound transactions from the network and local.
func NewTxPool(config TxPoolConfig) TxsPool {
	config = (&config).sanitize()
	// Create the transaction pool with its initial settings
	pool := &TxPool{
		config:   config,
		all:      newTxLookup(),
		txsQueue: tools.NewQueue(config.GlobalSlots, config.MaxTrxPerBlock),
		process:  make([]types.Hash, 0),
	}

	return pool
}

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) Get(hash types.Hash) *types.Transaction {
	return t.all[hash]
}

// Count returns the current number of items in the lookup.
func (t *txLookup) Count() int {
	return len(t.all)
}

// Add adds a transaction to the lookup.
func (t *txLookup) Add(tx *types.Transaction) {
	t.all[common.TxHash(tx)] = tx
}

// Remove removes a transaction from the lookup.
func (t *txLookup) Remove(hash types.Hash) {
	delete(t.all, hash)
}

// Get pending txs from txpool.
func (pool *TxPool) GetTxs() []*types.Transaction {
	txList := make([]*types.Transaction, 0)
	txs := pool.txsQueue.Consumer()
	for _, value := range txs {
		tx := value.(*types.Transaction)
		log.Info("Get tx %x form txpool.", common.TxHash(tx))
		txList = append(txList, tx)
		pool.process = append(pool.process, common.TxHash(tx))
		pool.all.Remove(common.TxHash(tx))
	}
	log.Info("Get txs %d form txpool.", len(txList))
	return txList
}

// Update processing queue, clean txs from process and all queue.
func (pool *TxPool) DelTxs() error {
	// TODO: ued in the future
	log.Info("Update txpool after the txs has been applied by producer.")
	for _, txHash := range pool.process {
		pool.all.Remove(txHash)
	}
	pool.process = make([]types.Hash, 0)
	return nil
}

func (pool *TxPool) addTx(tx *types.Transaction) {
	// Add to queue
	pool.txsQueue.Producer(tx)
	// Add to all
	pool.all.Add(tx)
}

// Adding transaction to the txpool
func (pool *TxPool) AddTx(tx *types.Transaction) error {
	if uint64(pool.all.Count()) >= pool.config.GlobalSlots {
		log.Error("Txpool has full.")
		return fmt.Errorf("txpool has full")
	}
	if nil != pool.all.Get(common.TxHash(tx)) {
		log.Error("The tx %v has exist, please confirm.", common.TxHash(tx))
		return fmt.Errorf("the tx %v has exist", common.TxHash(tx))
	}
	log.Info("Add tx %x to txpool.", common.TxHash(tx))
	pool.addTx(tx)
	return nil
}

func (pool *TxPool) GetTxByHash(hash types.Hash) *types.Transaction {
	txs := pool.all.Get(hash)
	if nil == txs {
		log.Warn("Txs [%v] not exist in pool.", hash)
		return nil
	}
	return txs
}
