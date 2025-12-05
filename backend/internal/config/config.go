package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Common contains Elasticsearch parameters shared by every service.
type Common struct {
	ElasticsearchAddr  string
	ElasticsearchIndex string
}

// Worker holds configuration for the Kafka -> Elasticsearch worker.
type Worker struct {
	Common
	KafkaBrokers     []string
	KafkaTopic       string
	KafkaConsumer    string
	KeywordLimit     int
	KeywordMinLength int
	DedupeCapacity   int
	DedupeTTL        time.Duration
	BatchSize        int
	CommitInterval   time.Duration
}

// API describes HTTP-layer configuration.
type API struct {
	Common
	BindAddr    string
	DefaultPage int
	MaxPage     int
}

// Retention configures the cleanup loop.
type Retention struct {
	Common
	Interval  time.Duration
	MaxAge    time.Duration
	BatchSize int
}

// LoadWorker builds a Worker config from environment variables.
func LoadWorker() (*Worker, error) {
	c := &Worker{
		Common: Common{
			ElasticsearchAddr:  getEnv("ELASTICSEARCH_ADDR", "http://elasticsearch:9200"),
			ElasticsearchIndex: getEnv("ELASTICSEARCH_INDEX", "news"),
		},
		KafkaBrokers:     splitAndTrim(getEnv("KAFKA_BROKERS", "kafka:9092")),
		KafkaTopic:       getEnv("KAFKA_TOPIC", "news_raw"),
		KafkaConsumer:    getEnv("KAFKA_CONSUMER_GROUP", "news-worker"),
		KeywordLimit:     getInt("WORKER_KEYWORD_LIMIT", 8),
		KeywordMinLength: getInt("WORKER_KEYWORD_MIN_LEN", 4),
		DedupeCapacity:   getInt("WORKER_DEDUPE_CAPACITY", 20000),
		DedupeTTL:        getDuration("WORKER_DEDUPE_TTL", "24h"),
		BatchSize:        getInt("WORKER_BATCH_SIZE", 10),
		CommitInterval:   getDuration("WORKER_COMMIT_INTERVAL", "2s"),
	}

	if len(c.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}

	if c.BatchSize <= 0 {
		return nil, fmt.Errorf("WORKER_BATCH_SIZE must be positive")
	}
	if c.DedupeCapacity <= 0 {
		return nil, fmt.Errorf("WORKER_DEDUPE_CAPACITY must be positive")
	}
	if c.KeywordLimit <= 0 {
		return nil, fmt.Errorf("WORKER_KEYWORD_LIMIT must be positive")
	}
	if c.KeywordMinLength < 0 {
		return nil, fmt.Errorf("WORKER_KEYWORD_MIN_LEN cannot be negative")
	}

	return c, nil
}

// LoadAPI builds an API config from environment variables.
func LoadAPI() (*API, error) {
	c := &API{
		Common: Common{
			ElasticsearchAddr:  getEnv("ELASTICSEARCH_ADDR", "http://elasticsearch:9200"),
			ElasticsearchIndex: getEnv("ELASTICSEARCH_INDEX", "news"),
		},
		BindAddr:    getEnv("API_BIND_ADDR", "0.0.0.0:8080"),
		DefaultPage: getInt("API_PAGE_SIZE", 20),
		MaxPage:     getInt("API_MAX_PAGE_SIZE", 100),
	}

	if c.DefaultPage <= 0 {
		return nil, fmt.Errorf("API_PAGE_SIZE must be positive")
	}
	if c.MaxPage <= 0 {
		return nil, fmt.Errorf("API_MAX_PAGE_SIZE must be positive")
	}
	if c.DefaultPage > c.MaxPage {
		return nil, fmt.Errorf("API_PAGE_SIZE cannot exceed API_MAX_PAGE_SIZE")
	}

	return c, nil
}

// LoadRetention builds a Retention config from environment variables.
func LoadRetention() (*Retention, error) {
	c := &Retention{
		Common: Common{
			ElasticsearchAddr:  getEnv("ELASTICSEARCH_ADDR", "http://elasticsearch:9200"),
			ElasticsearchIndex: getEnv("ELASTICSEARCH_INDEX", "news"),
		},
		Interval:  getDuration("RETENTION_CRON", "24h"),
		MaxAge:    getDuration("RETENTION_MAX_AGE", "168h"),
		BatchSize: getInt("RETENTION_BATCH_SIZE", 500),
	}

	if c.MaxAge <= 0 {
		return nil, fmt.Errorf("RETENTION_MAX_AGE must be positive")
	}

	if c.Interval <= 0 {
		return nil, fmt.Errorf("RETENTION_CRON must be positive")
	}

	if c.BatchSize <= 0 {
		return nil, fmt.Errorf("RETENTION_BATCH_SIZE must be positive")
	}

	return c, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getDuration(key, fallback string) time.Duration {
	raw := getEnv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		fd, ferr := parseDuration(fallback)
		if ferr != nil {
			panic(fmt.Sprintf("invalid fallback duration %q: %v", fallback, ferr))
		}
		return fd
	}
	return d
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseDuration(raw string) (time.Duration, error) {
	return time.ParseDuration(raw)
}
