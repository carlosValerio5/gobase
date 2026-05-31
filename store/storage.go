package store

import "time"

// Storage is the key-value store contract shared by local and cluster clients.
type Storage interface {
	Get(key string) (value []byte, ok bool)
	Set(key string, value []byte)
	SetWithTTL(key string, value []byte, ttl time.Duration)
	Delete(key string) (existed bool)
	Len() int
	Stats() Stats
	Close()
}
