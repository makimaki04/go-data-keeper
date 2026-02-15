package cli

import (
	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync user changes since last rev",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value(appKey{}).(*clientapp.App)
		return app.SyncChanges()
	},
}

func init() {

}
