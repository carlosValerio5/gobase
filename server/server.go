// Package server exposes a Gobase store over TCP or unix domain sockets.
package server

import (
	"fmt"
	"net"
	"strings"
	"time"

	"gobase"
	"gobase/protocol"
)

// ServerConfig configures the network listener.
type ServerConfig struct {
	// Network is "tcp" or "unix". Defaults to "tcp".
	Network string
	// Addr is a host:port for tcp or a filesystem path for unix.
	Addr string
}

// Serve runs a TCP server backed by store until the listener fails.
func Serve(cfg ServerConfig, store *gobase.Store) error {
	return ServeStorage(cfg, store)
}

// ServeStorage runs a network server backed by any Storage implementation.
func ServeStorage(cfg ServerConfig, store gobase.Storage) error {
	network := cfg.Network
	if network == "" {
		network = "tcp"
	}
	ln, err := net.Listen(network, cfg.Addr)
	if err != nil {
		return err
	}
	return ServeStorageOn(ln, store)
}

// ServeStorageOn serves store on an existing listener until Accept fails.
func ServeStorageOn(ln net.Listener, store gobase.Storage) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleConn(conn, store)
	}
}

func handleConn(conn net.Conn, store gobase.Storage) {
	defer conn.Close()
	for {
		req, err := protocol.ReadRequest(conn)
		if err != nil {
			return
		}
		resp := dispatch(store, req)
		if err := protocol.WriteResponse(conn, req.Op, resp); err != nil {
			return
		}
	}
}

func dispatch(store gobase.Storage, req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpGet:
		val, ok := store.Get(req.Key)
		if !ok {
			return protocol.Response{Status: protocol.StatusNotFound}
		}
		return protocol.Response{Status: protocol.StatusOK, Value: val}
	case protocol.OpSet:
		store.Set(req.Key, req.Value)
		return protocol.Response{Status: protocol.StatusOK}
	case protocol.OpSetWithTTL:
		ttl := ttlFromExpiresAt(req.ExpiresAt)
		store.SetWithTTL(req.Key, req.Value, ttl)
		return protocol.Response{Status: protocol.StatusOK}
	case protocol.OpDelete:
		existed := store.Delete(req.Key)
		return protocol.Response{Status: protocol.StatusOK, Existed: existed}
	case protocol.OpLen:
		return protocol.Response{Status: protocol.StatusOK, Len: int64(store.Len())}
	case protocol.OpStats:
		return protocol.Response{Status: protocol.StatusOK, Stats: store.Stats()}
	case protocol.OpPing:
		return protocol.Response{Status: protocol.StatusOK}
	default:
		return protocol.Response{Status: protocol.StatusError, ErrMsg: fmt.Sprintf("unknown op %d", req.Op)}
	}
}

func ttlFromExpiresAt(expiresAt int64) time.Duration {
	if expiresAt == 0 {
		return 0
	}
	now := time.Now().UnixNano()
	if expiresAt <= now {
		return 0
	}
	return time.Duration(expiresAt - now)
}

// ParseAddr splits network and address from "host:port" or "unix:///path".
func ParseAddr(raw string) (network, addr string, err error) {
	if strings.HasPrefix(raw, "unix://") {
		return "unix", strings.TrimPrefix(raw, "unix://"), nil
	}
	return "tcp", raw, nil
}
