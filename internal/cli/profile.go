package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/drsoft-oss/proxymetrics/internal/ipcheck"
	"github.com/drsoft-oss/proxymetrics/internal/profile"
	"github.com/drsoft-oss/proxymetrics/internal/store/duckdb"
)

func profileCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Inspect observed upstream profiles"}
	cmd.AddCommand(profileListCmd(), profileTestCmd())
	return cmd
}

// openStoreAndRegistry is shared by every profile/events subcommand that needs the DB.
func openStoreAndRegistry(cmd *cobra.Command) (*duckdb.Store, *profile.Registry, error) {
	cfg, err := loadCfg(cmd)
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(cfg.Storage.DataDir, 0o755); err != nil {
		return nil, nil, err
	}
	s, err := duckdb.Open(cfg.Storage.DataDir + "/events.duckdb")
	if err != nil {
		return nil, nil, err
	}
	reg, err := profile.NewRegistry(context.Background(), s)
	if err != nil {
		s.Close()
		return nil, nil, err
	}
	return s, reg, nil
}

func mask(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-3:]
}

func profileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles in a tabular format",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, reg, err := openStoreAndRegistry(cmd)
			if err != nil {
				return err
			}
			defer s.Close()
			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tLABEL\tVENDOR\tTYPE\tREGION\tUPSTREAM\tPRICE/GB")
			for _, p := range reg.All() {
				price := ""
				if p.PricePerGB != nil {
					price = fmt.Sprintf("%.4f %s", *p.PricePerGB, p.Currency)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					p.ID, p.Label, p.Vendor, p.Type, p.Region, mask(p.UpstreamURL), price)
			}
			return tw.Flush()
		},
	}
}

func profileTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <id>",
		Short: "Send one request through the profile to a rotating IP-check API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg(cmd)
			if err != nil {
				return err
			}
			s, reg, err := openStoreAndRegistry(cmd)
			if err != nil {
				return err
			}
			defer s.Close()
			p, ok := reg.Lookup(args[0])
			if !ok {
				return fmt.Errorf("profile %q not found", args[0])
			}

			builder := profile.DefaultProxyClientBuilder{Timeout: 15 * time.Second}
			client, err := builder.ClientFor(p)
			if err != nil {
				return err
			}
			ipc := ipcheck.New(cfg.IPCheck.APIs, client)

			start := time.Now()
			res, err := ipc.Check(context.Background())
			latency := time.Since(start)
			if err != nil {
				fmt.Printf("FAIL after %v: %v\n", latency, err)
				return err
			}
			fmt.Printf("OK in %v\n  exit IP: %s\n  via: %s\n", latency, res.IP, res.APIUsed)
			return nil
		},
	}
}
