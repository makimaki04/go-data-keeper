package cli

import (
	"fmt"

	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/spf13/cobra"
)

var loginLogin string
var loginPassword string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginLogin == "" || loginPassword == "" {
			return fmt.Errorf("login and password are required")
		}

		app := cmd.Context().Value(appKey{}).(*clientapp.App)

		return app.Login(loginLogin, loginPassword)
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginLogin, "login", "", "User login")
	loginCmd.Flags().StringVar(&loginPassword, "password", "", "User password")
	_ = loginCmd.MarkFlagRequired("login")
	_ = loginCmd.MarkFlagRequired("password")
}
