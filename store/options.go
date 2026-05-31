package store

import "time"

type config struct {
	shardCount     int
	defaultTTL     time.Duration
	reaperInterval time.Duration
}

// Option configures a Store.
type Option func(*config)

// WithShards sets the number of shards to the next power of two of n.
func WithShards(n int) Option {
	return func(c *config) {
		c.shardCount = n
	}
}

// WithDefaultTTL sets the TTL applied by Set when no explicit TTL is given.
func WithDefaultTTL(d time.Duration) Option {
	return func(c *config) {
		c.defaultTTL = d
	}
}

// WithReaperInterval sets how often expired keys are swept. Zero disables the reaper.
func WithReaperInterval(d time.Duration) Option {
	return func(c *config) {
		c.reaperInterval = d
	}
}
