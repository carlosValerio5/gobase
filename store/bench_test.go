package store_test

import (
	"fmt"
	"testing"

	"gobase/store"
)

func BenchmarkSet(b *testing.B) {
	s := store.New(store.WithReaperInterval(0))
	defer s.Close()
	val := []byte("value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Set(fmt.Sprintf("key-%d", i), val)
			i++
		}
	})
}

func BenchmarkGet(b *testing.B) {
	s := store.New(store.WithReaperInterval(0))
	defer s.Close()
	for i := 0; i < 10000; i++ {
		s.Set(fmt.Sprintf("key-%d", i), []byte("value"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Get(fmt.Sprintf("key-%d", i%10000))
			i++
		}
	})
}
