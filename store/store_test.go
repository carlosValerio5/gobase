package store_test

import (
	"sync"
	"testing"
	"time"

	"gobase/store"
)

func TestGetSetDelete(t *testing.T) {
	s := store.New(store.WithReaperInterval(0))
	defer s.Close()

	s.Set("foo", []byte("bar"))
	val, ok := s.Get("foo")
	if !ok || string(val) != "bar" {
		t.Fatalf("Get() = (%q, %v), want (bar, true)", val, ok)
	}

	if existed := s.Delete("foo"); !existed {
		t.Fatal("Delete() existed = false, want true")
	}
	if _, ok := s.Get("foo"); ok {
		t.Fatal("Get() after delete = ok, want false")
	}
	if existed := s.Delete("foo"); existed {
		t.Fatal("Delete() missing key existed = true, want false")
	}
}

func TestTTLExpiry(t *testing.T) {
	s := store.New(store.WithReaperInterval(0))
	defer s.Close()

	s.SetWithTTL("temp", []byte("x"), 20*time.Millisecond)
	if _, ok := s.Get("temp"); !ok {
		t.Fatal("Get() before expiry = false, want true")
	}

	time.Sleep(30 * time.Millisecond)
	if _, ok := s.Get("temp"); ok {
		t.Fatal("Get() after expiry = true, want false")
	}

	stats := s.Stats()
	if stats.Expired == 0 {
		t.Fatal("Stats().Expired = 0, want > 0")
	}
}

func TestLen(t *testing.T) {
	s := store.New(store.WithReaperInterval(0))
	defer s.Close()

	s.Set("a", []byte("1"))
	s.Set("b", []byte("2"))
	s.SetWithTTL("c", []byte("3"), time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	if n := s.Len(); n != 2 {
		t.Fatalf("Len() = %d, want 2", n)
	}
}

func TestStats(t *testing.T) {
	s := store.New(store.WithReaperInterval(0))
	defer s.Close()

	s.Set("k", []byte("v"))
	s.Get("k")
	s.Get("missing")
	s.Delete("k")

	stats := s.Stats()
	if stats.Hits != 1 || stats.Misses != 1 || stats.Sets != 1 || stats.Deletes != 1 {
		t.Fatalf("Stats() = %+v, unexpected counts", stats)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := store.New(store.WithReaperInterval(0))
	defer s.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			s.Set(key, []byte("v"))
			s.Get(key)
			s.Delete(key)
		}(i)
	}
	wg.Wait()
}

func TestStorageInterface(t *testing.T) {
	var _ store.Storage = store.New(store.WithReaperInterval(0))
}
