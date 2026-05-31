package store

import (
	"sync"
	"time"
)

const (
	defaultShardCount     = 256
	defaultReaperInterval = time.Minute
)

// Store is a sharded in-memory key-value store.
type Store struct {
	shards      []*shard
	shardMask   uint64
	defaultTTL  time.Duration
	stats       statCounters
	reaperEvery time.Duration

	stop     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// New creates a Store with optional configuration.
func New(opts ...Option) *Store {
	cfg := config{
		shardCount:     defaultShardCount,
		reaperInterval: defaultReaperInterval,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	shardCount := nextPowerOfTwo(cfg.shardCount)
	shards := make([]*shard, shardCount)
	for i := range shards {
		shards[i] = newShard()
	}

	s := &Store{
		shards:      shards,
		shardMask:   uint64(shardCount - 1),
		defaultTTL:  cfg.defaultTTL,
		reaperEvery: cfg.reaperInterval,
		stop:        make(chan struct{}),
	}

	if cfg.reaperInterval > 0 {
		s.wg.Add(1)
		go s.reapLoop()
	}

	return s
}

func (s *Store) shardFor(key string) *shard {
	return s.shards[KeyHash(key)&s.shardMask]
}

// Get returns the value for key. Returned bytes must not be mutated by the caller.
func (s *Store) Get(key string) (value []byte, ok bool) {
	return s.shardFor(key).get(key, time.Now().UnixNano(), &s.stats)
}

// Set stores value under key using the default TTL if configured.
func (s *Store) Set(key string, value []byte) {
	s.SetWithTTL(key, value, s.defaultTTL)
}

// SetWithTTL stores value under key with the given TTL. Zero ttl means no expiry.
func (s *Store) SetWithTTL(key string, value []byte, ttl time.Duration) {
	now := time.Now().UnixNano()
	exp := expiresAtFor(ttl, now)
	s.shardFor(key).set(key, value, exp, &s.stats)
}

// Delete removes key and reports whether it existed.
func (s *Store) Delete(key string) (existed bool) {
	return s.shardFor(key).delete(key, &s.stats)
}

// Len returns the number of non-expired keys.
func (s *Store) Len() int {
	now := time.Now().UnixNano()
	n := 0
	for _, sh := range s.shards {
		n += sh.len(now)
	}
	return n
}

// Stats returns a snapshot of store counters.
func (s *Store) Stats() Stats {
	return s.stats.snapshot()
}

// Close stops the background reaper if running.
func (s *Store) Close() {
	s.stopOnce.Do(func() {
		close(s.stop)
		s.wg.Wait()
	})
}

func nextPowerOfTwo(n int) int {
	if n < 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}
