package cli

import (
	"fmt"

	"github.com/makimaki04/go-data-keeper.git/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "get build version, date and commit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Build version: %s\n", buildinfo.BuildVersion)
		fmt.Printf("Build date: %s\n", buildinfo.BuildDate)
		fmt.Printf("Build commit: %s\n", buildinfo.BuildCommit)
	},
}

func init() {}
