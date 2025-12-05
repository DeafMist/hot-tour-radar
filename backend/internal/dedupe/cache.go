package dedupe

import (
	"sync"
	"time"
)

type entry struct {
	key string
	ts  time.Time
}

// Cache keeps a fixed-size set of recently processed document hashes.
type Cache struct {
	mu       sync.Mutex
	items    map[string]time.Time
	order    []entry
	capacity int
	ttl      time.Duration
}

// NewCache creates a cache with the provided capacity and ttl.
func NewCache(capacity int, ttl time.Duration) *Cache {
	if capacity <= 0 {
		capacity = 1
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &Cache{
		items:    make(map[string]time.Time, capacity),
		order:    make([]entry, 0, capacity),
		capacity: capacity,
		ttl:      ttl,
	}
}

// IsSeen returns true when the key has already been observed inside the ttl window.
// It does not mark the key as seen; use MarkSeen() to record a key.
func (c *Cache) IsSeen(key string) bool {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if ts, ok := c.items[key]; ok {
		if now.Sub(ts) <= c.ttl {
			return true
		}
	}
	return false
}

// MarkSeen records that a key has been processed.
func (c *Cache) MarkSeen(key string) {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = now
	c.order = append(c.order, entry{key: key, ts: now})
	c.compact(now)
}

func (c *Cache) compact(now time.Time) {
	cutoff := now.Add(-c.ttl)

	for len(c.order) > 0 && (len(c.items) > c.capacity || c.order[0].ts.Before(cutoff)) {
		oldest := c.order[0]
		c.order = c.order[1:]

		if ts, ok := c.items[oldest.key]; ok {
			if ts == oldest.ts {
				delete(c.items, oldest.key)
			}
		}
	}
}
