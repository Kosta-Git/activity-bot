package activity

import (
	"activity-bot/pkg/abi/stargateFinanceFTM"
	"activity-bot/pkg/abi/usdcFTM"
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

type StargateSwapFTM struct {
	FromPool           *big.Int
	ToPool             *big.Int
	ToChainId          uint16
	stargateFinanceFTM *stargateFinanceFTM.StargateFinanceFTM
	usdcFTM            *usdcFTM.UsdcFTM
	ValueSupplier      *random.Supplier
	value              *big.Int // Computed on can execute
}

func NewStargateSwapFTM(fromPool *big.Int, toPool *big.Int, toChainId uint16, valueSupplier *random.Supplier) *StargateSwapFTM {
	return &StargateSwapFTM{
		FromPool:      fromPool,
		ToPool:        toPool,
		ToChainId:     toChainId,
		ValueSupplier: valueSupplier,
	}
}

func (s *StargateSwapFTM) CanExecute(ac ActivityContext) (bool, error) {
	log.Printf("Creating StargateFinanceFTM contract instance\n")
	sgFinanceAvax, err := stargateFinanceFTM.NewStargateFinanceFTM(common.HexToAddress(constants.FTM_STARGATE_CONTRACT), ac.Client)
	if err != nil {
		return false, err
	}
	s.stargateFinanceFTM = sgFinanceAvax

	log.Printf("Creating UsdcFTM contract instance\n")
	usdAvaContract, err := usdcFTM.NewUsdcFTM(common.HexToAddress(constants.FTM_USDC_CONTRACT), ac.Client)
	if err != nil {
		return false, err
	}
	s.usdcFTM = usdAvaContract

	log.Printf("Generating a random value to swap using value supplier [%s, %s]\n", s.ValueSupplier.Min().String(), s.ValueSupplier.Max().String())
	balance, err := s.usdcFTM.BalanceOf(&bind.CallOpts{}, ac.Account.Address)
	if err != nil {
		return false, err
	}
	if balance.Cmp(s.ValueSupplier.Min()) < 0 {
		return false, errors.New(fmt.Sprintf("Account [%s] has not enough USDC balance to execute transfer", ac.Account.Address.Hex()))
	}

	s.value = s.ValueSupplier.Supply()
	// Make sure our value is not bigger than the account balance
	for {
		if s.value.Cmp(balance) > 0 {
			s.value = s.ValueSupplier.Supply()
		} else {
			break
		}
	}

	return true, nil
}

func (s *StargateSwapFTM) Execute(ac ActivityContext) (bool, error) {
	log.Printf("[%s] started cross swapping using StargateFinanceFTM\n", ac.Account.Address.Hex())

	// Quote LZ Fees
	fees, _, err := s.stargateFinanceFTM.QuoteLayerZeroFee(
		&bind.CallOpts{},
		s.ToChainId,
		1,
		ac.Account.Address.Bytes(),
		common.Hex2Bytes("0x"),
		stargateFinanceFTM.IStargateRouterlzTxObj{
			DstGasForCall:   big.NewInt(0),
			DstNativeAmount: big.NewInt(0),
			DstNativeAddr:   common.Hex2Bytes("0x"),
		})
	if err != nil {
		return false, err
	}
	log.Printf("[%s] LZ Quote Fees: %v\n", ac.Account.Address, fees)

	// USDC allowance
	log.Println("Checking if USDC allowance required")
	allowance, err := s.usdcFTM.Allowance(&bind.CallOpts{}, common.HexToAddress(constants.FTM_STARGATE_CONTRACT), ac.Account.Address)
	if err != nil {
		return false, err
	}
	if allowance.Cmp(s.value) < 0 {
		ac.Transactor.Value = big.NewInt(0)
		tx, err := s.usdcFTM.Approve(ac.Transactor, common.HexToAddress(constants.FTM_STARGATE_CONTRACT), big.NewInt(0).Sub(s.value, allowance))
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

	// Bridge
	amountAsFloat := big.NewFloat(0).SetInt(s.value)
	slippage := big.NewFloat(0.99)
	minAmount, _ := amountAsFloat.Mul(amountAsFloat, slippage).Int(nil)

	ac.Transactor.Value = fees
	ac.Transactor.GasLimit = 600000
	tx, err := s.stargateFinanceFTM.Swap(
		ac.Transactor,
		s.ToChainId,
		s.FromPool, // https://stargateprotocol.gitbook.io/stargate/developers/pool-ids
		s.ToPool,   // https://stargateprotocol.gitbook.io/stargate/developers/pool-ids
		ac.Account.Address,
		s.value,
		minAmount,
		stargateFinanceFTM.IStargateRouterlzTxObj{
			DstGasForCall:   big.NewInt(0),
			DstNativeAmount: big.NewInt(0),
			DstNativeAddr:   common.Hex2Bytes("0x"),
		},
		ac.Account.Address.Bytes(),
		make([]byte, 0),
	)
	ac.Transactor.Value = big.NewInt(0)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("StargateFinance Cross swap tx sent: %s", tx.Hash().Hex())

	receipt, err := util.WaitForReceipt(tx, ac.Waiter)
	if receipt.Status != types.ReceiptStatusSuccessful {
		return false, errors.New(fmt.Sprintf("StargateFinanceFTM Cross swap tx failed: %s", receipt.TxHash.Hex()))
	}
	log.Printf("StargateFinance Cross swap tx confirmed: %s", receipt.TxHash.Hex())
	return true, nil
}
