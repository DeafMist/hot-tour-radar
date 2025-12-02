package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/DeafMist/hot-tour-radar/backend/internal/config"
	"github.com/DeafMist/hot-tour-radar/backend/internal/dedupe"
	"github.com/DeafMist/hot-tour-radar/backend/internal/elasticsearch"
	"github.com/DeafMist/hot-tour-radar/backend/internal/logger"
	"github.com/DeafMist/hot-tour-radar/backend/internal/models"
	"github.com/DeafMist/hot-tour-radar/backend/internal/processing"
)

type rawNews struct {
	Title     string `json:"title"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
}

type newsIndexer interface {
	IndexNews(ctx context.Context, doc models.NewsDocument) error
}

func main() {
	log := logger.New("worker")
	cfg, err := config.LoadWorker()
	if err != nil {
		log.Error("load config", slog.Any("err", err))
		os.Exit(1)
	}

	esClient, err := elasticsearch.New(cfg.ElasticsearchAddr, cfg.ElasticsearchIndex, log)
	if err != nil {
		log.Error("init elasticsearch", slog.Any("err", err))
		os.Exit(1)
	}

	cache := dedupe.NewCache(cfg.DedupeCapacity, cfg.DedupeTTL)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		Topic:          cfg.KafkaTopic,
		GroupID:        cfg.KafkaConsumer,
		QueueCapacity:  cfg.BatchSize,
		MinBytes:       1e3,
		MaxBytes:       10e6,
		CommitInterval: 0, // Disable auto-commit; manual commit only
	})
	defer reader.Close()

	dlqWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:     cfg.KafkaBrokers,
		Topic:       cfg.KafkaTopic + "_dlq",
		MaxAttempts: 3,
	})
	defer dlqWriter.Close()

	log.Info("worker started",
		slog.String("topic", cfg.KafkaTopic),
		slog.String("group", cfg.KafkaConsumer),
		slog.String("dlq_topic", cfg.KafkaTopic+"_dlq"),
	)

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Info("context canceled, stopping")
				return
			}
			log.Error("fetch message", slog.Any("err", err))
			continue
		}

		if err := processMessage(ctx, log, esClient, cache, cfg, msg); err != nil {
			log.Warn("process message failed, sending to DLQ",
				slog.Any("err", err),
				slog.Int("partition", msg.Partition),
				slog.Int64("offset", msg.Offset),
			)

			// Send to DLQ with error context, retry with backoff
			dlqMsg := kafka.Message{
				Value: msg.Value,
				Headers: append(msg.Headers,
					kafka.Header{Key: "original_partition", Value: []byte(fmt.Sprintf("%d", msg.Partition))},
					kafka.Header{Key: "original_offset", Value: []byte(fmt.Sprintf("%d", msg.Offset))},
					kafka.Header{Key: "error", Value: []byte(err.Error())},
					kafka.Header{Key: "timestamp", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
				),
			}

			// Retry DLQ write with exponential backoff
			dlqSuccess := false
			for attempt := range 5 {
				if dlqErr := dlqWriter.WriteMessages(ctx, dlqMsg); dlqErr == nil {
					dlqSuccess = true
					log.Info("message sent to DLQ",
						slog.Int("partition", msg.Partition),
						slog.Int64("offset", msg.Offset),
						slog.Int("attempt", attempt+1),
					)
					break
				} else {
					backoff := time.Duration(1<<uint(attempt)) * time.Second
					log.Warn("DLQ write failed, retrying",
						slog.Any("err", dlqErr),
						slog.Int("attempt", attempt+1),
						slog.Duration("backoff", backoff),
					)
					select {
					case <-time.After(backoff):
						// Continue to next attempt
					case <-ctx.Done():
						log.Info("context canceled during DLQ retry")
						return
					}
				}
			}

			// Only commit if DLQ write succeeded; otherwise skip commit and reprocess on restart
			if dlqSuccess {
				if err := reader.CommitMessages(ctx, msg); err != nil {
					log.Error("commit failed message to dlq", slog.Any("err", err))
				}
			} else {
				log.Error("DLQ write exhausted retries, message may be lost if later messages commit",
					slog.Int("partition", msg.Partition),
					slog.Int64("offset", msg.Offset),
				)
			}
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Error("commit message", slog.Any("err", err))
		}
	}
}

func processMessage(ctx context.Context, log *slog.Logger, esClient newsIndexer, cache *dedupe.Cache, cfg *config.Worker, msg kafka.Message) error {
	var payload rawNews
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		return err
	}

	title := strings.TrimSpace(payload.Title)
	text := strings.TrimSpace(payload.Text)
	urls := processing.ExtractURLs(text)
	if title == "" && text == "" {
		return errors.New("empty payload")
	}

	// Generate title from text if missing
	if title == "" && text != "" {
		title = processing.GenerateTitleFromText(text, 10)
	}

	ts := parseTimestamp(payload.Timestamp)
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	// Clean text for keyword extraction (remove URLs, punctuation, etc.)
	cleanedText := processing.CleanText(text)
	keywords := processing.ExtractKeywords(title+" "+cleanedText, cfg.KeywordLimit, cfg.KeywordMinLength)
	source := strings.TrimSpace(payload.Source)
	if source == "" {
		source = "unknown"
	}

	doc := models.NewsDocument{
		ID:        processing.BuildDocumentID(title, cleanedText, ts),
		Title:     title,
		Text:      text, // Original text with all punctuation and URLs
		Timestamp: ts,
		Keywords:  keywords,
		Source:    source,
		URLs:      urls,
	}

	if doc.ID == "" {
		doc.ID = uuid.NewString()
	}

	if cache.IsSeen(doc.ID) {
		log.Debug("duplicate news", slog.String("id", doc.ID))
		return nil
	}

	if err := esClient.IndexNews(ctx, doc); err != nil {
		return err
	}

	cache.MarkSeen(doc.ID)
	log.Info("indexed news", slog.String("id", doc.ID), slog.String("title", doc.Title))
	return nil
}

func parseTimestamp(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
	}

	for _, f := range formats {
		if ts, err := time.Parse(f, raw); err == nil {
			return ts
		}
	}

	return time.Time{}
}
