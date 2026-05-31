# Gobase

In-memory key-value store for unix systems, optimized for low-latency concurrent access.

See [docs/structure.md](docs/structure.md) for the repository layout and package roles.

## v1: Single-node usage

```go
package main

import (
    "fmt"
    "time"

    "gobase/store"
)

func main() {
    db := store.New(
        store.WithShards(256),
        store.WithDefaultTTL(time.Hour),
    )
    defer db.Close()

    db.Set("user:1", []byte("alice"))
    db.SetWithTTL("session:abc", []byte("token"), 30*time.Minute)

    if val, ok := db.Get("user:1"); ok {
        fmt.Println(string(val))
    }

    fmt.Println(db.Stats())
}
```

Depend on the `Storage` interface so you can swap implementations later:

```go
var db store.Storage = store.New()
```

## Design

- Sharded in-process maps with per-shard `RWMutex`
- Shared `KeyHash` for deterministic routing (local shards and cluster nodes)
- Lazy TTL expiry on read plus optional background reaper
- Zero-copy reads: do not mutate slices returned by `Get`

## v2: Distributed store

Partitioned multi-node operation (no replication, no persistence):

| Package | Role |
|---------|------|
| `gobase/store` | Local engine on each node |
| `gobase/cluster` | Static node config and key-to-node routing |
| `gobase/protocol` | Binary wire codec |
| `gobase/server` | TCP server wrapping a local `Store` |
| `gobase/client` | Cluster client implementing `store.Storage` |

```go
import (
    "gobase/client"
    "gobase/cluster"
    "gobase/store"
)

var db store.Storage
db, _ = client.New(cluster.Config{
    Nodes: []string{"10.0.0.1:7400", "10.0.0.2:7400"},
})
```

Node count must be a power of two. Each key lives on exactly one node; node failure loses that partition's data.

## Requirements

- Go 1.22+
- Unix (linux, darwin)

## Tests

```bash
go test ./...
go test -race ./...
go test -bench=. ./store/...
```
