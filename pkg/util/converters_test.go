package util

import (
	"math/big"
	"reflect"
	"testing"
)

func TestUsdcToMinimumUnit(t *testing.T) {
	type args struct {
		usdcAmount float64
	}
	tests := []struct {
		name    string
		args    args
		want    *big.Int
		wantErr bool
	}{
		{
			name: "0.2 USDC",
			args: args{
				usdcAmount: 0.2,
			},
			want:    big.NewInt(200000),
			wantErr: false,
		},
		{
			name: "1 USDC",
			args: args{
				usdcAmount: 1,
			},
			want:    big.NewInt(1000000),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UsdcToMinimumUnit(tt.args.usdcAmount)
			if (err != nil) != tt.wantErr {
				t.Errorf("UsdcToMinimumUnit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UsdcToMinimumUnit() got = %v, want %v", got, tt.want)
			}
		})
	}
}
