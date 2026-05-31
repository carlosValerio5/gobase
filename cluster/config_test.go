package cluster_test

import (
	"testing"

	"gobase/cluster"
)

func TestNodeForKey(t *testing.T) {
	cfg := cluster.Config{Nodes: []string{"a:1", "b:2", "c:3", "d:4"}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	idx1, addr1 := cfg.NodeForKey("user:42")
	idx2, addr2 := cfg.NodeForKey("user:42")
	if idx1 != idx2 || addr1 != addr2 {
		t.Fatalf("routing not deterministic: (%d,%s) vs (%d,%s)", idx1, addr1, idx2, addr2)
	}
	if addr1 != cfg.Nodes[idx1] {
		t.Fatalf("addr = %q, want %q", addr1, cfg.Nodes[idx1])
	}
}

func TestValidatePowerOfTwo(t *testing.T) {
	if err := (cluster.Config{Nodes: []string{"a"}}).Validate(); err != nil {
		t.Fatalf("single node: %v", err)
	}
	if err := (cluster.Config{Nodes: []string{"a", "b", "c"}}).Validate(); err == nil {
		t.Fatal("expected error for non-power-of-two node count")
	}
}
