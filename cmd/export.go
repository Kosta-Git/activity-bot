package cmd

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var (
	folder   string
	password string
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Exports all private keys",
	Run: func(cmd *cobra.Command, args []string) {
		exportKeys()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&folder, "folder", "folder", "The folder from which to export the private keys")
	rootCmd.PersistentFlags().StringVar(&password, "password", "password", "The password to use to decrypt the private keys")

	rootCmd.AddCommand(exportCmd)
}

func exportKeys() {
	entries, err := os.ReadDir(folder)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(fmt.Sprintf("%s%s%s", folder, string(os.PathSeparator), e.Name()))
		if err != nil {
			log.Printf("Error reading file %s: %v\n", e.Name(), err)
		}
		key, err := keystore.DecryptKey(data, password)
		if err != nil {
			fmt.Printf("Error decrypting key: %v\n", err)
			return
		}
		fmt.Printf("Private key: %v\n", hexutil.Encode(key.PrivateKey.D.Bytes()))
	}
}
