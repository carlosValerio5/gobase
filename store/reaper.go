package store

import "time"

func (s *Store) reapLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.reaperEvery)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.sweepExpired()
		}
	}
}

func (s *Store) sweepExpired() {
	now := time.Now().UnixNano()
	for _, sh := range s.shards {
		sh.sweep(now, &s.stats)
	}
}
