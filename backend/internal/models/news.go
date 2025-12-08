package models

import "time"

// NewsDocument represents the canonical structure stored in Elasticsearch.
type NewsDocument struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Keywords  []string  `json:"keywords"`
	Source    string    `json:"source"`
	URLs      []string  `json:"urls"`
}
