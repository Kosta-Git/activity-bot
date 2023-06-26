package util

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
)

func WaitForReceipt(tx *types.Transaction, awaiter *Waiter) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return WaitForReceiptOrTimeout(tx, awaiter, ctx)
}

func WaitForReceiptOrTimeout(tx *types.Transaction, awaiter *Waiter, ctx context.Context) (*types.Receipt, error) {
	receiptCh, err := awaiter.WaitForTransaction(tx.Hash())
	if err != nil {
		return nil, err
	}
	select {
	case receipt := <-receiptCh:
		return receipt, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
