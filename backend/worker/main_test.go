package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"

	"github.com/DeafMist/hot-tour-radar/backend/internal/config"
	"github.com/DeafMist/hot-tour-radar/backend/internal/dedupe"
	"github.com/DeafMist/hot-tour-radar/backend/internal/models"
)

type stubIndexer struct {
	docs []models.NewsDocument
}

func (s *stubIndexer) IndexNews(_ context.Context, doc models.NewsDocument) error {
	s.docs = append(s.docs, doc)
	return nil
}

func TestProcessMessageIndexesDocument(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cache := dedupe.NewCache(100, time.Hour)
	idx := &stubIndexer{}
	cfg := &config.Worker{
		Common: config.Common{
			ElasticsearchAddr:  "http://test",
			ElasticsearchIndex: "news",
		},
		KeywordLimit:     5,
		KeywordMinLength: 3,
	}

	payload := rawNews{
		Title:     "Горящий тур",
		Text:      "<b>Море и солнце</b> ждут",
		Timestamp: "2024-01-02T15:04:05Z",
		Source:    "rss",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	msg := kafka.Message{Value: data}

	require.NoError(t, processMessage(context.Background(), log, idx, cache, cfg, msg))

	require.Equal(t, 1, len(idx.docs))

	doc := idx.docs[0]
	require.Equal(t, "Горящий тур", doc.Title)
	require.Equal(t, "rss", doc.Source)
	require.NotEmpty(t, doc.Keywords)

	require.NoError(t, processMessage(context.Background(), log, idx, cache, cfg, msg))
	require.Equal(t, 1, len(idx.docs))
}

func TestParseTimestamp(t *testing.T) {
	ts := parseTimestamp("2024-02-03T04:05:06Z")
	require.False(t, ts.IsZero())
	require.Equal(t, 2024, ts.Year())
	require.Equal(t, time.UTC, ts.Location())
	require.Equal(t, 2, int(ts.Month()))
	require.Equal(t, 3, ts.Day())
	require.Equal(t, 4, ts.Hour())
	require.Equal(t, 5, ts.Minute())
	require.Equal(t, 6, ts.Second())

	legacy := parseTimestamp("2024-02-03 04:05:06")
	require.False(t, legacy.IsZero())
	require.Equal(t, 2024, legacy.Year())
	require.Equal(t, 2, int(legacy.Month()))
	require.Equal(t, 3, legacy.Day())
	require.Equal(t, 4, legacy.Hour())
	require.Equal(t, 5, legacy.Minute())
	require.Equal(t, 6, legacy.Second())

	require.True(t, parseTimestamp("invalid").IsZero())
}

func TestProcessMessageGeneratesTitleWhenMissing(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cache := dedupe.NewCache(100, time.Hour)
	idx := &stubIndexer{}
	cfg := &config.Worker{
		Common: config.Common{
			ElasticsearchAddr:  "http://test",
			ElasticsearchIndex: "news",
		},
		KeywordLimit:     5,
		KeywordMinLength: 3,
	}

	payload := rawNews{
		Title:     "", // Empty title
		Text:      "Горящий тур в Турцию! Всего 30000 рублей. Вылет завтра.",
		Timestamp: "2024-01-02T15:04:05Z",
		Source:    "telegram",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	msg := kafka.Message{Value: data}

	require.NoError(t, processMessage(context.Background(), log, idx, cache, cfg, msg))

	require.Equal(t, 1, len(idx.docs))

	doc := idx.docs[0]
	// Title should be auto-generated from text
	require.Equal(t, "Горящий тур в Турцию", doc.Title)
	require.Equal(t, "telegram", doc.Source)
	require.NotEmpty(t, doc.Keywords)
}
