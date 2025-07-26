package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bxrne/branchlore/internal/config"
	"github.com/bxrne/branchlore/internal/git"
	"github.com/bxrne/branchlore/internal/metrics"
	"github.com/bxrne/branchlore/internal/server"
	"github.com/bxrne/branchlore/internal/storage"
	"github.com/bxrne/branchlore/internal/types"
)

var (
	configPath = flag.String("config", "", "config file path")
	addr       = flag.String("addr", "", "server address")
	verbose    = flag.Bool("verbose", false, "verbose logging")
	simulate   = flag.Bool("simulate", false, "run in simulation mode")
)

type Daemon struct {
	config  *types.Config
	git     *git.Repository
	storage *storage.FileSystem
	server  *server.HTTPServer
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewDaemon(cfg *types.Config) *Daemon {
	fs := storage.NewFileSystem(cfg)
	gitRepo := git.NewRepository(fs.GetRepoPath())
	httpServer := server.NewHTTPServer(cfg, gitRepo, fs)

	return &Daemon{
		config:  cfg,
		git:     gitRepo,
		storage: fs,
		server:  httpServer,
	}
}

func (d *Daemon) Start(ctx context.Context) error {
	slog.Info("Starting Branchlore daemon", "config", d.config)

	ctx, d.cancel = context.WithCancel(ctx)

	if err := d.git.Init(d.storage.GetRepoPath()); err != nil {
		return err
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.runMonitoringLoop(ctx)
	}()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		if err := d.server.Start(ctx, d.config.ServerAddr); err != nil {
			slog.Error("HTTP server failed", "error", err)
		}
	}()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.runHealthChecks(ctx)
	}()

	slog.Info("Daemon started successfully",
		"server_addr", d.config.ServerAddr,
		"repo_path", d.storage.GetRepoPath(),
		"simulate", d.config.Simulate)

	return nil
}

func (d *Daemon) Stop() error {
	slog.Info("Stopping daemon")

	if d.cancel != nil {
		d.cancel()
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Daemon stopped gracefully")
	case <-time.After(10 * time.Second):
		slog.Warn("Daemon stop timeout, forcing shutdown")
	}

	return nil
}

func (d *Daemon) runMonitoringLoop(ctx context.Context) {
	slog.Info("Starting monitoring loop")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Monitoring loop stopped")
			return
		case <-ticker.C:
			d.performMaintenance()
		}
	}
}

func (d *Daemon) performMaintenance() {
	start := time.Now()
	defer func() {
		metrics.MaintenanceLoopDuration.Observe(time.Since(start).Seconds())
	}()

	if d.config.Simulate {
		slog.Debug("Simulating maintenance tasks")
		return
	}

	branches, err := d.git.ListBranches()
	if err != nil {
		slog.Error("Failed to list branches during maintenance", "error", err)
		return
	}

	dbPaths, err := d.storage.GetBranchDBs()
	if err != nil {
		slog.Error("Failed to get branch databases during maintenance", "error", err)
		return
	}

	metrics.BranchCount.Set(float64(len(branches)))
	metrics.DatabaseCount.Set(float64(len(dbPaths)))

	slog.Debug("Maintenance check completed",
		"branches", len(branches),
		"databases", len(dbPaths))

	for branchName, dbPath := range dbPaths {
		size, err := d.storage.GetSize(dbPath)
		if err != nil {
			continue
		}

		metrics.DatabaseSize.WithLabelValues(branchName).Set(float64(size))

		slog.Debug("Branch database status",
			"branch", branchName,
			"size", size,
			"path", dbPath)
	}
}

func (d *Daemon) runHealthChecks(ctx context.Context) {
	slog.Info("Starting health check loop")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Health check loop stopped")
			return
		case <-ticker.C:
			d.performHealthCheck()
		}
	}
}

func (d *Daemon) performHealthCheck() {
	start := time.Now()
	defer func() {
		metrics.HealthCheckDuration.Observe(time.Since(start).Seconds())
	}()

	if d.config.Simulate {
		slog.Debug("Simulating health checks")
		return
	}

	repoPath := d.storage.GetRepoPath()
	if !d.storage.PathExists(repoPath) {
		slog.Error("Repository path does not exist", "path", repoPath)
		return
	}

	hash, err := d.git.GetCurrentHash()
	if err != nil {
		slog.Error("Failed to get current hash during health check", "error", err)
		return
	}

	slog.Debug("Health check passed",
		"repo_path", repoPath,
		"current_hash", hash[:8])
}

func main() {
	flag.Parse()

	if *verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
	}

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	daemon := NewDaemon(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := daemon.Start(ctx); err != nil {
		slog.Error("Failed to start daemon", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	slog.Info("Shutdown signal received")

	if err := daemon.Stop(); err != nil {
		slog.Error("Error during shutdown", "error", err)
		os.Exit(1)
	}
}

func loadConfig() (*types.Config, error) {
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = config.GetDefaultConfigPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	config.LoadFromEnv(cfg)

	if *addr != "" {
		cfg.ServerAddr = *addr
	}

	if *simulate {
		cfg.Simulate = true
	}

	return cfg, nil
}
