package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	adminhttp "pipescope/internal/admin/http"
	adminservice "pipescope/internal/admin/service"
	"pipescope/internal/config"
	"pipescope/internal/gateway/proxy"
	"pipescope/internal/gateway/rule"
	"pipescope/internal/gateway/session"
	"pipescope/internal/geo/areacity"
	"pipescope/internal/geo/ip2region"
	sqlitestore "pipescope/internal/store/sqlite"

	_ "modernc.org/sqlite"
)

func main() {
	configPath := flag.String("config", "assets/config.example.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		log.Fatalf("pipescope runtime failed: %v", err)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	db, err := sql.Open("sqlite", cfg.Data.SQLitePath)
	if err != nil {
		return err
	}
	defer db.Close()

	store := sqlitestore.New(db)
	if err := store.InitSchema(ctx); err != nil {
		return err
	}

	queueSize := cfg.Writer.QueueSize
	if queueSize <= 0 {
		queueSize = 1024
	}

	events := make(chan session.Event, queueSize)
	writer := sqlitestore.NewWriter(
		db,
		events,
		cfg.Writer.BatchSize,
		time.Duration(cfg.Writer.FlushInterval)*time.Millisecond,
	)
	regionSearcher, err := ip2region.NewSearcherWithConfig(ip2region.Config{
		V4XDBPath:   cfg.Data.IP2RegionXDB,
		V6XDBPath:   cfg.Data.IP2RegionV6XDB,
		CachePolicy: cfg.Data.IP2RegionCachePolicy,
		Searchers:   cfg.Data.IP2RegionSearcherPool,
	})
	if err != nil {
		return fmt.Errorf("init ip2region searcher: %w", err)
	}
	defer regionSearcher.Close()

	areaMatcher, err := initAreaCityMatcher(ctx, db, cfg)
	if err != nil {
		return err
	}
	writer.SetGeoEnricher(
		regionSearcher,
		areaMatcher,
	)

	writerCtx, writerCancel := context.WithCancel(context.Background())
	defer writerCancel()
	writerErrCh := make(chan error, 1)
	go func() {
		writerErrCh <- writer.Run(writerCtx)
	}()

	runner := proxy.NewRunner(convertRules(cfg.ProxyRules), events)
	runner.SetTimeouts(
		time.Duration(cfg.Timeouts.DialMS)*time.Millisecond,
		time.Duration(cfg.Timeouts.IdleMS)*time.Millisecond,
	)
	if err := runner.Start(ctx); err != nil {
		return err
	}
	defer runner.Close()

	adminAddr := net.JoinHostPort(cfg.Admin.Host, fmt.Sprintf("%d", cfg.Admin.Port))
	httpSrv := &nethttp.Server{
		Addr:    adminAddr,
		Handler: newAdminHandler(db),
	}

	httpErrCh := make(chan error, 1)
	go func() {
		err := httpSrv.ListenAndServe()
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			httpErrCh <- err
			return
		}
		httpErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		writerCancel()
		if err := <-writerErrCh; err != nil {
			return err
		}
		return nil
	case err := <-httpErrCh:
		writerCancel()
		_ = <-writerErrCh
		return err
	}
}

func initAreaCityMatcher(ctx context.Context, db *sql.DB, cfg *config.Config) (sqlitestore.AdcodeMatcher, error) {
	if strings.TrimSpace(cfg.Data.AreaCityAPIBaseURL) != "" {
		m := areacity.NewHTTPMatcher(cfg.Data.AreaCityAPIBaseURL, cfg.Data.AreaCityAPIInstance)
		if err := m.Ping(ctx); err != nil {
			return nil, fmt.Errorf("ping areacity api failed: %w", err)
		}
		return m, nil
	}

	if err := loadAreaCityData(ctx, db, cfg.Data.AreaCityCSVPath); err != nil {
		return nil, err
	}
	return areacity.NewMatcher(db), nil
}

func loadAreaCityData(ctx context.Context, db *sql.DB, csvPath string) error {
	if csvPath == "" {
		return nil
	}
	if _, err := os.Stat(csvPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return areacity.NewImporter(db).ImportCSV(ctx, csvPath)
}

func newAdminHandler(db *sql.DB) nethttp.Handler {
	svc := adminservice.New(db)
	return adminhttp.NewServer(svc).Handler()
}

func convertRules(src []config.ProxyRule) []rule.Rule {
	out := make([]rule.Rule, 0, len(src))
	for _, r := range src {
		out = append(out, rule.Rule{
			ID:      r.ID,
			Listen:  r.Listen,
			Forward: r.Forward,
		})
	}
	return out
}
