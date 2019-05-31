package txpool

import (
	"errors"
	"fmt"
	"github.com/DSiSc/craft/log"
	"github.com/DSiSc/craft/types"
	"github.com/DSiSc/monkey"
	"github.com/DSiSc/repository"
	"github.com/DSiSc/txpool/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"
)

type MockEvent struct {
	m           sync.RWMutex
	Subscribers map[types.EventType]map[types.Subscriber]types.EventFunc
}

func NewMockEvent() types.EventCenter {
	return &MockEvent{
		Subscribers: make(map[types.EventType]map[types.Subscriber]types.EventFunc),
	}
}

//  adds a new subscriber to Event.
func (e *MockEvent) Subscribe(eventType types.EventType, eventFunc types.EventFunc) types.Subscriber {
	e.m.Lock()
	defer e.m.Unlock()

	sub := make(chan interface{})
	_, ok := e.Subscribers[eventType]
	if !ok {
		e.Subscribers[eventType] = make(map[types.Subscriber]types.EventFunc)
	}
	e.Subscribers[eventType][sub] = eventFunc

	return sub
}

// UnSubscribe removes the specified subscriber
func (e *MockEvent) UnSubscribe(eventType types.EventType, subscriber types.Subscriber) (err error) {
	e.m.Lock()
	defer e.m.Unlock()

	subEvent, ok := e.Subscribers[eventType]
	if !ok {
		err = errors.New("event type not exist")
		return
	}

	delete(subEvent, subscriber)
	close(subscriber)

	return
}

// Notify subscribers that Subscribe specified event
func (e *MockEvent) Notify(eventType types.EventType, value interface{}) (err error) {

	e.m.RLock()
	defer e.m.RUnlock()

	subs, ok := e.Subscribers[eventType]
	if !ok {
		err = errors.New("event type not register")
		return
	}

	switch value.(type) {
	case error:
		log.Error("Receive errors is [%v].", value)
	}
	log.Info("Receive eventType is [%d].", eventType)

	for _, event := range subs {
		go e.NotifySubscriber(event, value)
	}
	return nil
}

func (e *MockEvent) NotifySubscriber(eventFunc types.EventFunc, value interface{}) {
	if eventFunc == nil {
		return
	}

	// invoke subscriber event func
	eventFunc(value)

}

