package store

import "sync"

type shard struct {
	mu   sync.RWMutex
	data map[string]entry
}

func newShard() *shard {
	return &shard{data: make(map[string]entry)}
}

func (s *shard) get(key string, now int64, stats *statCounters) (value []byte, ok bool) {
	s.mu.RLock()
	e, found := s.data[key]
	s.mu.RUnlock()
	if !found {
		stats.misses.Add(1)
		return nil, false
	}
	if isExpired(e, now) {
		stats.misses.Add(1)
		stats.expired.Add(1)
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return nil, false
	}
	stats.hits.Add(1)
	return e.value, true
}

func (s *shard) set(key string, value []byte, expiresAt int64, stats *statCounters) {
	val := append([]byte(nil), value...)
	s.mu.Lock()
	s.data[key] = entry{value: val, expiresAt: expiresAt}
	s.mu.Unlock()
	stats.sets.Add(1)
}

func (s *shard) delete(key string, stats *statCounters) (existed bool) {
	s.mu.Lock()
	_, existed = s.data[key]
	if existed {
		delete(s.data, key)
		stats.deletes.Add(1)
	}
	s.mu.Unlock()
	return existed
}

func (s *shard) len(now int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, e := range s.data {
		if !isExpired(e, now) {
			n++
		}
	}
	return n
}

func (s *shard) sweep(now int64, stats *statCounters) {
	s.mu.Lock()
	for key, e := range s.data {
		if isExpired(e, now) {
			delete(s.data, key)
			stats.expired.Add(1)
		}
	}
	s.mu.Unlock()
}
