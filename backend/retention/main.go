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

	// Retry Elasticsearch connection with backoff
	var esClient *elasticsearch.Client
	maxRetries := 10
	retryDelay := 2 * time.Second
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	for i := 0; i < maxRetries; i++ {
		esClient, err = elasticsearch.New(cfg.ElasticsearchAddr, cfg.ElasticsearchIndex, log)
		if err != nil {
			log.Warn("failed to create elasticsearch client, retrying",
				slog.Any("err", err),
				slog.Int("attempt", i+1),
				slog.Int("max_retries", maxRetries),
			)
		} else {
			// Verify connectivity with ping
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if pingErr := esClient.Ping(pingCtx); pingErr == nil {
				cancel()
				break
			} else {
				log.Warn("elasticsearch ping failed, retrying",
					slog.Any("err", pingErr),
					slog.Int("attempt", i+1),
					slog.Int("max_retries", maxRetries),
					slog.Duration("retry_in", retryDelay),
				)
			}
			cancel()
		}

		select {
		case <-time.After(retryDelay):
			// Continue to next attempt
		case <-ctx.Done():
			log.Info("shutdown signal received during startup")
			os.Exit(0)
		}
		retryDelay *= 2 // Exponential backoff
		if retryDelay > 30*time.Second {
			retryDelay = 30 * time.Second
		}
	}

	// Final check
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if esClient == nil || esClient.Ping(pingCtx) != nil {
		log.Error("failed to connect to elasticsearch after retries")
		os.Exit(1)
	}

	log.Info("connected to elasticsearch")

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	log.Info("retention job running",
		slog.Duration("interval", cfg.Interval),
		slog.Duration("max_age", cfg.MaxAge),
	)

	// Run immediately on start, but don't fail if ES is temporarily unavailable
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
	subCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	deleted, err := esClient.DeleteOlderThan(subCtx, cfg.MaxAge, cfg.BatchSize)
	if err != nil {
		log.Warn("retention run failed (will retry on next interval)", slog.Any("err", err))
		return
	}

	if deleted > 0 {
		log.Info("retention run completed", slog.Int64("deleted", deleted))
	} else {
		log.Debug("retention run completed, no old documents found")
	}
}
