// Package ui embeds the dashboard SPA produced by the Vite build into the
// proxymetrics binary. The Makefile's build-ui target populates ui-dist/
// from ../../ui/dist/ before `go build` runs.
package ui

import "embed"

//go:embed all:ui-dist
var distFS embed.FS
