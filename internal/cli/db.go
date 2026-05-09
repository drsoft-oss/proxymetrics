package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/drsoft-oss/proxymetrics/internal/store/sqlite"
)

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "db", Short: "Inspect and maintain the embedded database"}
	cmd.AddCommand(dbSchemaCmd(), dbStatsCmd(), dbVacuumCmd(), dbBackupCmd(), dbRestoreCmd())
	return cmd
}

func dbSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print column names + types per table",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := openStoreAndRegistry(cmd)
			if err != nil {
				return err
			}
			defer s.Close()

			// SQLite has no information_schema; walk sqlite_master for tables and
			// pragma_table_info for columns.
			tableRows, err := s.QueryRollupSource(context.Background(), `
				SELECT name FROM sqlite_master
				WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
				ORDER BY name`)
			if err != nil {
				return err
			}
			var tables []string
			for tableRows.Next() {
				var t string
				if err := tableRows.Scan(&t); err != nil {
					tableRows.Close()
					return err
				}
				tables = append(tables, t)
			}
			tableRows.Close()
			if err := tableRows.Err(); err != nil {
				return err
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "TABLE\tCOLUMN\tTYPE")
			for _, table := range tables {
				colRows, err := s.QueryRollupSource(context.Background(),
					`SELECT name, type FROM pragma_table_info(?) ORDER BY cid`, table)
				if err != nil {
					return err
				}
				for colRows.Next() {
					var col, typ string
					if err := colRows.Scan(&col, &typ); err != nil {
						colRows.Close()
						return err
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", table, col, typ)
				}
				colRows.Close()
				if err := colRows.Err(); err != nil {
					return err
				}
			}
			return tw.Flush()
		},
	}
}

func dbStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Print row counts, file size, and event timestamp range",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := openStoreAndRegistry(cmd)
			if err != nil {
				return err
			}
			defer s.Close()

			stats, err := s.DBStats(context.Background())
			if err != nil {
				return err
			}
			fmt.Printf("Size on disk: %d bytes (%.2f MB)\n", stats.SizeBytes, float64(stats.SizeBytes)/(1024*1024))
			fmt.Printf("Events range: %s -> %s\n", stats.OldestEventTS.Format("2006-01-02 15:04:05"), stats.NewestEventTS.Format("2006-01-02 15:04:05"))

			tables := make([]string, 0, len(stats.Rows))
			for k := range stats.Rows {
				tables = append(tables, k)
			}
			sort.Strings(tables)
			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "\nTABLE\tROWS")
			for _, t := range tables {
				fmt.Fprintf(tw, "%s\t%d\n", t, stats.Rows[t])
			}
			return tw.Flush()
		},
	}
}

func dbVacuumCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vacuum",
		Short: "CHECKPOINT and force compaction; reclaim disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := openStoreAndRegistry(cmd)
			if err != nil {
				return err
			}
			defer s.Close()

			fmt.Println("Running vacuum...")
			if err := s.Vacuum(context.Background()); err != nil {
				return err
			}
			fmt.Println("Done.")
			return nil
		},
	}
}

func dbBackupCmd() *cobra.Command {
	var out string
	c := &cobra.Command{
		Use:   "backup",
		Short: "Snapshot the data directory to a tarball",
		RunE: func(cmd *cobra.Command, args []string) error {
			if out == "" {
				return fmt.Errorf("--out is required")
			}
			s, _, err := openStoreAndRegistry(cmd)
			if err != nil {
				return err
			}
			cfg, err := loadCfg(cmd)
			if err != nil {
				s.Close()
				return err
			}

			if err := s.Vacuum(context.Background()); err != nil {
				s.Close()
				return err
			}
			s.Close()

			fmt.Printf("Creating snapshot %s ...\n", out)
			if err := tarGz(cfg.Storage.DataDir, out); err != nil {
				return err
			}
			fmt.Println("Done.")
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "destination tarball path")
	return c
}

func dbRestoreCmd() *cobra.Command {
	var (
		in    string
		yes   bool
		force bool
	)
	c := &cobra.Command{
		Use:   "restore",
		Short: "Restore data dir from a snapshot tarball",
		RunE: func(cmd *cobra.Command, args []string) error {
			if in == "" {
				return fmt.Errorf("--in is required")
			}
			cfg, err := loadCfg(cmd)
			if err != nil {
				return err
			}

			if !yes && !confirm(fmt.Sprintf("Restore from %s into %s? Existing data will be replaced. [y/N]: ", in, cfg.Storage.DataDir)) {
				return fmt.Errorf("aborted")
			}

			dbPath := cfg.Storage.DataDir + "/events.db"
			if _, statErr := os.Stat(dbPath); statErr == nil && !force {
				probe, openErr := sqlite.Open(dbPath)
				if openErr != nil {
					return fmt.Errorf("cannot acquire exclusive lock on events.db (is `serve` running?): %w. Pass --force to override", openErr)
				}
				probe.Close()
			}

			tmp := cfg.Storage.DataDir + ".restore.tmp"
			if err := os.RemoveAll(tmp); err != nil {
				return err
			}
			if err := os.MkdirAll(tmp, 0o755); err != nil {
				return err
			}
			fmt.Printf("Extracting %s ...\n", in)
			if err := untarGz(in, tmp); err != nil {
				return err
			}

			archive := cfg.Storage.DataDir + ".replaced." + fmt.Sprintf("%d", time.Now().Unix())
			if err := os.Rename(cfg.Storage.DataDir, archive); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("archive existing: %w", err)
			}
			if err := os.Rename(tmp, cfg.Storage.DataDir); err != nil {
				return fmt.Errorf("install restored: %w", err)
			}
			fmt.Printf("Restore complete. Old data archived at %s\n", archive)
			fmt.Println("Restart `proxymetrics serve` to use the restored database.")
			return nil
		},
	}
	c.Flags().StringVar(&in, "in", "", "source tarball path")
	c.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	c.Flags().BoolVar(&force, "force", false, "skip the lock check (use only if serve is definitely not running)")
	return c
}

func tarGz(dir, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		fr, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fr.Close()
		_, err = io.Copy(tw, fr)
		return err
	})
}

func untarGz(in, dir string) error {
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, hdr.Name)
		cleanDst := filepath.Clean(dst) + string(os.PathSeparator)
		cleanDir := filepath.Clean(dir) + string(os.PathSeparator)
		if !strings.HasPrefix(cleanDst, cleanDir) {
			return fmt.Errorf("tar: illegal path %q", hdr.Name)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		fw, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		if _, err := io.Copy(fw, tr); err != nil {
			fw.Close()
			return err
		}
		fw.Close()
	}
}
