package store

import "sync/atomic"

// Stats holds aggregated store counters.
type Stats struct {
	Hits    int64
	Misses  int64
	Sets    int64
	Deletes int64
	Expired int64
}

type statCounters struct {
	hits    atomic.Int64
	misses  atomic.Int64
	sets    atomic.Int64
	deletes atomic.Int64
	expired atomic.Int64
}

func (c *statCounters) snapshot() Stats {
	return Stats{
		Hits:    c.hits.Load(),
		Misses:  c.misses.Load(),
		Sets:    c.sets.Load(),
		Deletes: c.deletes.Load(),
		Expired: c.expired.Load(),
	}
}

// Add combines other into s.
func (s Stats) Add(other Stats) Stats {
	return Stats{
		Hits:    s.Hits + other.Hits,
		Misses:  s.Misses + other.Misses,
		Sets:    s.Sets + other.Sets,
		Deletes: s.Deletes + other.Deletes,
		Expired: s.Expired + other.Expired,
	}
}
