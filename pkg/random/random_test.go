package random

import (
	"github.com/ethereum/go-ethereum/params"
	"testing"
)

func TestSupply(t *testing.T) {
	s := NewSupplier(params.Ether, 100, 200)
	for i := 0; i < 10; i++ {
		amount := s.Supply()
		if amount.Cmp(s.min) == -1 || amount.Cmp(s.max) == 1 {
			t.Errorf("Generated amount %v out of range [%v, %v]", amount, s.min, s.max)
		}
	}
}
