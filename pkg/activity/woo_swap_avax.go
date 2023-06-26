package activity

import (
	"activity-bot/pkg/abi/wooRouterAvax"
	"activity-bot/pkg/constants"
	"activity-bot/pkg/random"
	"activity-bot/pkg/util"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"log"
	"math/big"
)

type WooSwapAvax struct {
	FromToken     common.Address
	ToToken       common.Address
	ValueSupplier *random.Supplier
	wooRouterAvax *wooRouterAvax.WooRouterAvax
	value         *big.Int // Computed on can execute
}

func NewWooSwapAvax(fromToken string, toToken string, valueSupplier *random.Supplier) *WooSwapAvax {
	return &WooSwapAvax{
		FromToken:     common.HexToAddress(fromToken),
		ToToken:       common.HexToAddress(toToken),
		ValueSupplier: valueSupplier,
	}
}

func (w *WooSwapAvax) CanExecute(ac ActivityContext) (bool, error) {
	log.Printf("Creating WooRouterAvax contract instance\n")
	contract, err := wooRouterAvax.NewWooRouterAvax(common.HexToAddress(constants.AVA_WOO_SWAP_CONTRACT), ac.Client)
	if err != nil {
		log.Fatal(err)
	}
	w.wooRouterAvax = contract

	log.Printf("Generating a random value to swap using value supplier [%s, %s]\n", w.ValueSupplier.Min().String(), w.ValueSupplier.Max().String())
	accountBalance, err := ac.Client.BalanceAt(ac.Context, ac.Account.Address, nil)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Error getting account balance [%s]: %v", ac.Account.Address.Hex(), err))
	}

	if accountBalance.Cmp(w.ValueSupplier.Min()) < 0 {
		return false, errors.New(fmt.Sprintf("Account [%s] has not enough balance to execute transfer", ac.Account.Address.Hex()))
	}

	w.value = w.ValueSupplier.Supply()
	// Make sure our value is not bigger than the account balance
	for {
		if w.value.Cmp(accountBalance) > 0 {
			w.value = w.ValueSupplier.Supply()
		} else {
			break
		}
	}

	return true, nil
}

func (w *WooSwapAvax) Execute(ac ActivityContext) (bool, error) {
	log.Printf("[%s] started swapping using WooSwapAvax\n", ac.Account.Address.Hex())

	result, err := w.wooRouterAvax.QuerySwap(&bind.CallOpts{}, w.FromToken, w.ToToken, w.value)
	if err != nil {
		return false, err
	}

	ac.Transactor.GasLimit = uint64(350000)
	ac.Transactor.Value = w.value
	tx, err := w.wooRouterAvax.Swap(ac.Transactor, w.FromToken, w.ToToken, w.value, result, ac.Account.Address, ac.Account.Address)
	if err != nil {
		return false, err
	}
	receipt, err := util.WaitForReceipt(tx, ac.Waiter)
	if err != nil {
		return false, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return false, errors.New(fmt.Sprintf("Transaction failed with status: %d", receipt.Status))
	}

	log.Printf("[%s] swapp of %s@%s wei to [%s] as %s@%s wei completed, transaction hash: %s\n",
		ac.Account.Address.Hex(),
		w.value.String(),
		w.FromToken,
		ac.Account.Address.Hex(),
		result.String(),
		w.ToToken,
		tx.Hash().Hex())
	return true, nil
}
