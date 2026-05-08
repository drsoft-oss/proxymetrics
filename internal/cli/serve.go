package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/anonymous-proxies/proxymetrics/internal/api"
	"github.com/anonymous-proxies/proxymetrics/internal/audit"
	"github.com/anonymous-proxies/proxymetrics/internal/broadcaster"
	"github.com/anonymous-proxies/proxymetrics/internal/config"
	"github.com/anonymous-proxies/proxymetrics/internal/events"
	"github.com/anonymous-proxies/proxymetrics/internal/geo"
	"github.com/anonymous-proxies/proxymetrics/internal/ipcheck"
	"github.com/anonymous-proxies/proxymetrics/internal/pricing"
	"github.com/anonymous-proxies/proxymetrics/internal/profile"
	"github.com/anonymous-proxies/proxymetrics/internal/proxy/ca"
	"github.com/anonymous-proxies/proxymetrics/internal/proxy/core"
	"github.com/anonymous-proxies/proxymetrics/internal/rollup"
	"github.com/anonymous-proxies/proxymetrics/internal/store/duckdb"
	"github.com/anonymous-proxies/proxymetrics/internal/ui"
)

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the proxy and admin listeners",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg(cmd)
			if err != nil {
				return err
			}
			return runServe(cfg)
		},
	}
}

