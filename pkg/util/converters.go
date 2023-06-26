package util

import (
	"math/big"
)

func UsdcToMinimumUnit(usdcAmount float64) (*big.Int, error) {
	// Convert the USDC value to a big.Float
	amount := big.NewFloat(usdcAmount)

	// Divide the minimum unit by 10^6 to adjust for the USDC decimals
	amount = new(big.Float).Mul(amount, big.NewFloat(1e6))

	// Convert the result to a big.Int and return it
	result, _ := amount.Int(nil)
	return result, nil
}
