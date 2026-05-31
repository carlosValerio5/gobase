package gobase

import "hash/maphash"

var keyHashSeed = maphash.MakeSeed()

// KeyHash returns a deterministic hash of key used for shard and node routing.
func KeyHash(key string) uint64 {
	var h maphash.Hash
	h.SetSeed(keyHashSeed)
	h.WriteString(key)
	return h.Sum64()
}
