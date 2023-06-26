package activity

import (
	"activity-bot/pkg/abi/bitcoinBridgeAvax"
	"activity-bot/pkg/abi/wrappedBitcoinAvax"
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

type BitcoinBridgeAvax struct {
	FromChainId        uint16
	ToChainId          uint16
	bitcoinBridgeAvax  *bitcoinBridgeAvax.BitcoinBridgeAvax
	wrappedBitcoinAvax *wrappedBitcoinAvax.WrappedBitcoinAvax
	ValueSupplier      *random.Supplier
	value              *big.Int // Computed on can execute
}

func NewBitcoinBridgeAvax(fromChainId uint16, toChainId uint16, valueSupplier *random.Supplier) *BitcoinBridgeAvax {
	return &BitcoinBridgeAvax{
		FromChainId:   fromChainId,
		ToChainId:     toChainId,
		ValueSupplier: valueSupplier,
	}
}

func (b *BitcoinBridgeAvax) CanExecute(ac ActivityContext) (bool, error) {
	log.Printf("Creating BitcoinBridgeAvax contract instance\n")
	bitcoinBridgeContract, err := bitcoinBridgeAvax.NewBitcoinBridgeAvax(common.HexToAddress(constants.AVA_BITCOIN_BRIDGE_CONTRACT), ac.Client)
	if err != nil {
		return false, err
	}
	b.bitcoinBridgeAvax = bitcoinBridgeContract

	log.Printf("Creating Btc.B contract instance\n")
	wrappedBitcoinContract, err := wrappedBitcoinAvax.NewWrappedBitcoinAvax(common.HexToAddress(constants.AVA_BTCB), ac.Client)
	if err != nil {
		return false, err
	}
	b.wrappedBitcoinAvax = wrappedBitcoinContract

	log.Printf("Generating a random value to bridge using value supplier [%s, %s]\n", b.ValueSupplier.Min().String(), b.ValueSupplier.Max().String())
	balance, err := b.wrappedBitcoinAvax.BalanceOf(&bind.CallOpts{}, ac.Account.Address)
	if err != nil {
		return false, err
	}
	if balance.Cmp(b.ValueSupplier.Min()) < 0 {
		return false, errors.New(fmt.Sprintf("Account [%s] has not enough AVA balance to execute transfer", ac.Account.Address.Hex()))
	}

	b.value = b.ValueSupplier.Supply()
	// Make sure our value is not bigger than the account balance
	for {
		if b.value.Cmp(balance) > 0 {
			b.value = b.ValueSupplier.Supply()
		} else {
			break
		}
	}

	return true, nil
}

func (b *BitcoinBridgeAvax) Execute(ac ActivityContext) (bool, error) {
	log.Printf("[%s] started cross swapping using BitcoinBridgeAvax\n", ac.Account.Address.Hex())

	log.Println("Checking if Btc.B allowance required")
	allowance, err := b.wrappedBitcoinAvax.Allowance(&bind.CallOpts{}, common.HexToAddress(constants.AVA_BITCOIN_BRIDGE_CONTRACT), ac.Account.Address)
	if err != nil {
		return false, err
	}
	if allowance.Cmp(b.value) < 0 {
		ac.Transactor.Value = big.NewInt(0)
		tx, err := b.wrappedBitcoinAvax.Approve(ac.Transactor, common.HexToAddress(constants.AVA_BITCOIN_BRIDGE_CONTRACT), big.NewInt(0).Sub(b.value, allowance))
		if err != nil {
			return false, err
		}
		log.Printf("Approve tx sent: %s", tx.Hash().Hex())
		receipt, err := util.WaitForReceipt(tx, ac.Waiter)
		if err != nil {
			return false, err
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return false, errors.New(fmt.Sprintf("Approve tx failed: %s", receipt.TxHash.Hex()))
		}
	}

	// Quote LZ Fees
	fees, err := b.bitcoinBridgeAvax.QuoteOFTFee(&bind.CallOpts{}, b.ToChainId, b.value)
	if err != nil {
		return false, err
	}
	log.Printf("[%s] OFT Quote Fees: %v\n", ac.Account.Address, fees)

	// Bridge
	var addressBytes []byte
	addressBytes = append(make([]byte, 12), ac.Account.Address.Bytes()...)
	var addressArr [32]byte
	copy(addressArr[:], addressBytes)
	staticParams := common.Hex2Bytes("0002000000000000000000000000000000000000000000000000000000000003d0900000000000000000000000000000000000000000000000000000000000000000")
	params := append(staticParams, ac.Account.Address.Bytes()...)
	ac.Transactor.Value = fees
	ac.Transactor.GasLimit = 300000
	tx, err := b.bitcoinBridgeAvax.SendFrom(
		ac.Transactor,
		ac.Account.Address,
		b.FromChainId,
		addressArr,
		b.value,
		b.value,
		bitcoinBridgeAvax.ICommonOFTLzCallParams{
			RefundAddress:     ac.Account.Address,
			ZroPaymentAddress: common.HexToAddress("0x"),
			AdapterParams:     params,
		})
	if err != nil {
		return false, err
	}
	receipt, err := util.WaitForReceipt(tx, ac.Waiter)
	if err != nil {
		return false, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return false, errors.New(fmt.Sprintf("Approve tx failed: %s", receipt.TxHash.Hex()))
	}
	return true, nil
}
