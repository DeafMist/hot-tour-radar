package dedupe_test

import (
	"testing"
	"time"

	"github.com/DeafMist/hot-tour-radar/backend/internal/dedupe"
	"github.com/stretchr/testify/require"
)

func TestCacheSeenDuplicate(t *testing.T) {
	cache := dedupe.NewCache(10, time.Minute)
	require.False(t, cache.IsSeen("alpha"))
	cache.MarkSeen("alpha")
	require.True(t, cache.IsSeen("alpha"))
}

func TestCacheTTLExpiry(t *testing.T) {
	cache := dedupe.NewCache(10, 20*time.Millisecond)
	require.False(t, cache.IsSeen("beta"))
	cache.MarkSeen("beta")
	time.Sleep(25 * time.Millisecond)
	require.False(t, cache.IsSeen("beta"))
}

func TestCacheCapacityEvictsOldest(t *testing.T) {
	cache := dedupe.NewCache(1, time.Minute)
	require.False(t, cache.IsSeen("first"))
	cache.MarkSeen("first")

	require.False(t, cache.IsSeen("second"))
	cache.MarkSeen("second")

	require.False(t, cache.IsSeen("first"))
	require.True(t, cache.IsSeen("second"))
}
