package cli

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/spf13/cobra"
)

var itemID string

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete  item using item_id",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value(appKey{}).(*clientapp.App)

		id, err := uuid.Parse(itemID)
		if err != nil {
			return fmt.Errorf("wrong item_id format: %v", err)
		}

		return app.DeleteItem(id)
	},
}

func init() {
	deleteCmd.Flags().StringVar(&itemID, "id", "", "item_id for delete")
	_ = deleteCmd.MarkFlagRequired("id")
}
