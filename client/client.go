// Package client provides a cluster-aware Storage implementation.
package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"gobase/store"
	"gobase/cluster"
	"gobase/protocol"
)

// Client routes key operations to the correct cluster node.
type Client struct {
	cfg   cluster.Config
	pools []*connPool

	stop     chan struct{}
	stopOnce sync.Once
}

// New creates a cluster client. cfg must pass Validate.
func New(cfg cluster.Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	pools := make([]*connPool, len(cfg.Nodes))
	for i, addr := range cfg.Nodes {
		pools[i] = newConnPool(addr)
	}
	return &Client{
		cfg:   cfg,
		pools: pools,
		stop:  make(chan struct{}),
	}, nil
}

// Get returns the value for key from the owning node.
func (c *Client) Get(key string) (value []byte, ok bool) {
	idx, _ := c.cfg.NodeForKey(key)
	resp, err := c.roundTrip(idx, protocol.Request{Op: protocol.OpGet, Key: key})
	if err != nil {
		return nil, false
	}
	if resp.Status == protocol.StatusNotFound {
		return nil, false
	}
	if resp.Status != protocol.StatusOK {
		return nil, false
	}
	return resp.Value, true
}

// Set stores value under key on the owning node.
func (c *Client) Set(key string, value []byte) {
	c.SetWithTTL(key, value, 0)
}

// SetWithTTL stores value with ttl on the owning node.
func (c *Client) SetWithTTL(key string, value []byte, ttl time.Duration) {
	idx, _ := c.cfg.NodeForKey(key)
	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().UnixNano() + ttl.Nanoseconds()
	}
	req := protocol.Request{
		Op:        protocol.OpSetWithTTL,
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	}
	if ttl == 0 {
		req.Op = protocol.OpSet
	}
	_, _ = c.roundTrip(idx, req)
}

// Delete removes key from the owning node.
func (c *Client) Delete(key string) (existed bool) {
	idx, _ := c.cfg.NodeForKey(key)
	resp, err := c.roundTrip(idx, protocol.Request{Op: protocol.OpDelete, Key: key})
	if err != nil || resp.Status != protocol.StatusOK {
		return false
	}
	return resp.Existed
}

// Len returns the total key count across all nodes.
func (c *Client) Len() int {
	type result struct {
		n   int64
		err error
	}
	ch := make(chan result, len(c.pools))
	var wg sync.WaitGroup
	for i := range c.pools {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := c.roundTrip(idx, protocol.Request{Op: protocol.OpLen})
			if err != nil {
				ch <- result{err: err}
				return
			}
			if resp.Status != protocol.StatusOK {
				ch <- result{err: fmt.Errorf("client: len node %d: status %d", idx, resp.Status)}
				return
			}
			ch <- result{n: resp.Len}
		}(i)
	}
	wg.Wait()
	close(ch)

	total := 0
	for r := range ch {
		if r.err != nil {
			return 0
		}
		total += int(r.n)
	}
	return total
}

// Stats aggregates counters from all nodes.
func (c *Client) Stats() store.Stats {
	type result struct {
		stats store.Stats
		err   error
	}
	ch := make(chan result, len(c.pools))
	var wg sync.WaitGroup
	for i := range c.pools {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := c.roundTrip(idx, protocol.Request{Op: protocol.OpStats})
			if err != nil {
				ch <- result{err: err}
				return
			}
			if resp.Status != protocol.StatusOK {
				ch <- result{err: fmt.Errorf("client: stats node %d: status %d", idx, resp.Status)}
				return
			}
			ch <- result{stats: resp.Stats}
		}(i)
	}
	wg.Wait()
	close(ch)

	var total store.Stats
	for r := range ch {
		if r.err != nil {
			return store.Stats{}
		}
		total = total.Add(r.stats)
	}
	return total
}

// Close closes idle connections.
func (c *Client) Close() {
	c.stopOnce.Do(func() {
		close(c.stop)
		for _, p := range c.pools {
			p.close()
		}
	})
}

func (c *Client) roundTrip(nodeIdx int, req protocol.Request) (protocol.Response, error) {
	conn, err := c.pools[nodeIdx].acquire()
	if err != nil {
		return protocol.Response{}, err
	}
	defer c.pools[nodeIdx].release(conn)

	if err := protocol.WriteRequest(conn, req); err != nil {
		c.pools[nodeIdx].discard(conn)
		return protocol.Response{}, err
	}
	resp, err := protocol.ReadResponse(conn, req.Op)
	if err != nil {
		c.pools[nodeIdx].discard(conn)
		return protocol.Response{}, err
	}
	return resp, nil
}

type connPool struct {
	addr string
	mu   sync.Mutex
	conn net.Conn
}

func newConnPool(addr string) *connPool {
	return &connPool{addr: addr}
}

func (p *connPool) acquire() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		c := p.conn
		p.conn = nil
		return c, nil
	}
	return dial(p.addr)
}

func (p *connPool) release(conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		conn.Close()
		return
	}
	p.conn = conn
}

func (p *connPool) discard(conn net.Conn) {
	conn.Close()
	p.mu.Lock()
	if p.conn == conn {
		p.conn = nil
	}
	p.mu.Unlock()
}

func (p *connPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
}

func dial(addr string) (net.Conn, error) {
	network := "tcp"
	dialAddr := addr
	if len(addr) > 7 && addr[:7] == "unix://" {
		network = "unix"
		dialAddr = addr[7:]
	}
	return net.Dial(network, dialAddr)
}

var _ store.Storage = (*Client)(nil)
