package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DeafMist/hot-tour-radar/backend/internal/config"
	"github.com/DeafMist/hot-tour-radar/backend/internal/elasticsearch"
	"github.com/DeafMist/hot-tour-radar/backend/internal/logger"
)

func main() {
	log := logger.New("retention")
	cfg, err := config.LoadRetention()
	if err != nil {
		log.Error("load config", slog.Any("err", err))
		os.Exit(1)
	}

	esClient, err := elasticsearch.New(cfg.ElasticsearchAddr, cfg.ElasticsearchIndex, log)
	if err != nil {
		log.Error("init elasticsearch", slog.Any("err", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	log.Info("retention job running",
		slog.Duration("interval", cfg.Interval),
		slog.Duration("max_age", cfg.MaxAge),
	)

	runOnce(ctx, log, esClient, cfg)

	for {
		select {
		case <-ctx.Done():
			log.Info("shutdown signal received")
			return
		case <-ticker.C:
			runOnce(ctx, log, esClient, cfg)
		}
	}
}

func runOnce(ctx context.Context, log *slog.Logger, esClient *elasticsearch.Client, cfg *config.Retention) {
	subCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	deleted, err := esClient.DeleteOlderThan(subCtx, cfg.MaxAge, cfg.BatchSize)
	if err != nil {
		log.Error("retention run failed", slog.Any("err", err))
		return
	}

	log.Info("retention run completed", slog.Int64("deleted", deleted))
}
