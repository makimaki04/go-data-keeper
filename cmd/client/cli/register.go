package cli

import (
	"fmt"

	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/spf13/cobra"
)

var regLogin, regPassword string

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if regLogin == "" || regPassword == "" {
			return fmt.Errorf("login and password are required")
		}

		app := cmd.Context().Value(appKey{}).(*clientapp.App)

		return app.Register(regLogin, regPassword)
	},
}

func init() {
	registerCmd.Flags().StringVar(&regLogin, "login", "", "User login")
	registerCmd.Flags().StringVar(&regPassword, "password", "", "User password")
	_ = registerCmd.MarkFlagRequired("login")
	_ = registerCmd.MarkFlagRequired("password")
}