func runServe(cfg config.Config) error {
	logger := newLogger(cfg.Logging)

	if err := os.MkdirAll(cfg.Storage.DataDir, 0o755); err != nil {
		return fmt.Errorf("data dir: %w", err)
	}
	dbPath := filepath.Join(cfg.Storage.DataDir, "events.duckdb")
	s, err := duckdb.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	// CA — generate on first run.
	if !ca.Exists(cfg.Storage.DataDir) {
		gen, err := ca.Generate(cfg.Storage.DataDir)
		if err != nil {
			return err
		}
		logger.Info("generated MITM CA", "fingerprint", gen.Fingerprint(), "valid_until", gen.Cert.NotAfter)
	}
	rootCA, err := ca.Load(cfg.Storage.DataDir)
	if err != nil {
		return err
	}
	logger.Info("loaded MITM CA", "fingerprint", rootCA.Fingerprint())
	leaf := ca.NewLeafFactory(rootCA, 10000, 24*time.Hour)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Geo audit module wiring (Task 15).
	auditStore := audit.NewStore(s.DB())
	centroidResolver := geo.NewCentroidResolver(geo.CentroidConfig{
		BaseURL:   cfg.Geo.Nominatim.BaseURL,
		UserAgent: cfg.Geo.Nominatim.UserAgent,
		Timeout:   cfg.Geo.Nominatim.Timeout,
		Cache:     auditStore,
	})
	geoCascade := geo.Cascade{geo.NewIPAPIIs(cfg.Geo.IPAPI.BaseURL)}
	if cfg.Geo.MaxMind.CityDB != "" {
		mm, err := geo.NewMaxMind(geo.MaxMindConfig{
			CityDB:           cfg.Geo.MaxMind.CityDB,
			ConnectionTypeDB: cfg.Geo.MaxMind.ConnectionTypeDB,
			ISPDB:            cfg.Geo.MaxMind.ISPDB,
			IPCheck:          ipcheck.New(cfg.IPCheck.APIs, http.DefaultClient),
		})
		if err != nil {
			logger.Warn("maxmind fallback disabled", "err", err)
		} else {
			defer mm.Close()
			geoCascade = append(geoCascade, mm)
		}
	} else {
		logger.Info("maxmind fallback not configured (geo.maxmind.city_db is empty)")
	}
	auditManager := audit.NewManager(audit.ManagerConfig{
		Store:             auditStore,
		Lookup:            geoCascade,
		Centroids:         centroidResolver,
		MaxInFlight:       cfg.Audit.MaxInFlight,
		PerRequestTimeout: cfg.Audit.PerRequestTimeout,
		MinRequestsPerRun: cfg.Audit.MinRequestsPerRun,
		MaxRequestsPerRun: cfg.Audit.MaxRequestsPerRun,
		BaseCtx:           rootCtx,
	})
	if err := auditManager.CleanupOrphans(rootCtx); err != nil {
		logger.Warn("audit orphan cleanup", "err", err)
	}

	registry, err := profile.NewRegistry(rootCtx, s)
	if err != nil {
		return err
	}

	// Pricing.
	calc := pricing.New(s)
	if err := calc.Reload(rootCtx); err != nil {
		return err
	}

	// shutdownSignal is closed by the shutdown goroutine after the proxy server
	// has finished draining in-flight requests, so the event writer doesn't
	// start its drain before those requests have had a chance to emit events.
	shutdownSignal := make(chan struct{})

	// Event writer.
	writer := events.NewWriter(events.Config{
		Store:           s,
		ChannelSize:     cfg.Events.ChannelSize,
		BatchSize:       cfg.Events.BatchSize,
		FlushInterval:   cfg.Events.FlushInterval,
		ShutdownSignal:  shutdownSignal,
		ProfileObserver: profileObserver{registry},
	})

	bcast := broadcaster.New()

	sched := rollup.NewScheduler(s, rollup.SchedulerConfig{
		TickEvery: time.Minute,
		Logger:    slogAdapter{logger},
	})

	// Proxy core.
	proxyCore, err := core.New(core.Config{
		DeploymentSecret: cfg.Server.DeploymentSecret,
		Registry:         registry,
		LeafFactory:      leaf,
		Emitter:          writer,
		Cost:             calc.Compute,
		DefaultTeam:      cfg.Defaults.Team,
		DefaultProject:   cfg.Defaults.Project,
		Logger:           slogAdapter{logger},
		Broadcaster:      bcast,
	})
	if err != nil {
		return err
	}

	// Admin mux.
	adminMux := http.NewServeMux()
	caMux := ca.HTTPHandler(rootCA)
	adminMux.Handle("/cacert", caMux)
	adminMux.Handle("/cacert.crt", caMux)
	adminMux.Handle("/cacert/fingerprint", caMux)
	profile.RegisterHTTP(adminMux, registry)
	profile.RegisterTestEndpoint(adminMux, registry,
		ipcheck.New(cfg.IPCheck.APIs, http.DefaultClient),
		profile.DefaultProxyClientBuilder{Timeout: 15 * time.Second},
	)
	api.Register(adminMux, s, bcast, cfg.Server.DeploymentSecret)
	audit.RegisterRoutes(adminMux, auditManager)
	adminMux.Handle("/", ui.Handler())

	proxySrv := &http.Server{Addr: cfg.Server.ProxyListen, Handler: proxyCore}
	adminSrv := &http.Server{Addr: cfg.Server.APIListen, Handler: adminMux}

	g, gctx := errgroup.WithContext(rootCtx)

	g.Go(func() error { return writer.Run(gctx) })
	g.Go(func() error { return sched.Run(gctx) })
	g.Go(func() error {
		logger.Info("admin listening", "addr", cfg.Server.APIListen)
		if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		logger.Info("proxy listening", "addr", cfg.Server.ProxyListen)
		if err := proxySrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-gctx.Done()
		drainCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.DrainTimeout)
		defer cancel()
		_ = proxySrv.Shutdown(drainCtx)
		_ = adminSrv.Shutdown(drainCtx)
		close(shutdownSignal) // Tell the writer it's safe to drain now.
		return nil
	})

	return g.Wait()
}

func newLogger(c config.Logging) *slog.Logger {
	level := slog.LevelInfo
	switch c.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	switch c.Format {
	case "text":
		h = slog.NewTextHandler(os.Stderr, opts)
	case "json":
		h = slog.NewJSONHandler(os.Stderr, opts)
	default:
		if isStdoutTTY() {
			h = slog.NewTextHandler(os.Stderr, opts)
		} else {
			h = slog.NewJSONHandler(os.Stderr, opts)
		}
	}
	return slog.New(h)
}

func isStdoutTTY() bool {
	st, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

// profileObserver adapts *profile.Registry to events.ProfileObserver. The
// writer calls Observe once per distinct (vendor, type) seen in a flushed
// batch; the registry handles caching/dedup internally so repeat calls for
// the same combo are cheap.
type profileObserver struct{ r *profile.Registry }

func (p profileObserver) Observe(ctx context.Context, vendor, typ string) error {
	return p.r.UpsertObserved(ctx, vendor, typ)
}

type slogAdapter struct{ l *slog.Logger }

func (s slogAdapter) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s slogAdapter) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s slogAdapter) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s slogAdapter) Error(msg string, args ...any) { s.l.Error(msg, args...) }
