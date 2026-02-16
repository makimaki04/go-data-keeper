package cli

import (
	"fmt"
	"slices"
	"text/tabwriter"

	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/spf13/cobra"
)

var all, deleted bool
var listItemType string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Get items list",
	RunE: func(cmd *cobra.Command, args []string) error {
		if listItemType != "" && !slices.Contains(types, listItemType) {
			return fmt.Errorf("unsupported type, allowed: %v", types)
		}

		app := cmd.Context().Value(appKey{}).(*clientapp.App)

		items, err := app.GetList(all, listItemType, deleted)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		defer w.Flush()

		fmt.Fprintln(w, "ID\tTYPE\tDELETED\tUPDATED_REV")
		for _, it := range items {
			fmt.Fprintf(w, "%s\t%s\t%v\t%d\n", it.ID, it.Type, it.Deleted, it.UpdatedRev)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&all, "all", false, "returns all items, included deleted")
	listCmd.Flags().BoolVar(&deleted, "deleted", false, "returns all deleted items")
	listCmd.Flags().StringVar(&listItemType, "type", "", "returns items of the required type: login/pass, text, binary, card")
}
