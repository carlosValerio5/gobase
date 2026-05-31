package protocol_test

import (
	"bytes"
	"testing"

	"gobase/store"
	"gobase/protocol"
)

func TestRoundTripGet(t *testing.T) {
	var buf bytes.Buffer
	req := protocol.Request{Op: protocol.OpGet, Key: "foo"}
	if err := protocol.WriteRequest(&buf, req); err != nil {
		t.Fatal(err)
	}
	got, err := protocol.ReadRequest(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "foo" || got.Op != protocol.OpGet {
		t.Fatalf("ReadRequest() = %+v", got)
	}

	buf.Reset()
	resp := protocol.Response{Status: protocol.StatusOK, Value: []byte("bar")}
	if err := protocol.WriteResponse(&buf, protocol.OpGet, resp); err != nil {
		t.Fatal(err)
	}
	gotResp, err := protocol.ReadResponse(&buf, protocol.OpGet)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotResp.Value) != "bar" {
		t.Fatalf("ReadResponse() value = %q", gotResp.Value)
	}
}

func TestRoundTripSetWithTTL(t *testing.T) {
	var buf bytes.Buffer
	req := protocol.Request{
		Op:        protocol.OpSetWithTTL,
		Key:       "k",
		Value:     []byte("v"),
		ExpiresAt: 12345,
	}
	if err := protocol.WriteRequest(&buf, req); err != nil {
		t.Fatal(err)
	}
	got, err := protocol.ReadRequest(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExpiresAt != 12345 {
		t.Fatalf("ExpiresAt = %d", got.ExpiresAt)
	}
}

func TestRoundTripStats(t *testing.T) {
	var buf bytes.Buffer
	resp := protocol.Response{
		Status: protocol.StatusOK,
		Stats:  store.Stats{Hits: 1, Misses: 2, Sets: 3, Deletes: 4, Expired: 5},
	}
	if err := protocol.WriteResponse(&buf, protocol.OpStats, resp); err != nil {
		t.Fatal(err)
	}
	got, err := protocol.ReadResponse(&buf, protocol.OpStats)
	if err != nil {
		t.Fatal(err)
	}
	if got.Stats != resp.Stats {
		t.Fatalf("Stats = %+v, want %+v", got.Stats, resp.Stats)
	}
}
