package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/clientapp"
	"github.com/spf13/cobra"
)

var getPassword, getItemID, out string

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get item using item_id",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value(appKey{}).(*clientapp.App)

		id, err := uuid.Parse(getItemID)
		if err != nil {
			return fmt.Errorf("couldn't parse uuid: %v", err)
		}

		env, err := app.GetItem(getPassword, id)
		if err != nil {
			return err
		}

		if env.Type == "binary" && out == "" {
			return fmt.Errorf("binary type requires --out")
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		defer w.Flush()

		hasMeta := len(env.Meta) > 0
		hasMetaText := env.MetaText != ""
		fmt.Fprintln(w, "TYPE\tID\tHAS_META\tHAS_META_TEXT")
		fmt.Fprintf(w, "%s\t%s\t%v\t%v\n", env.Type, id.String(), hasMeta, hasMetaText)
		fmt.Fprintln(w)

		if env.MetaText != "" {
			fmt.Fprintln(w, "META_TEXT")
			fmt.Fprintln(w, env.MetaText)
			fmt.Fprintln(w)
		}

		if len(env.Meta) > 0 {
			fmt.Fprintln(w, "META")
			fmt.Fprintln(w, "KEY\tVALUE")

			keys := make([]string, 0, len(env.Meta))
			for k := range env.Meta {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Fprintf(w, "%s\t%s\n", k, env.Meta[k])
			}
			fmt.Fprintln(w)
		}

		if env.Type == "binary" {
			if err := os.WriteFile(out, env.Data, 0600); err != nil {
				return err
			}

			fmt.Fprintln(w, "OUT_PATH")
			fmt.Fprintln(w, out)
			return nil
		}

		fmt.Fprintln(w, "DATA")
		fmt.Fprintln(w, string(env.Data))
		return nil
	},
}

func init() {
	getCmd.Flags().StringVar(&getPassword, "password", "", "master password")
	getCmd.Flags().StringVar(&getItemID, "id", "", "item_id")
	getCmd.Flags().StringVar(&out, "out", "", "out file for binary data")
	_ = getCmd.MarkFlagRequired("password")
	_ = getCmd.MarkFlagRequired("id")
}
