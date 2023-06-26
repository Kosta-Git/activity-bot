package activity

import (
	"activity-bot/pkg/random"
	"activity-bot/pkg/util"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"log"
	"math/big"
)

const gasLimit uint64 = 21000

type TransferNative struct {
	to            common.Address
	valueSupplier *random.Supplier
	// Computed on can execute
	value *big.Int
}

func NewTransferNative(to common.Address, valueSupplier *random.Supplier) *TransferNative {
	return &TransferNative{
		to:            to,
		valueSupplier: valueSupplier,
	}
}

func (t *TransferNative) CanExecute(ac ActivityContext) (bool, error) {
	accountBalance, err := ac.Client.BalanceAt(ac.Context, ac.Account.Address, nil)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error getting account balance [%s]: %v", ac.Account.Address.Hex(), err))
	}

	if accountBalance.Cmp(t.valueSupplier.Min()) < 0 {
		return false, errors.New(fmt.Sprintf("Account [%s] has not enough balance to execute transfer", ac.Account.Address.Hex()))
	}

	t.value = t.valueSupplier.Supply()
	// Make sure our value is not bigger than the account balance
	for {
		if t.value.Cmp(accountBalance) > 0 {
			t.value = t.valueSupplier.Supply()
		} else {
			break
		}
	}

	return true, nil
}

func (t *TransferNative) Execute(ac ActivityContext) (bool, error) {
	log.Printf("[%s] started transfering %s wei to [%s]\n", ac.Account.Address.Hex(), t.value.String(), t.to)

	chainId, err := ac.Client.ChainID(ac.Context)
	if err != nil {
		return false, err
	}
	nonce, err := ac.Client.PendingNonceAt(ac.Context, ac.Account.Address)
	if err != nil {
		return false, err
	}
	feeCap, err := ac.Client.SuggestGasPrice(ac.Context)
	if err != nil {
		return false, err
	}
	tipCap, err := ac.Client.SuggestGasTipCap(ac.Context)
	if err != nil {
		return false, err
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &t.to,
		Value:     t.value,
		Data:      nil,
	})

	signedTx, err := ac.Transactor.Signer(ac.Transactor.From, tx)
	if err != nil {
		return false, err
	}

	err = ac.Client.SendTransaction(ac.Context, signedTx)
	if err != nil {
		return false, err
	}

	receipt, err := util.WaitForReceipt(signedTx, ac.Waiter)
	if err != nil {
		return false, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return false, errors.New(fmt.Sprintf("Transaction failed with status: %d", receipt.Status))
	}

	log.Printf("[%s] transfer of %s wei to [%s] completed, transaction hash: %s\n",
		ac.Account.Address.Hex(),
		t.value.String(),
		t.to,
		signedTx.Hash().Hex())
	return true, nil
}
