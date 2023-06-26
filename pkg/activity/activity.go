package activity

import (
	"activity-bot/pkg/util"
	"context"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ActivityContext struct {
	Account    *accounts.Account
	Client     *ethclient.Client
	Transactor *bind.TransactOpts
	Context    context.Context
	Waiter     *util.Waiter
}

type Activity interface {
	CanExecute(activityContext ActivityContext) (bool, error)
	Execute(activityContext ActivityContext) (bool, error)
}
