/*
Copyright © 2026 Dima <goodd214080@gmail.com>
*/
package cli

import (
	"context"
	"os"

	"github.com/makimaki04/go-data-keeper.git/internal/clientapi"
	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/makimaki04/go-data-keeper.git/internal/clientstate"
	"github.com/makimaki04/go-data-keeper.git/internal/logger"
	"github.com/spf13/cobra"
)

var (
	loggerCfgPath = "configs/logger.json"
	serverURL     string
	statePath     string
	vaultPath     string
)

type appKey struct{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "go-data-keeper is a securely store system",
	Long: `go-data-keeper is a client-server system 
	that allows users to securely store 
	usernames, passwords, binary data, and other private information.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logger, err := logger.NewLogger(loggerCfgPath)
		if err != nil {
			return err
		}

		store, err := clientstate.NewStore(statePath, vaultPath, logger)
		if err != nil {
			return err
		}

		api := clientapi.NewHTTPClient(serverURL, logger)

		app := clientapp.NewApp(api, *store, logger)

		state, err := store.LoadState()
		if err != nil {
			return err
		}

		api.SetToken(state.JWTToken)

		cmd.SetContext(context.WithValue(cmd.Context(), appKey{}, app))

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.client.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)

	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://127.0.0.1:8080", "Server URL")
	rootCmd.PersistentFlags().StringVar(&statePath, "state", "D:\\prog\\data\\gophkeeper\\state.json", "State file path")
	rootCmd.PersistentFlags().StringVar(&vaultPath, "vault", "D:\\prog\\data\\gophkeeper\\vault.json", "Vault file path")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
