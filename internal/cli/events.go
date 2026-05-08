package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/drsoft-oss/proxymetrics/internal/store"
)

func eventsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "events", Short: "Inspect raw events"}
	cmd.AddCommand(eventsTailCmd())
	return cmd
}

func eventsTailCmd() *cobra.Command {
	var n int
	var profileID string
	c := &cobra.Command{
		Use:   "tail",
		Short: "Print the most recent events from DuckDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := openStoreAndRegistry(cmd)
			if err != nil {
				return err
			}
			defer s.Close()
			rows, err := s.TailEvents(context.Background(), store.TailOptions{N: n, ProfileID: profileID})
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "TS\tPROFILE\tHOST\tCODE\tCLASS\tBYTES_IN\tLATENCY_MS\tCOST")
			for _, e := range rows {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%d\t%d\t%.4f\n",
					e.TS.Format("2006-01-02 15:04:05"), e.ProfileID, e.TargetHost,
					e.StatusCode, e.StatusClass, e.BytesIn, e.LatencyMS, e.CostUSD)
			}
			return tw.Flush()
		},
	}
	c.Flags().IntVar(&n, "n", 50, "max rows")
	c.Flags().StringVar(&profileID, "profile", "", "filter by profile id")
	return c
}
