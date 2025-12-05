package config_test

import (
	"testing"
	"time"

	"github.com/DeafMist/hot-tour-radar/backend/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoadWorkerDefaults(t *testing.T) {
	t.Setenv("ELASTICSEARCH_ADDR", "")
	t.Setenv("ELASTICSEARCH_INDEX", "")
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("KAFKA_TOPIC", "")
	t.Setenv("KAFKA_CONSUMER_GROUP", "")

	cfg, err := config.LoadWorker()
	require.NoError(t, err)

	require.Equal(t, "http://elasticsearch:9200", cfg.ElasticsearchAddr)
	require.Equal(t, "news", cfg.ElasticsearchIndex)
	require.Len(t, cfg.KafkaBrokers, 1)
	require.Equal(t, "kafka:9092", cfg.KafkaBrokers[0])
	require.Equal(t, "news_raw", cfg.KafkaTopic)
	require.Equal(t, "news-worker", cfg.KafkaConsumer)
}

func TestLoadWorkerOverrides(t *testing.T) {
	t.Setenv("ELASTICSEARCH_ADDR", "http://localhost:9999")
	t.Setenv("ELASTICSEARCH_INDEX", "custom")
	t.Setenv("KAFKA_BROKERS", "broker-a:29092,broker-b:29093")
	t.Setenv("KAFKA_TOPIC", "custom_topic")
	t.Setenv("KAFKA_CONSUMER_GROUP", "custom-group")
	t.Setenv("WORKER_KEYWORD_LIMIT", "12")
	t.Setenv("WORKER_KEYWORD_MIN_LEN", "5")
	t.Setenv("WORKER_DEDUPE_CAPACITY", "5")
	t.Setenv("WORKER_DEDUPE_TTL", "48h")
	t.Setenv("WORKER_BATCH_SIZE", "3")
	t.Setenv("WORKER_COMMIT_INTERVAL", "5s")

	cfg, err := config.LoadWorker()
	require.NoError(t, err)

	require.Equal(t, "http://localhost:9999", cfg.ElasticsearchAddr)
	require.Equal(t, "custom", cfg.ElasticsearchIndex)
	require.Len(t, cfg.KafkaBrokers, 2)
	require.Equal(t, "broker-a:29092", cfg.KafkaBrokers[0])
	require.Equal(t, "custom_topic", cfg.KafkaTopic)
	require.Equal(t, "custom-group", cfg.KafkaConsumer)
	require.Equal(t, 12, cfg.KeywordLimit)
	require.Equal(t, 5, cfg.KeywordMinLength)
	require.Equal(t, 5, cfg.DedupeCapacity)
	require.Equal(t, 48*time.Hour, cfg.DedupeTTL)
	require.Equal(t, 3, cfg.BatchSize)
	require.Equal(t, 5*time.Second, cfg.CommitInterval)
}

func TestLoadAPI(t *testing.T) {
	t.Setenv("API_BIND_ADDR", ":9090")
	t.Setenv("API_PAGE_SIZE", "15")
	t.Setenv("API_MAX_PAGE_SIZE", "200")
	t.Setenv("ELASTICSEARCH_ADDR", "http://api-es:9200")
	t.Setenv("ELASTICSEARCH_INDEX", "api-index")

	cfg, err := config.LoadAPI()
	require.NoError(t, err)
	require.Equal(t, ":9090", cfg.BindAddr)
	require.Equal(t, 15, cfg.DefaultPage)
	require.Equal(t, 200, cfg.MaxPage)
	require.Equal(t, "http://api-es:9200", cfg.ElasticsearchAddr)
	require.Equal(t, "api-index", cfg.ElasticsearchIndex)
}

func TestLoadRetention(t *testing.T) {
	t.Setenv("ELASTICSEARCH_ADDR", "http://ret-es:9200")
	t.Setenv("ELASTICSEARCH_INDEX", "ret-index")
	t.Setenv("RETENTION_CRON", "12h")
	t.Setenv("RETENTION_MAX_AGE", "36h")
	t.Setenv("RETENTION_BATCH_SIZE", "123")

	cfg, err := config.LoadRetention()
	require.NoError(t, err)

	require.Equal(t, 12*time.Hour, cfg.Interval)
	require.Equal(t, 36*time.Hour, cfg.MaxAge)
	require.Equal(t, 123, cfg.BatchSize)
	require.Equal(t, "http://ret-es:9200", cfg.ElasticsearchAddr)
	require.Equal(t, "ret-index", cfg.ElasticsearchIndex)
}