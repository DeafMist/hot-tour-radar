package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/DeafMist/hot-tour-radar/backend/internal/models"
)

// Client wraps go-elasticsearch with helpers tailored to this project.
type Client struct {
	es    *elasticsearch.Client
	index string
	log   *slog.Logger
}

// SearchParams narrow the search endpoint query.
type SearchParams struct {
	Query    string
	Keywords []string
	Source   string
	From     int
	Size     int
	Sort     string
	Start    *time.Time
	End      *time.Time
}

// SearchResult bundles hits and total count.
type SearchResult struct {
	Total int64
	Items []models.NewsDocument
}

// New instantiates the Elasticsearch client.
func New(addr, index string, logger *slog.Logger) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{addr},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}

	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &Client{es: es, index: index, log: logger}, nil
}

// Ping checks if Elasticsearch is available.
func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("ping elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch ping failed: %s", res.Status())
	}

	return nil
}

// IndexNews writes a document into Elasticsearch.
func (c *Client) IndexNews(ctx context.Context, doc models.NewsDocument) error {
	payload, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal doc: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      c.index,
		DocumentID: doc.ID,
		Body:       bytes.NewReader(payload),
		Refresh:    "false",
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("index doc: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("index doc failed: %s", strings.TrimSpace(string(body)))
	}

	return nil
}

// SearchNews executes a bool query with optional filters.
func (c *Client) SearchNews(ctx context.Context, params SearchParams) (*SearchResult, error) {
	if params.Size <= 0 {
		params.Size = 20
	}
	if params.Size > 200 {
		params.Size = 200
	}
	if params.From < 0 {
		params.From = 0
	}

	must := make([]map[string]any, 0, 2)
	filters := make([]map[string]any, 0, 3)

	if params.Query != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  params.Query,
				"fields": []string{"title^2", "text"},
			},
		})
	}

	if len(params.Keywords) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"keywords": params.Keywords,
			},
		})
	}

	if params.Source != "" {
		filters = append(filters, map[string]any{
			"term": map[string]any{
				"source": params.Source,
			},
		})
	}

	if params.Start != nil || params.End != nil {
		rangeQuery := map[string]any{}
		if params.Start != nil {
			rangeQuery["gte"] = params.Start.UTC().Format(time.RFC3339)
		}
		if params.End != nil {
			rangeQuery["lte"] = params.End.UTC().Format(time.RFC3339)
		}
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"timestamp": rangeQuery,
			},
		})
	}

	boolQuery := map[string]any{}
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}
	if len(must) == 0 && len(filters) == 0 {
		boolQuery["must"] = []map[string]any{
			{"match_all": map[string]any{}},
		}
	}

	body := map[string]any{
		"from":             params.From,
		"size":             params.Size,
		"track_total_hits": true,
		"query": map[string]any{
			"bool": boolQuery,
		},
	}

	sortField := params.Sort
	if sortField == "" {
		sortField = "timestamp:desc"
	}

	parts := strings.Split(sortField, ":")
	order := "desc"
	field := parts[0]
	if field == "" {
		field = "timestamp"
	}
	if len(parts) > 1 && parts[1] != "" {
		order = parts[1]
	}
	body["sort"] = []map[string]any{
		{field: map[string]any{"order": order}},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal search body: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.index),
		c.es.Search.WithBody(bytes.NewReader(payload)),
	)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		data, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search failed: %s", strings.TrimSpace(string(data)))
	}

	var parsed struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source models.NewsDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	items := make([]models.NewsDocument, 0, len(parsed.Hits.Hits))
	for _, hit := range parsed.Hits.Hits {
		items = append(items, hit.Source)
	}

	return &SearchResult{
		Total: parsed.Hits.Total.Value,
		Items: items,
	}, nil
}

// DeleteOlderThan removes documents older than ttl using batched delete-by-query.
// It loops until a batch returns fewer deleted documents than the requested batchSize.
func (c *Client) DeleteOlderThan(ctx context.Context, maxAge time.Duration, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}

	cutoff := time.Now().Add(-maxAge).UTC().Format(time.RFC3339)
	totalDeleted := int64(0)

	for {
		body := map[string]any{
			"query": map[string]any{
				"range": map[string]any{
					"timestamp": map[string]any{
						"lte": cutoff,
					},
				},
			},
		}

		payload, err := json.Marshal(body)
		if err != nil {
			return totalDeleted, fmt.Errorf("marshal delete body: %w", err)
		}

		res, err := c.es.DeleteByQuery(
			[]string{c.index},
			bytes.NewReader(payload),
			c.es.DeleteByQuery.WithContext(ctx),
			c.es.DeleteByQuery.WithWaitForCompletion(true),
			c.es.DeleteByQuery.WithConflicts("proceed"),
			c.es.DeleteByQuery.WithScrollSize(batchSize),
		)
		if err != nil {
			return totalDeleted, fmt.Errorf("delete by query: %w", err)
		}

		if res.IsError() {
			data, _ := io.ReadAll(res.Body)
			res.Body.Close()
			return totalDeleted, fmt.Errorf("delete by query failed: %s", strings.TrimSpace(string(data)))
		}

		var parsed struct {
			Deleted int64 `json:"deleted"`
		}
		if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
			res.Body.Close()
			return totalDeleted, fmt.Errorf("decode delete response: %w", err)
		}
		res.Body.Close()

		totalDeleted += parsed.Deleted

		if parsed.Deleted < int64(batchSize) {
			break
		}
	}

	return totalDeleted, nil
}

// Health pings Elasticsearch to ensure connectivity.
func (c *Client) Health(ctx context.Context) error {
	res, err := c.es.Cluster.Health(c.es.Cluster.Health.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusBadRequest {
		data, _ := io.ReadAll(res.Body)
		return fmt.Errorf("cluster health bad: %s", strings.TrimSpace(string(data)))
	}
	return nil
}
