package integration_test

import (
	"fmt"
	"net"
	"sync"
	"testing"

	"gobase/client"
	"gobase/cluster"
	"gobase/server"
	"gobase/store"
)

type testCluster struct {
	addrs  []string
	stores []*store.Store
	ln     []net.Listener
}

func startCluster(t *testing.T, n int) *testCluster {
	t.Helper()
	if n&(n-1) != 0 {
		t.Fatalf("node count %d must be power of two", n)
	}

	tc := &testCluster{
		addrs:  make([]string, n),
		stores: make([]*store.Store, n),
		ln:     make([]net.Listener, n),
	}

	for i := 0; i < n; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		tc.ln[i] = ln
		tc.addrs[i] = ln.Addr().String()
		tc.stores[i] = store.New(store.WithReaperInterval(0))

		go func(l net.Listener, s store.Storage) {
			_ = server.ServeStorageOn(l, s)
		}(ln, tc.stores[i])
	}
	return tc
}

func (tc *testCluster) close(t *testing.T) {
	t.Helper()
	for _, ln := range tc.ln {
		ln.Close()
	}
	for _, s := range tc.stores {
		s.Close()
	}
}

func TestClusterRouting(t *testing.T) {
	tc := startCluster(t, 4)
	defer tc.close(t)

	cfg := cluster.Config{Nodes: tc.addrs}
	cl, err := client.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()

	key := "routing-key"
	val := []byte("cluster-value")
	cl.Set(key, val)

	idx, _ := cfg.NodeForKey(key)
	got, ok := tc.stores[idx].Get(key)
	if !ok || string(got) != string(val) {
		t.Fatalf("key not on expected node %d: (%q, %v)", idx, got, ok)
	}

	for i, s := range tc.stores {
		if i == idx {
			continue
		}
		if _, ok := s.Get(key); ok {
			t.Fatalf("key incorrectly present on node %d", i)
		}
	}

	clientVal, ok := cl.Get(key)
	if !ok || string(clientVal) != string(val) {
		t.Fatalf("client Get() = (%q, %v)", clientVal, ok)
	}
}

func TestClusterLenAndStats(t *testing.T) {
	tc := startCluster(t, 2)
	defer tc.close(t)

	cl, err := client.New(cluster.Config{Nodes: tc.addrs})
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()

	cl.Set("a", []byte("1"))
	cl.Set("b", []byte("2"))

	if n := cl.Len(); n != 2 {
		t.Fatalf("Len() = %d, want 2", n)
	}
	stats := cl.Stats()
	if stats.Sets < 2 {
		t.Fatalf("Stats().Sets = %d, want >= 2", stats.Sets)
	}
}

func TestClusterNodeFailure(t *testing.T) {
	tc := startCluster(t, 2)
	defer tc.close(t)

	cfg := cluster.Config{Nodes: tc.addrs}
	cl, err := client.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	failKey := ""
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("probe-%d", i)
		if idx, _ := cfg.NodeForKey(key); idx == 0 {
			failKey = key
			break
		}
	}
	if failKey == "" {
		t.Fatal("could not find key routed to node 0")
	}

	cl.Set(failKey, []byte("gone"))
	cl.Close()
	tc.ln[0].Close()

	cl2, err := client.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cl2.Close()

	if _, ok := cl2.Get(failKey); ok {
		t.Fatal("Get() succeeded after node failure, want false")
	}

	liveKey := ""
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("live-%d", i)
		if idx, _ := cfg.NodeForKey(key); idx == 1 {
			liveKey = key
			break
		}
	}
	if liveKey == "" {
		t.Fatal("could not find key routed to node 1")
	}

	cl2.Set(liveKey, []byte("ok"))
	if _, ok := cl2.Get(liveKey); !ok {
		t.Fatal("Get() on live node failed")
	}
}

func TestClusterConcurrent(t *testing.T) {
	tc := startCluster(t, 4)
	defer tc.close(t)

	cl, err := client.New(cluster.Config{Nodes: tc.addrs})
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", n)
			cl.Set(key, []byte("v"))
			cl.Get(key)
		}(i)
	}
	wg.Wait()
}

func TestClientImplementsStorage(t *testing.T) {
	var _ store.Storage = (*client.Client)(nil)
}
