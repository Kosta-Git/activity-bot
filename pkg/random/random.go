package random

import (
	"crypto/rand"
	"math/big"
)

type Supplier struct {
	unit *big.Int
	min  *big.Int
	max  *big.Int
}

func NewSupplier(unit, min, max int64) *Supplier {
	u := big.NewInt(unit)
	return &Supplier{
		unit: u,
		min:  new(big.Int).Mul(big.NewInt(min), u),
		max:  new(big.Int).Mul(big.NewInt(max), u),
	}
}

func (s *Supplier) Min() *big.Int {
	return s.min
}

func (s *Supplier) Max() *big.Int {
	return s.max
}

func (s *Supplier) Supply() *big.Int {
	diff := new(big.Int).Sub(s.max, s.min)
	r, err := rand.Int(rand.Reader, diff)
	if err != nil {
		panic(err)
	}
	return new(big.Int).Add(r, s.min)
}
