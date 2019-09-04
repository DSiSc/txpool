// TxPool contains all currently known transactions.
// Transactions enter the pool when they are received from the network or submitted locally.
// They exit the pool when they are included in the blockchain.
// The pool seperate processable transactions (which can be applied to the current state) and future transactions.
// Transactions move between those two states over time as they are received and processed.

package txpool

import (
	"fmt"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/monitor"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/repository"
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
	config      TxPoolConfig
	txBuffer    *tools.ListBuffer
	chain       *repository.Repository
	mu          sync.RWMutex
	eventCenter types.EventCenter
}

// TxPoolConfig are the configuration parameters of the transaction pool.
type TxPoolConfig struct {
	GlobalSlots    uint64 // Maximum number of executable transaction slots for txpool
	MaxTrsPerBlock uint64 // Maximum num of transactions a block
	TxMaxCacheTime uint64 // Maximum cache time(second) of transactions in tx pool
}

var DefaultTxPoolConfig = TxPoolConfig{
	GlobalSlots:    40960,
	MaxTrsPerBlock: 20480,
	TxMaxCacheTime: 600,
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
	if config.TxMaxCacheTime <= 0 || config.TxMaxCacheTime > DefaultTxPoolConfig.TxMaxCacheTime {
		log.Warn("Sanitizing invalid txs pool max num cache time(%ds) of transactions in tx pool.", config.TxMaxCacheTime)
		config.TxMaxCacheTime = DefaultTxPoolConfig.TxMaxCacheTime
	}
}

// NewTxPool creates a new transaction pool to gather, sort and filter inbound transactions from the network and local.
func NewTxPool(config TxPoolConfig, eventCenter types.EventCenter) TxsPool {
	config.sanitize()
	// Create the transaction pool with its initial settings
	pool := &TxPool{
		config:      config,
		txBuffer:    tools.NewListBuffer(config.GlobalSlots, config.TxMaxCacheTime),
		eventCenter: eventCenter,
	}
	GlobalTxsPool = pool

	// subscribe block commit event
	pool.eventCenter.Subscribe(types.EventBlockCommitted, pool.updateChainInstance)
	pool.eventCenter.Subscribe(types.EventBlockWritten, pool.updateChainInstance)

	return pool
}

// Get pending txs from txpool.
func (pool *TxPool) GetTxs() []*types.Transaction {
	txList := make([]*types.Transaction, 0)
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	log.Debug("total number of tx in pool is: %d", pool.txBuffer.Len())
	for addr, l := range pool.txBuffer.TimedTxGroups() {
		startNonce := pool.getChainNonce(addr)
		log.Debug("account %x chain nonce %d VS %d", addr, startNonce, pool.txBuffer.NonceInBuffer(addr))
		for elem := l.Front(); elem != nil; {
			nextElem := elem.Next()
			timedTx := elem.Value.(*tools.TimedTransaction)
			if timedTx.Tx.Data.AccountNonce == startNonce {
				txList = append(txList, timedTx.Tx)
				startNonce++
				if uint64(len(txList)) >= pool.config.MaxTrsPerBlock {
					monitor.JTMetrics.TxpoolOutgoingTx.Add(float64(len(txList)))
					return txList
				}
			} else if timedTx.Tx.Data.AccountNonce < startNonce {
				pool.txBuffer.RemoveTx(timedTx.Tx.Hash.Load().(types.Hash))
			}
			elem = nextElem
		}
	}
	monitor.JTMetrics.TxpoolOutgoingTx.Add(float64(len(txList)))
	return txList
}

// Update processing queue, clean txs from process and all queue.
func (pool *TxPool) DelTxs(txs []*types.Transaction) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for _, tx := range txs {
		pool.txBuffer.RemoveOlderTx(*tx.Data.From, tx.Data.AccountNonce)
	}
}

// Adding transaction to the txpool
func (pool *TxPool) AddTx(tx *types.Transaction) error {
	hash := common.TxHash(tx)
	monitor.JTMetrics.TxpoolIngressTx.Add(float64(1))
	pool.mu.Lock()
	chainNonce := pool.getChainNonce(*tx.Data.From)
	if tx.Data.AccountNonce < chainNonce {
		pool.mu.Unlock()
		return fmt.Errorf("Tx %x nonce is too low", hash)
	}

	preLen := pool.txBuffer.Len()
	if err := pool.txBuffer.AddTx(tx); err != nil {
		pool.mu.Unlock()
		if err == tools.DuplicateError {
			monitor.JTMetrics.TxpoolDuplacatedTx.Add(float64(1))
			log.Debug("The tx %x has exist, please confirm.", hash)
			return fmt.Errorf("the tx %x has exist", hash)
		} else {
			return fmt.Errorf("Tx pool is full, will discard tx %x. ", hash)
		}
	}

	if pool.txBuffer.Len() <= preLen {
		log.Error("Tx pool is full, have discard some tx.")
		monitor.JTMetrics.TxpoolDiscardedTx.Add(float64(1))
	} else {
		monitor.JTMetrics.TxpoolPooledTx.Add(float64(1))
	}
	pool.mu.Unlock()
	log.Debug("tx num in pool: %d", pool.txBuffer.Len())
	pool.eventCenter.Notify(types.EventAddTxToTxPool, tx)
	return nil
}

func GetTxByHash(hash types.Hash) *types.Transaction {
	GlobalTxsPool.mu.RLock()
	defer GlobalTxsPool.mu.RUnlock()
	if txElem := GlobalTxsPool.txBuffer.GetTx(hash); txElem != nil {
		return txElem
	}
	return nil
}

func GetPoolNonce(address types.Address) uint64 {
	GlobalTxsPool.mu.RLock()
	defer GlobalTxsPool.mu.RUnlock()
	return GlobalTxsPool.txBuffer.NonceInBuffer(address)
}

// get account's nonce from chain
func (pool *TxPool) getChainNonce(address types.Address) uint64 {
	if nil == pool.chain {
		pool.updateChainInstanceWithoutLock()
	}
	return pool.chain.GetNonce(address)
}

// update chain instance after committing block
func (pool *TxPool) updateChainInstance(event interface{}) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.updateChainInstanceWithoutLock()
}

// update chain instance after committing block
func (pool *TxPool) updateChainInstanceWithoutLock() {
	if latestRepo, err := repository.NewLatestStateRepository(); err == nil {
		pool.chain = latestRepo
	} else {
		log.Error("failed to get latest blockchain, as: %v. We will panic tx pool, as error is not recoverable", err)
		panic(fmt.Sprintf("failed to get latest blockchain, as: %v", err))
	}
}
