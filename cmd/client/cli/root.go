/*
Copyright © 2026 Dima <goodd214080@gmail.com>
*/
// Package cli defines the client command-line interface.
package cli

import (
	"context"
	"os"
	"path/filepath"

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

func defaultLocalPaths() (state string, vault string) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, herr := os.UserHomeDir()
		if herr != nil || home == "" {
			dir = "."
		} else {
			dir = home
		}
	}

	base := filepath.Join(dir, "gophkeeper")
	return filepath.Join(base, "state.json"), filepath.Join(base, "vault.json")
}

// Execute runs the root command.
// Execute exits the process with status 1 if command execution fails.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(versionCmd)

	defState, defVault := defaultLocalPaths()
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://127.0.0.1:8080", "Server URL")
	rootCmd.PersistentFlags().StringVar(&statePath, "state", defState, "State file path")
	rootCmd.PersistentFlags().StringVar(&vaultPath, "vault", defVault, "Vault file path")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
