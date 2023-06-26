package cmd

import (
	"activity-bot/pkg/account"
	activities "activity-bot/pkg/activity"
	"activity-bot/pkg/random"
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"log"

	"github.com/spf13/cobra"
)

// transactionCmd represents the transaction command
var transactionCmd = &cobra.Command{
	Use:   "transaction",
	Short: "Does a transaction activity on local chain",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	rootCmd.AddCommand(transactionCmd)
}

func run() {
	cl, err := ethclient.Dial("HTTP://127.0.0.1:7545")
	defer cl.Close()

	if err != nil {
		panic(err)
	}
	chainId, err := cl.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	am := account.NewAccountManager("./keystore")
	am.UnlockAll("password")

	activity := activities.NewTransferNative(
		common.HexToAddress("0x3654114f003C108A339664f909131b4C07b0F779"),
		random.NewSupplier(params.GWei, 10000000, 100000000))

	account := am.Accounts()[0]
	transactor, err := am.NewTransactor(account, chainId)
	if err != nil {
		panic(err)
	}
	activityContext := activities.ActivityContext{
		Account:    &account,
		Client:     cl,
		Transactor: transactor,
		Context:    context.Background(),
	}

	r, err := activity.CanExecute(activityContext)
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Printf("Can execute: %v", r)

	r, err = activity.Execute(activityContext)
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Printf("Executed: %v", r)
}
