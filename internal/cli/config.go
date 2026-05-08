package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anonymous-proxies/proxymetrics/internal/config"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Inspect or validate configuration"}
	cmd.AddCommand(configValidateCmd())
	return cmd
}

func configValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Parse and validate the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("config")
			if _, err := config.Load(path); err != nil {
				return err
			}
			fmt.Printf("OK: %s\n", path)
			return nil
		},
	}
}
