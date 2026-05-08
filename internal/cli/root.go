package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// Build-time variables, set via -ldflags in the Makefile.
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

// Root returns the configured root command. Each subcommand is registered here.
func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "proxymetrics",
		Short:         "Self-hosted MITM proxy router with per-vendor cost and error attribution",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringP("config", "c", configPathDefault(), "path to config.yaml")

	cmd.AddCommand(versionCmd())
	cmd.AddCommand(configCmd())
	cmd.AddCommand(cacertCmd())
	cmd.AddCommand(profileCmd())
	cmd.AddCommand(eventsCmd())
	cmd.AddCommand(serveCmd())
	cmd.AddCommand(dbCmd())

	return cmd
}

// configPathDefault honours $PROXYMETRICS_CONFIG, then falls back to ./config.yaml.
func configPathDefault() string {
	if v := os.Getenv("PROXYMETRICS_CONFIG"); v != "" {
		return v
	}
	return "./config.yaml"
}
