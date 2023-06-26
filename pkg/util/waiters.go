package util

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"sync"
	"time"
)

type transactionWaiter struct {
	TxHash   common.Hash
	Listener chan<- *types.Receipt
}

type blockWaiter struct {
	BlockToWait uint64
	Listener    chan<- interface{}
}

type Waiter struct {
	client               *ethclient.Client
	transactionWaiters   []transactionWaiter
	blockWaiters         []blockWaiter
	lastBlockId          uint64
	lastBlockIdUpdatedAt time.Time
	supportsSubscribing  bool
	pollTimeDuration     time.Duration
	lock                 *sync.Mutex
}

func NewWaiter(client *ethclient.Client, pollTimeDuration time.Duration, supportsSubscribing bool) *Waiter {
	return &Waiter{
		client:               client,
		transactionWaiters:   make([]transactionWaiter, 0),
		blockWaiters:         make([]blockWaiter, 0),
		lastBlockId:          0,
		lastBlockIdUpdatedAt: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		lock:                 &sync.Mutex{},
		supportsSubscribing:  supportsSubscribing,
		pollTimeDuration:     pollTimeDuration,
	}
}

func (w *Waiter) WaitForTransaction(txHash common.Hash) (<-chan *types.Receipt, error) {
	// Make sure to lock the waiter before accessing the transactionWaiters
	w.lock.Lock()
	defer w.lock.Unlock()
	listener := make(chan *types.Receipt)
	w.transactionWaiters = append(w.transactionWaiters, transactionWaiter{
		TxHash:   txHash,
		Listener: listener,
	})

	return listener, nil
}

func (w *Waiter) WaitForBlocks(blocksToWait uint64) (<-chan interface{}, error) {
	// Make sure to lock the waiter before accessing the transactionWaiters
	w.lock.Lock()
	defer w.lock.Unlock()
	listener := make(chan interface{})
	err := w.updateBlockId()
	if err != nil {
		return nil, err
	}

	w.blockWaiters = append(w.blockWaiters, blockWaiter{
		BlockToWait: w.lastBlockId + blocksToWait,
		Listener:    listener,
	})

	return listener, nil
}

func (w *Waiter) Start() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	if w.supportsSubscribing {
		go func() {
			err := w.eventStartListening(ctx.Done())
			if err != nil {
				log.Fatal(err)
			}
		}()
	} else {
		go w.pollingStartListening(ctx.Done())
	}
	return cancel
}

func (w *Waiter) pollingStartListening(done <-chan struct{}) {
	for {
		select {
		case <-done:
			log.Println("Stop listening for new blocks")
			break
		default:
			time.Sleep(w.pollTimeDuration)
			w.lock.Lock()
			// If there are no more waiters, stop listening
			if len(w.transactionWaiters) == 0 && len(w.blockWaiters) == 0 {
				log.Println("No more waiters, unsubscribing from new block headers")
				w.lock.Unlock()
				continue
			}

			if w.hasTxWaiters() {
				w.tryNotifyTxReceipt()
			}

			if w.hasBlockWaiters() {
				err := w.updateBlockId()
				if err != nil {
					log.Printf("Failed to update block id: %v", err)
					w.lock.Unlock()
					continue
				}
				w.notifyBlockWaiters()
			}
			w.lock.Unlock()
		}
	}
}

func (w *Waiter) eventStartListening(done <-chan struct{}) error {
	w.lock.Lock()
	headers := make(chan *types.Header)
	sub, err := w.client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatalf("Failed to subscribe to new block headers: %v\n", err)
		return err
	}
	w.lock.Unlock()

	select {

	case <-done:
		log.Println("Stop listening for new blocks")
		sub.Unsubscribe()
		break
	default:
		w.lock.Lock()

		if len(w.transactionWaiters) == 0 && len(w.blockWaiters) == 0 {
			log.Println("No more waiters, unsubscribing from new block headers")
			sub.Unsubscribe()
			w.lock.Unlock()
			return nil
		}

		select {
		case err := <-sub.Err():
			w.lock.Unlock()
			log.Fatalf("Subscription error: %v", err)
		case header := <-headers:
			w.lastBlockId = header.Number.Uint64()

			if len(w.blockWaiters) > 0 {
				w.notifyBlockWaiters()
			}

			if len(w.transactionWaiters) > 0 {
				block, err := w.client.BlockByHash(context.Background(), header.Hash())
				log.Printf("Failed to get block details for block %v: %v", header.Number, err)
				w.notifyTxWaiters(block)
			}
		}

		w.lock.Unlock()
	}
	return nil
}

func (w *Waiter) notifyBlockWaiters() {
	updatedWaiters := make([]blockWaiter, 0)
	for _, waiter := range w.blockWaiters {
		if waiter.BlockToWait <= w.lastBlockId {
			waiter.Listener <- nil
		} else {
			updatedWaiters = append(updatedWaiters, waiter)
		}
	}
	w.blockWaiters = updatedWaiters
}

func (w *Waiter) tryNotifyTxReceipt() {
	updatedWaiters := make([]transactionWaiter, 0)
	for _, waiter := range w.transactionWaiters {
		receipt, err := w.client.TransactionReceipt(context.Background(), waiter.TxHash)
		if err != nil {
			log.Printf("Failed to get receipt for transaction %v: %v", waiter.TxHash, err)
			continue
		}

		if receipt == nil {
			continue
		}

		w.transactionWaiters = updatedWaiters
		waiter.Listener <- receipt
	}
}

func (w *Waiter) notifyTxWaiters(block *types.Block) {
	updatedWaiters := make([]transactionWaiter, 0)
	for _, waiter := range w.transactionWaiters {
		tx := block.Transaction(waiter.TxHash)
		if tx == nil {
			updatedWaiters = append(updatedWaiters, waiter)
			continue
		}
		receipt, err := w.client.TransactionReceipt(context.Background(), waiter.TxHash)
		if err != nil {
			log.Printf("Failed to get receipt for transaction %v: %v", waiter.TxHash, err)
			continue
		}
		w.transactionWaiters = updatedWaiters
		waiter.Listener <- receipt
	}
}

func (w *Waiter) updateBlockId() error {
	if w.lastBlockIdUpdatedAt.Before(time.Now().Add(-1 * time.Second)) {
		block, err := w.client.BlockByNumber(context.Background(), nil)
		if err != nil {
			log.Printf("Failed to get latest block: %v\n", err)
			return err
		}
		w.lastBlockId = block.NumberU64()
		w.lastBlockIdUpdatedAt = time.Now()
		return nil
	}
	return nil
}

func (w *Waiter) hasTxWaiters() bool {
	return len(w.transactionWaiters) > 0
}

func (w *Waiter) hasBlockWaiters() bool {
	return len(w.blockWaiters) > 0
}
