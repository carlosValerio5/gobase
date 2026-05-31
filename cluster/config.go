// Package cluster provides static cluster configuration and key-to-node routing.
package cluster

import (
	"fmt"

	"gobase"
)

// Config holds a fixed list of node addresses. Len(Nodes) must be a power of two.
type Config struct {
	Nodes []string
}

// Validate checks that the config is usable for routing.
func (c Config) Validate() error {
	if len(c.Nodes) == 0 {
		return fmt.Errorf("cluster: at least one node required")
	}
	if len(c.Nodes)&(len(c.Nodes)-1) != 0 {
		return fmt.Errorf("cluster: node count %d must be a power of two", len(c.Nodes))
	}
	return nil
}

// NodeMask returns the bitmask used for node index selection.
func (c Config) NodeMask() uint64 {
	return uint64(len(c.Nodes) - 1)
}

// NodeForKey returns the index and address of the node that owns key.
func (c Config) NodeForKey(key string) (index int, addr string) {
	idx := int(KeyHash(key) & c.NodeMask())
	return idx, c.Nodes[idx]
}

// KeyHash re-exports gobase.KeyHash for routing consistency.
func KeyHash(key string) uint64 {
	return gobase.KeyHash(key)
}
