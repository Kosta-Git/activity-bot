package account

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"log"
	"math/big"
)

type AccountManager struct {
	keystore *keystore.KeyStore
	manager  *accounts.Manager
}

func NewAccountManager(keystorePath string) *AccountManager {
	ks := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
	return &AccountManager{
		keystore: ks,
		manager:  accounts.NewManager(&accounts.Config{InsecureUnlockAllowed: false}, ks),
	}
}

func (am *AccountManager) Accounts() []accounts.Account {
	return am.keystore.Accounts()
}

func (am *AccountManager) CreateAccount(password string) (accounts.Account, error) {
	return am.keystore.NewAccount(password)
}

func (am *AccountManager) UnlockAll(password string) {
	for _, account := range am.keystore.Accounts() {
		if err := am.keystore.Unlock(account, password); err != nil {
			log.Fatalf("Unable to unlock account: %s\n", account.Address)
		}
	}
}

func (am *AccountManager) NewTransactor(account accounts.Account, chainId *big.Int) (*bind.TransactOpts, error) {
	if !am.keystore.HasAddress(account.Address) {
		return nil, fmt.Errorf("account not found in keystore: %s", account.Address.Hex())
	}
	return bind.NewKeyStoreTransactorWithChainID(am.keystore, account, chainId)
}
