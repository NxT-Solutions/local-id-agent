package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rqc-icu/localid-agent/services/agent/internal/api"
	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/logging"
	"github.com/rqc-icu/localid-agent/services/agent/internal/providers"
)

var (
	loadConfig    = config.Load
	setupLogging  = logging.Setup
	newProvider   = providers.New
	newServer     = api.NewServer
	notifyContext = signal.NotifyContext
	runMain       = run
	exitMain      = os.Exit
)

func run(args []string) error {
	fs := flag.NewFlagSet("localid-agent", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", "", "path to JSON config file (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *configPath == "" {
		return fmt.Errorf("missing required --config flag")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := setupLogging(cfg.Logging.Level)

	provider, err := newProvider(cfg.Providers)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	ctx, stop := notifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := newServer(cfg, provider, logger)
	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func main() {
	if err := runMain(os.Args[1:]); err != nil {
		slog.Error(err.Error())
		exitMain(1)
	}
}
