package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rqc-icu/localid-agent/internal/api"
	"github.com/rqc-icu/localid-agent/internal/config"
	"github.com/rqc-icu/localid-agent/internal/logging"
	"github.com/rqc-icu/localid-agent/internal/providers"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config file (required)")
	flag.Parse()

	if *configPath == "" {
		slog.Error("missing required --config flag")
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := logging.Setup(cfg.Logging.Level)

	provider, err := providers.New(cfg.Providers)
	if err != nil {
		logger.Error("failed to create provider", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := api.NewServer(cfg, provider, logger)
	if err := server.Run(ctx); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
