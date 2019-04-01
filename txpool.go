// TxPool contains all currently known transactions.
// Transactions enter the pool when they are received from the network or submitted locally.
// They exit the pool when they are included in the blockchain.
// The pool seperate processable transactions (which can be applied to the current state) and future transactions.
// Transactions move between those two states over time as they are received and processed.

package txpool

import (
	"fmt"
	"github.com/DSiSc/blockchain"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/monitor"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/txpool/common"
	"github.com/DSiSc/txpool/tools"
	"sync"
	"time"
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
	nonceBuffer map[types.Address]uint64
	chain       *blockchain.BlockChain
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
		txBuffer:    tools.NewListBuffer(),
		nonceBuffer: make(map[types.Address]uint64),
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
	tmpNonceCache := make(map[types.Address]uint64)
	txList := make([]*types.Transaction, 0)
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	if element := pool.txBuffer.Front(); element != nil {
	BEGIN:
		tx := element.Value().(*types.Transaction)
		from := *tx.Data.From
		txAccountNonce := tx.Data.AccountNonce

		// check nonce
		if cachedNonce, ok := tmpNonceCache[from]; ok {
			//check cached nonce
			if (txAccountNonce - cachedNonce) == 1 {
				tmpNonceCache[from] = txAccountNonce
				txList = append(txList, tx)
			} else if txAccountNonce <= cachedNonce {
				pool.delTx(tx)
			}
		} else {
			// check chain nonce
			chainNonce := pool.getChainNonce(from)
			if txAccountNonce == chainNonce {
				tmpNonceCache[from] = txAccountNonce
				txList = append(txList, tx)
			} else if txAccountNonce < chainNonce {
				pool.delTx(tx)
			}
		}

		// retrieve next tx
		element = element.Next()
		if uint64(len(txList)) < pool.config.MaxTrsPerBlock && element != nil {
			goto BEGIN
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
		pool.delTx(tx)
	}
}

// Adding transaction to the txpool
func (pool *TxPool) AddTx(tx *types.Transaction) error {
	hash := common.TxHash(tx)
	monitor.JTMetrics.TxpoolIngressTx.Add(float64(1))
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if uint64(pool.txBuffer.Len()) >= pool.config.GlobalSlots {
		monitor.JTMetrics.TxpoolDiscardedTx.Add(float64(1))
		font := pool.txBuffer.Front()
		// delete timeout tx
		if uint64(time.Now().Sub(font.CreationTime()).Seconds()) > pool.config.TxMaxCacheTime {
			fontTx := font.Value().(*types.Transaction)
			pool.delTx(fontTx)
		} else {
			log.Error("Tx pool has full, which defined %d, but have %d.",
				pool.config.GlobalSlots, uint64(pool.txBuffer.Len()))
			return fmt.Errorf("txpool has full")
		}
	}
	if nil != pool.addTx(tx) {
		monitor.JTMetrics.TxpoolDuplacatedTx.Add(float64(1))
		log.Debug("The tx %x has exist, please confirm.", hash)
		return fmt.Errorf("the tx %x has exist", hash)
	}
	pool.eventCenter.Notify(types.EventAddTxToTxPool, hash)
	monitor.JTMetrics.TxpoolPooledTx.Add(float64(1))
	return nil
}

func GetTxByHash(hash types.Hash) *types.Transaction {
	GlobalTxsPool.mu.RLock()
	defer GlobalTxsPool.mu.RUnlock()
	if txElem := GlobalTxsPool.txBuffer.GetElement(hash); txElem != nil {
		return txElem.Value().(*types.Transaction)
	}
	return nil
}

func GetPoolNonce(address types.Address) uint64 {
	GlobalTxsPool.mu.RLock()
	defer GlobalTxsPool.mu.RUnlock()
	if _, ok := GlobalTxsPool.nonceBuffer[address]; ok {
		return GlobalTxsPool.nonceBuffer[address]
	}
	return 0
}

// add tx to buffer and cache the nonce
func (pool *TxPool) addTx(tx *types.Transaction) error {
	if nonce, ok := pool.nonceBuffer[*tx.Data.From]; !ok || nonce < tx.Data.AccountNonce {
		pool.nonceBuffer[*tx.Data.From] = tx.Data.AccountNonce
	}
	return pool.txBuffer.AddElement(tx.Hash.Load(), tx)
}

// delete tx from buffer and remove unused nonce cache
func (pool *TxPool) delTx(tx *types.Transaction) {
	if nonce, ok := pool.nonceBuffer[*tx.Data.From]; ok && nonce <= tx.Data.AccountNonce {
		delete(pool.nonceBuffer, *tx.Data.From)
	}
	pool.txBuffer.RemoveElementByKey(tx.Hash.Load())
}

// get account's nonce from chain
func (pool *TxPool) getChainNonce(address types.Address) uint64 {
	return pool.chain.GetNonce(address)
}

// update chain instance after committing block
func (pool *TxPool) updateChainInstance(event interface{}) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if latestChain, err := blockchain.NewLatestStateBlockChain(); err == nil {
		pool.chain = latestChain
	} else {
		log.Error("failed to get latest blockchain, as: %v. We will panic tx pool, as error is not recoverable", err)
		panic(fmt.Sprintf("failed to get latest blockchain, as: %v", err))
	}
}
