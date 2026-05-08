package main

import (
	"fmt"
	"os"

	"github.com/anonymous-proxies/proxymetrics/internal/cli"
)

func main() {
	if err := cli.Root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