//Notify all event subscribers
func (e *MockEvent) NotifyAll() (errs []error) {
	e.m.RLock()
	defer e.m.RUnlock()

	for eventType, _ := range e.Subscribers {
		if err := e.Notify(eventType, nil); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// unsubscribe all event and subscriber elegant
func (e *MockEvent) UnSubscribeAll() {
	e.m.Lock()
	defer e.m.Unlock()
	for eventtype, _ := range e.Subscribers {
		subs, ok := e.Subscribers[eventtype]
		if !ok {
			continue
		}
		for subscriber, _ := range subs {
			delete(subs, subscriber)
			close(subscriber)
		}
	}
	// TODO: open it when txswitch and blkswith stop complete
	//e.Subscribers = make(map[types.EventType]map[types.Subscriber]types.EventFunc)
	return
}

// mock a config for txpool
func mock_txpool_config(slot uint64) TxPoolConfig {
	mock_config := TxPoolConfig{
		GlobalSlots: slot,
	}
	return mock_config
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
		tx := common.NewTransaction(0, to[i], amount, uint64(i), amount, nil, common.HexToAddress(fmt.Sprintf("0x%d", i)))
		txList = append(txList, tx)
	}
	return txList
}

// Test new a txpool
func Test_NewTxPool(t *testing.T) {
	assert := assert.New(t)

	mock_config := mock_txpool_config(DefaultTxPoolConfig.GlobalSlots - 1)
	txpool := NewTxPool(mock_config, NewMockEvent())
	assert.NotNil(txpool)
	instance := txpool.(*TxPool)
	assert.Equal(DefaultTxPoolConfig.GlobalSlots-1, instance.config.GlobalSlots, "they should be equal")

	mock_config = mock_txpool_config(DefaultTxPoolConfig.GlobalSlots + 1)
	txpool = NewTxPool(mock_config, NewMockEvent())
	instance = txpool.(*TxPool)
	assert.Equal(DefaultTxPoolConfig.GlobalSlots, instance.config.GlobalSlots, "they should be equal")
}

// Test add a tx to txpool
func Test_AddTx(t *testing.T) {
	assert := assert.New(t)

	txList := mock_transactions(3)
	assert.NotNil(txList)

	events := NewMockEvent()
	events.Subscribe(types.EventAddTxToTxPool, func(v interface{}) {
		if nil != v {
			log.Info("add tx %v success.", v)
		}
	})

	var MockTxPoolConfig = TxPoolConfig{
		GlobalSlots:    2,
		MaxTrsPerBlock: 2,
	}

	txpool := NewTxPool(MockTxPoolConfig, events)
	assert.NotNil(txpool)

	err := txpool.AddTx(txList[0])
	assert.Nil(err)

	instance := txpool.(*TxPool)

	// add duplicate tx to txpool
	err = txpool.AddTx(txList[0])
	assert.NotNil(err)
	assert.Equal(1, instance.txBuffer.Len(), "they should be equal")

	err = txpool.AddTx(txList[1])
	assert.Nil(err)
	assert.Equal(2, instance.txBuffer.Len(), "they should be equal")

	err = txpool.AddTx(txList[2])
	assert.NotNil(err)

	instance.config.TxMaxCacheTime = 1
	time.Sleep(2 * time.Second)
	err = txpool.AddTx(txList[2])
	assert.Nil(err)
	assert.Equal(2, instance.txBuffer.Len())
}

// Test Get a tx from txpool
func Test_GetTxs(t *testing.T) {
	assert := assert.New(t)
	tx := mock_transactions(1)[0]
	assert.NotNil(tx)

	txpool := NewTxPool(DefaultTxPoolConfig, NewMockEvent())
	assert.NotNil(txpool)
	assert.Nil(txpool.AddTx(tx))

	chain := &repository.Repository{}
	monkey.Patch(repository.NewLatestStateRepository, func() (*repository.Repository, error) {
		return chain, nil
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(chain), "GetNonce", func(*repository.Repository, types.Address) uint64 {
		return 0
	})
	returnedTx := txpool.GetTxs()
	assert.NotNil(returnedTx)
	assert.Equal(1, len(returnedTx))

	monkey.PatchInstanceMethod(reflect.TypeOf(chain), "GetNonce", func(*repository.Repository, types.Address) uint64 {
		return 1
	})
	returnedTx = txpool.GetTxs()
	assert.Equal(0, len(returnedTx))
}

// Test DelTxs txs from txpool
func Test_DelTxs(t *testing.T) {
	assert := assert.New(t)
	txs := mock_transactions(3)

	txpool := NewTxPool(DefaultTxPoolConfig, NewMockEvent())
	assert.NotNil(txpool)
	pool := txpool.(*TxPool)
	pool.AddTx(txs[0])
	assert.Equal(1, pool.txBuffer.Len())

	pool.DelTxs([]*types.Transaction{txs[1]})
	assert.Equal(1, pool.txBuffer.Len())

	pool.DelTxs([]*types.Transaction{txs[0]})
	assert.Equal(0, pool.txBuffer.Len())

	pool.AddTx(txs[0])
	pool.AddTx(txs[1])
	pool.AddTx(txs[2])
	assert.Equal(3, pool.txBuffer.Len())
	pool.DelTxs([]*types.Transaction{txs[1], txs[2]})
	assert.Equal(1, pool.txBuffer.Len())
	pool.DelTxs([]*types.Transaction{txs[0]})
	assert.Equal(0, pool.txBuffer.Len())
}

func TestGetTxByHash(t *testing.T) {
	assert := assert.New(t)
	tx := mock_transactions(10)[9]
	assert.NotNil(tx)
	txpool := NewTxPool(DefaultTxPoolConfig, NewMockEvent())
	assert.NotNil(txpool)
	pool := txpool.(*TxPool)

	exceptTx := GetTxByHash(common.TxHash(tx))
	assert.Nil(exceptTx)

	// try to get exist tx
	err := pool.AddTx(tx)
	assert.Nil(err)
	exceptTx = GetTxByHash(common.TxHash(tx))
	assert.Equal(common.TxHash(tx), common.TxHash(exceptTx))
}

func TestGetPoolNonce(t *testing.T) {
	assert := assert.New(t)
	chain := &repository.Repository{}
	monkey.Patch(repository.NewLatestStateRepository, func() (*repository.Repository, error) {
		return chain, nil
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(chain), "GetNonce", func(*repository.Repository, types.Address) uint64 {
		return 0
	})
	var txs []*types.Transaction
	txpool := NewTxPool(DefaultTxPoolConfig, NewMockEvent())

	mockFromAddress := types.Address{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
	}
	for i := 0; i < 5; i++ {
		tx := common.NewTransaction(uint64(i), mockFromAddress, new(big.Int).SetUint64(uint64(i)), uint64(i), new(big.Int).SetUint64(uint64(i)), nil, mockFromAddress)
		txs = append(txs, tx)
	}

	txpool.AddTx(txs[0])
	txpool.AddTx(txs[1])
	exceptNonce := GetPoolNonce(mockFromAddress)
	assert.Equal(uint64(1), exceptNonce)

	txpool.AddTx(txs[2])
	exceptNonce = GetPoolNonce(mockFromAddress)
	assert.Equal(uint64(2), exceptNonce)
}

func TestNewTxPool(t *testing.T) {
	chain := &repository.Repository{}
	monkey.Patch(repository.NewLatestStateRepository, func() (*repository.Repository, error) {
		return chain, nil
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(chain), "GetNonce", func(*repository.Repository, types.Address) uint64 {
		return 0
	})
	var mockTxPoolConfig = TxPoolConfig{
		GlobalSlots:    4096,
		MaxTrsPerBlock: 512,
	}

	txpool := NewTxPool(mockTxPoolConfig, NewMockEvent())
	transactions := mock_transactions(4096)
	for index := 0; index < len(transactions); index++ {
		txpool.AddTx(transactions[index])
	}
	txs := txpool.(*TxPool)
	lens := txs.txBuffer.Len()
	assert.Equal(t, 4096, lens)

	mm := txpool.GetTxs()
	assert.Equal(t, 512, len(mm))
}
