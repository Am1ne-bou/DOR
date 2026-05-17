<!-- TODO: add demo GIF showing a message routed through 3 nodes with ACK received -->

# DOR -- Dynamic Onion Routing

[![CI](https://github.com/Am1ne-bou/DOR/actions/workflows/ci.yml/badge.svg)](https://github.com/Am1ne-bou/DOR/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go)

P2P anonymous communication system in Go. Every participant acts simultaneously as a client and a
relay. Messages are wrapped in layered RSA-2048 + AES-256-GCM encryption; no relay ever learns
both the sender and the recipient.

Fork of a 7-person school project (ENSEIRB-MATMECA S8), maintained solo.

---

## Two routing modes

### SEND -- Onion routing with NACK/Retry

Each hop is a single relay. If a relay cannot reach the next hop, it sends a NACK backward through
the chain using pre-generated per-hop identifiers. The sender detects the failure and retries with a
fresh path, capped by `maxRetries`.

NACK identifiers are rewritten at every hop so an observer cannot correlate failures across links --
the same anonymity invariant as the forward direction.

```
Sender --> R1 --> R2 --> R3 --> R4 (unreachable)
                              NACK:n3
                   NACK:n2 <--
          NACK:n1 <--
Sender <-- (retries with new route)
```

### SSEND -- Supernode / Adaptive Anycast Routing

Each hop is a cluster of M nodes. The sender encrypts the AES session key once per cluster member
(broadcast encryption). Any reachable member can decrypt and forward -- a single node going down
does not break the circuit.

```
wire: base64(K1);base64(K2);...;base64(KM):base64(AES-GCM payload)
```

Clusters are built client-side by `BuildSmartClusters`: nodes are scored by availability (Sa) and
network capacity (Sn), distributed across hops, then shuffled within each cluster to prevent
deterministic path prediction. The directory server stays a neutral peer registry.

---

## Architecture

```
+----------------------------------------------------------+
|         DIRECTORY SERVER (:8080 TLS)                     |
|  INIT / GET_LIST / GET_KEY / UPDATE_KEY / QUIT           |
|  SQLite: uuid | name | ip | port | pubkey | Sa | Sn      |
|  TestPing: every 10s, evicts unreachable nodes           |
+---^----------^----------^-------------------------------+
    | TLS      | TLS      | TLS
+---+-------+ +-+-------+ +-+--------+
| NODE A    | | NODE B  | | NODE C  |  ... (N nodes)
| :9000     | | :9001   | | :9002   |
+---+-------+ +-+-------+ +-+--------+
    |          |          |
    +----------+----------+  TCP point-to-point (onion msgs)

Each node: :9090 HTTP web UI  (ENABLE_WEB=1)
Dashboard: :8888 real-time routing graph
```

---

## Onion encryption (per hop)

```
plaintext payload
    |
    v
AES-256-GCM  (random 32-byte key, random nonce)
    |
    v
RSA-OAEP 2048  (target node public key)
    |
    v
SEND:  base64(enc_aes_key):base64(ciphertext)
SSEND: base64(K1);...;base64(KM):base64(ciphertext)
```

Layers are built innermost-first: FINAL (recipient) -> RELAY (hop N-1) -> ... -> RELAY (hop 1).
The ACK return onion is pre-built by the sender and embedded in the FINAL layer's Data field.
The recipient forwards it back without decrypting it -- the sender's address is never exposed.

---

## What I built vs what was inherited

**Inherited:**

- Basic TCP node communication loop
- Directory server skeleton
- Initial SQLite schema

**Solo additions and fixes (post-fork):**

- Full node architecture: goroutine layout, INIT handshake, CLI loop (`Node.go`, `main.go`)
- ACK/NACK/Retry: `PendingACKs`, `PendingRelays`, `SendWithRetry`, backward NACK propagation
- Onion layer format: seven-field pipe-delimited record, double-onion (forward + embedded return ACK)
- `BuildSmartClusters` + `BroadcastEncrypt`/`BroadcastDecrypt` (supernode implementation)
- `KeyCache` with `sync.RWMutex` -- fixed data race on public key map (TTL-based caching)
- NACK cascade cleanup -- `PendingRelays` map leak fixed
- MsgID space: 10^10 vs original 10^6 (birthday collision threshold ~140k vs ~1k concurrent msgs)
- Web UI locked to `127.0.0.1` (was `0.0.0.0`)
- `conn.Write` checks with early return on network failure (keycache, GetNodesList, GET_PUBKEY handler)
- `KeyCache.cleanCache()` -- background eviction goroutine, prevents unbounded growth on long-running nodes
- Message fragmentation: auto-split at 4096B, each chunk sent as an independent onion, reassembled in order at the FINAL node
- Replay/loop protection: per-node seen-MsgID set with 30s TTL eviction, duplicate RELAY/FINAL layers dropped
- GitHub Actions CI with race detector
- Integration tests (SQLite roundtrip) and unit tests (AES, onion parsing, broadcast enc/dec)
- End-to-end tests: `TestSendACKRoundtrip`, `TestSSendACKRoundtrip`, `TestSendFragmented` -- in-process, no Docker needed

---

## Components

| Path                        | Role                                                      |
|-----------------------------|-----------------------------------------------------------|
| `node_server/List_Serveur/` | Directory server (TLS, SQLite)                            |
| `node_server/node/`         | Node binary (relay + sender + web UI)                     |
| `node_server/model/`        | Shared types: Node, OnionLayer, AES/RSA crypto, Broadcast |
| `node_server/data/`         | SQLite wrapper                                            |
| `docker/`                   | Docker / Docker Compose setup                             |
| `scripts/`                  | Dev and launch helpers                                    |

---

## Quick start (Docker)

```bash
docker compose up --build
```

Nodes register automatically. Web UI available on port 9090 (`ENABLE_WEB=1`).

## Quick start (local)

```bash
bash scripts/gen-cert.sh        # generate TLS cert (required once)

# Terminal 1 - directory server
cd node_server/List_Serveur && go run serveur.go

# Terminal 2 - node A
cd node_server/node && NODE_ID=A go run .

# Terminal 3 - node B
cd node_server/node && NODE_ID=B PORT=9001 go run .
```

Or: `bash scripts/start_nodes_tmux.sh -n 4`

Inside a node prompt:

```
SEND:<hops>:<ip>:<port>:<message>               # onion routing with NACK/retry
SSEND:<group>:<hops>:<ip>:<port>:<message>      # supernode anycast mode
LIST:                                            # show registered nodes
REGEN:                                           # rotate RSA key
BENCH:<n>:<hops>:<ip>:<port>                    # latency benchmark
QUIT:                                           # leave
```

---

## Environment variables

| Variable          | Default                 | Effect                                                     |
|-------------------|-------------------------|------------------------------------------------------------|
| `NODE_ID`         | -                       | Node identifier (required)                                 |
| `PORT`            | random                  | TCP listen port                                            |
| `SERVER_ADDR`     | `localhost:8080`        | Directory server address                                   |
| `NETWORK_PROFILE` | -                       | `server`, `laptop_WIFI7`, `smartphone_4G`, `smartphone_2G` |
| `ENABLE_WEB`      | -                       | Set to `1` to enable web UI on port 9090                   |
| `WEB_PORT`        | `9090`                  | Web UI port                                                |
| `DASHBOARD_URL`   | `http://localhost:8888` | Telemetry dashboard endpoint                               |
| `SIM_LATENCY`     | -                       | Simulate network latency (max ms)                          |

---

## Protocol (wire format)

```
INIT:id:port:pubkey_b64:sa:sn  ->  directory server (register)
GET_LIST                        ->  directory server (fetch peers)
GET_KEY:ip:port                 ->  directory server (fetch pubkey)
QUIT:id                         ->  directory server (unregister)
GET_PUBKEY                      ->  node (direct pubkey fetch)
NACK:msgid                      ->  node (failure propagation)
<encrypted onion>               ->  node (relay or final)
```

Layer fields (pipe-separated): `Type | MsgID | Next | From | Data | Frag | Message`
Types: `RELAY` (forward hop), `FINAL` (destination), `ACK` (return path)
`Frag` format: `fragID:i/n` (chunk index and total), empty for non-fragmented messages

---

## Running tests

```bash
go test -race ./...                                        # all packages, race detector
go test ./node_server/node/ -run "TestSend|TestSSend|TestFrag" -v  # e2e: full 3-hop onion path, no docker
```

---

## Security

See [SECURITY.md](SECURITY.md) for threat model and known limitations.

---

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for contribution breakdown.
