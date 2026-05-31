# Repository structure

Gobase separates the single-node engine, distributed packages, and tests so the repo root stays minimal.

```
gobase/
├── go.mod                 # module gobase
├── README.md              # overview and quick start
├── docs/
│   └── structure.md       # this file
├── store/                 # v1: single-node in-memory engine
│   ├── store.go           # Store API (Get, Set, Delete, …)
│   ├── shard.go           # per-shard maps and locks
│   ├── hash.go            # KeyHash — shared routing primitive
│   ├── storage.go         # Storage interface
│   ├── entry.go           # value + TTL record
│   ├── stats.go           # counters
│   ├── options.go         # functional options
│   ├── reaper.go          # background TTL sweeper
│   └── *_test.go          # unit tests and benchmarks
├── cluster/               # static cluster config and key → node routing
├── protocol/              # binary wire codec for node RPC
├── server/                # TCP/unix server wrapping a local Store
├── client/                # cluster client implementing store.Storage
└── test/
    └── integration/       # multi-node cluster tests
```

## Packages

| Import path | Role |
|-------------|------|
| `gobase/store` | Local sharded KV engine; implement or depend on `store.Storage` |
| `gobase/cluster` | Node list, validation, `NodeForKey` |
| `gobase/protocol` | Request/response encoding (used by server and client) |
| `gobase/server` | Run a node: `Serve` / `ServeStorage` |
| `gobase/client` | Cluster-wide `Storage` with client-side routing |

## Data flow (cluster)

1. Application uses `client.Client` as `store.Storage`.
2. Client calls `cluster.Config.NodeForKey` → `store.KeyHash(key) & nodeMask`.
3. Client sends a `protocol` frame to that node's `server`.
4. Server dispatches to a local `store.Store`.

## Tests

- **Unit / bench:** `go test ./store/... ./cluster/... ./protocol/...`
- **Integration:** `go test ./test/integration/...`
- **All:** `go test ./...`
