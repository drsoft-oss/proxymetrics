package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/drsoft-oss/proxymetrics/internal/config"
	"github.com/drsoft-oss/proxymetrics/internal/proxy/ca"
)

func cacertCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cacert", Short: "Manage the MITM CA"}
	cmd.AddCommand(cacertInitCmd(), cacertShowCmd(), cacertPathCmd(), cacertRotateCmd())
	return cmd
}

func loadCfg(cmd *cobra.Command) (config.Config, error) {
	path, _ := cmd.Flags().GetString("config")
	return config.Load(path)
}

func cacertInitCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Generate the CA if missing",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg(cmd)
			if err != nil {
				return err
			}
			if ca.Exists(cfg.Storage.DataDir) && !force {
				return fmt.Errorf("cacert already exists in %s; pass --force to overwrite", cfg.Storage.DataDir)
			}
			if force && ca.Exists(cfg.Storage.DataDir) {
				if _, err := ca.Rotate(cfg.Storage.DataDir); err != nil {
					return err
				}
				a, err := ca.Load(cfg.Storage.DataDir)
				if err != nil {
					return err
				}
				fmt.Printf("CA: %s\nFingerprint: %s\nValid until: %s\n", a.PEMPath, a.Fingerprint(), a.Cert.NotAfter.UTC())
				return nil
			}
			a, err := ca.Generate(cfg.Storage.DataDir)
			if err != nil {
				return err
			}
			fmt.Printf("CA: %s\nFingerprint: %s\nValid until: %s\n", a.PEMPath, a.Fingerprint(), a.Cert.NotAfter.UTC())
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "regenerate even if a CA already exists")
	return c
}

func cacertShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the CA PEM, fingerprint, and validity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg(cmd)
			if err != nil {
				return err
			}
			a, err := ca.Load(cfg.Storage.DataDir)
			if err != nil {
				return err
			}
			pem, err := os.ReadFile(a.PEMPath)
			if err != nil {
				return err
			}
			fmt.Printf("Fingerprint: %s\nValid until: %s\nPath: %s\n\n%s",
				a.Fingerprint(), a.Cert.NotAfter.UTC(), a.PEMPath, pem)
			return nil
		},
	}
}

func cacertPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print absolute paths to the CA files",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg(cmd)
			if err != nil {
				return err
			}
			a, err := ca.Load(cfg.Storage.DataDir)
			if err != nil {
				return err
			}
			fmt.Println(a.PEMPath)
			fmt.Println(a.DERPath)
			fmt.Println(a.KeyPath)
			return nil
		},
	}
}

func cacertRotateCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "rotate",
		Short: "Archive the existing CA and generate a fresh one",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg(cmd)
			if err != nil {
				return err
			}
			if !yes && !confirm("Rotate CA? Every scraper trusting the current CA will fail until it re-installs the new one. [y/N]: ") {
				return fmt.Errorf("aborted")
			}
			a, err := ca.Rotate(cfg.Storage.DataDir)
			if err != nil {
				return err
			}
			fmt.Printf("New CA: %s\nFingerprint: %s\n", a.PEMPath, a.Fingerprint())
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}

func confirm(prompt string) bool {
	if !isStdinTTY() {
		return false
	}
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y"
}

func isStdinTTY() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}
