package gobase

import "time"

// entry holds a stored value and optional expiration time.
type entry struct {
	value     []byte
	expiresAt int64 // unix nanoseconds; 0 means no expiry
}

// isExpired reports whether e has expired at now.
func isExpired(e entry, now int64) bool {
	return e.expiresAt != 0 && now >= e.expiresAt
}

// expiresAtFor returns the unix-nano expiry for ttl; 0 ttl means no expiry.
func expiresAtFor(ttl time.Duration, now int64) int64 {
	if ttl <= 0 {
		return 0
	}
	return now + ttl.Nanoseconds()
}
