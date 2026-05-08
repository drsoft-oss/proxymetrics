package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, build date",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("proxymetrics %s (%s, built %s)\n", Version, GitCommit, BuildDate)
			return nil
		},
	}
}
