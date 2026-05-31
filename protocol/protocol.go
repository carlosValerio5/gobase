// Package protocol defines the binary wire format for Gobase cluster communication.
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"

	"gobase/store"
)

const Version byte = 1

// Op identifies a request type.
type Op byte

const (
	OpGet        Op = 1
	OpSet        Op = 2
	OpSetWithTTL Op = 3
	OpDelete     Op = 4
	OpLen        Op = 5
	OpStats      Op = 6
	OpPing       Op = 7
)

// Status identifies a response outcome.
type Status byte

const (
	StatusOK        Status = 0
	StatusNotFound  Status = 1
	StatusWrongNode Status = 2
	StatusError     Status = 3
)

// Request is a decoded wire request.
type Request struct {
	Op        Op
	Key       string
	Value     []byte
	ExpiresAt int64
}

// Response is a decoded wire response.
type Response struct {
	Status    Status
	Value     []byte
	Existed   bool
	Len       int64
	Stats     store.Stats
	NodeIndex uint16
	ErrMsg    string
}

// WriteRequest encodes and writes req to w.
func WriteRequest(w io.Writer, req Request) error {
	if err := binary.Write(w, binary.BigEndian, Version); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, req.Op); err != nil {
		return err
	}

	switch req.Op {
	case OpGet, OpDelete:
		return writeKey(w, req.Key)
	case OpSet, OpSetWithTTL:
		if err := writeKey(w, req.Key); err != nil {
			return err
		}
		if err := writeBytes(w, req.Value); err != nil {
			return err
		}
		if req.Op == OpSetWithTTL {
			return binary.Write(w, binary.BigEndian, req.ExpiresAt)
		}
		return nil
	case OpLen, OpStats, OpPing:
		return nil
	default:
		return fmt.Errorf("protocol: unknown op %d", req.Op)
	}
}

// ReadRequest decodes a request from r.
func ReadRequest(r io.Reader) (Request, error) {
	var ver byte
	if err := binary.Read(r, binary.BigEndian, &ver); err != nil {
		return Request{}, err
	}
	if ver != Version {
		return Request{}, fmt.Errorf("protocol: unsupported version %d", ver)
	}

	var op Op
	if err := binary.Read(r, binary.BigEndian, &op); err != nil {
		return Request{}, err
	}

	req := Request{Op: op}
	switch op {
	case OpGet, OpDelete:
		key, err := readKey(r)
		if err != nil {
			return Request{}, err
		}
		req.Key = key
	case OpSet, OpSetWithTTL:
		key, err := readKey(r)
		if err != nil {
			return Request{}, err
		}
		req.Key = key
		val, err := readBytes(r)
		if err != nil {
			return Request{}, err
		}
		req.Value = val
		if op == OpSetWithTTL {
			if err := binary.Read(r, binary.BigEndian, &req.ExpiresAt); err != nil {
				return Request{}, err
			}
		}
	case OpLen, OpStats, OpPing:
	default:
		return Request{}, fmt.Errorf("protocol: unknown op %d", op)
	}
	return req, nil
}

// WriteResponse encodes and writes resp to w for the given request op.
func WriteResponse(w io.Writer, op Op, resp Response) error {
	if err := binary.Write(w, binary.BigEndian, resp.Status); err != nil {
		return err
	}

	switch resp.Status {
	case StatusOK:
		switch op {
		case OpGet:
			return writeBytes(w, resp.Value)
		case OpDelete:
			var b byte
			if resp.Existed {
				b = 1
			}
			return binary.Write(w, binary.BigEndian, b)
		case OpLen:
			return binary.Write(w, binary.BigEndian, resp.Len)
		case OpStats:
			return writeStats(w, resp.Stats)
		case OpSet, OpSetWithTTL, OpPing:
			return nil
		default:
			return fmt.Errorf("protocol: unknown op %d", op)
		}
	case StatusNotFound:
		return nil
	case StatusWrongNode:
		return binary.Write(w, binary.BigEndian, resp.NodeIndex)
	case StatusError:
		return writeString(w, resp.ErrMsg)
	default:
		return fmt.Errorf("protocol: unknown status %d", resp.Status)
	}
}

// ReadResponse decodes a response from r for the given request op.
func ReadResponse(r io.Reader, op Op) (Response, error) {
	var status Status
	if err := binary.Read(r, binary.BigEndian, &status); err != nil {
		return Response{}, err
	}
	resp := Response{Status: status}

	switch status {
	case StatusOK:
		switch op {
		case OpGet:
			val, err := readBytes(r)
			if err != nil {
				return Response{}, err
			}
			resp.Value = val
		case OpDelete:
			var b byte
			if err := binary.Read(r, binary.BigEndian, &b); err != nil {
				return Response{}, err
			}
			resp.Existed = b == 1
		case OpLen:
			if err := binary.Read(r, binary.BigEndian, &resp.Len); err != nil {
				return Response{}, err
			}
		case OpStats:
			stats, err := readStats(r)
			if err != nil {
				return Response{}, err
			}
			resp.Stats = stats
		case OpSet, OpSetWithTTL, OpPing:
			// no payload
		}
	case StatusNotFound:
	case StatusWrongNode:
		if err := binary.Read(r, binary.BigEndian, &resp.NodeIndex); err != nil {
			return Response{}, err
		}
	case StatusError:
		msg, err := readString(r)
		if err != nil {
			return Response{}, err
		}
		resp.ErrMsg = msg
	default:
		return Response{}, fmt.Errorf("protocol: unknown status %d", status)
	}
	return resp, nil
}

func writeKey(w io.Writer, key string) error {
	if len(key) > 0xffff {
		return fmt.Errorf("protocol: key too long")
	}
	if err := binary.Write(w, binary.BigEndian, uint16(len(key))); err != nil {
		return err
	}
	_, err := io.WriteString(w, key)
	return err
}

func readKey(r io.Reader) (string, error) {
	var n uint16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return "", err
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func writeBytes(w io.Writer, b []byte) error {
	if err := binary.Write(w, binary.BigEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func readBytes(r io.Reader) ([]byte, error) {
	var n uint32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeString(w io.Writer, s string) error {
	if len(s) > 0xffff {
		return fmt.Errorf("protocol: error message too long")
	}
	if err := binary.Write(w, binary.BigEndian, uint16(len(s))); err != nil {
		return err
	}
	_, err := io.WriteString(w, s)
	return err
}

func readString(r io.Reader) (string, error) {
	var n uint16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return "", err
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func writeStats(w io.Writer, s store.Stats) error {
	fields := []int64{s.Hits, s.Misses, s.Sets, s.Deletes, s.Expired}
	for _, f := range fields {
		if err := binary.Write(w, binary.BigEndian, f); err != nil {
			return err
		}
	}
	return nil
}

func readStats(r io.Reader) (store.Stats, error) {
	var s store.Stats
	fields := []*int64{&s.Hits, &s.Misses, &s.Sets, &s.Deletes, &s.Expired}
	for _, f := range fields {
		if err := binary.Read(r, binary.BigEndian, f); err != nil {
			return store.Stats{}, err
		}
	}
	return s, nil
}
