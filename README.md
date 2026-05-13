# DOR - Dynamic Onion Routing

P2P anonymous communication system written in Go.
Fork of a school project (ENSEIRB-MATMECA S8), maintained solo.

## What it does

Messages are encrypted in onion layers (RSA-2048 + AES-256-GCM per hop) and routed through a chain of relay nodes. A TLS directory server keeps the node list. The sender gets an ACK back through a separate return route; on failure, NACK cascades trigger automatic retry.

`SSEND` (super-node mode) encrypts each layer with multiple public keys simultaneously so any node in a group can decrypt - providing redundancy at each hop.

## Components

| Path | Role |
|---|---|
| `node_server/List_Serveur/` | Directory server (TLS, SQLite) |
| `node_server/node/` | Node binary (relay + sender + web UI) |
| `node_server/model/` | Shared types: Node, OnionLayer, AES/RSA crypto, Broadcast |
| `node_server/data/` | SQLite wrapper |
| `docker/` | Docker / Docker Compose setup |
| `scripts/` | Dev and launch helpers |

## Quick start (Docker)

```bash
docker compose up --build
```

Nodes register automatically. The web UI of each node is available on port 9090 (set `ENABLE_WEB=1`).

## Quick start (local)

First generate a TLS certificate for the directory server:

```bash
bash scripts/gen-cert.sh
```

Then:

```bash
# Terminal 1 - directory server
cd node_server/List_Serveur
go run serveur.go

# Terminal 2 - node A
cd node_server/node
NODE_ID=A go run .

# Terminal 3 - node B
cd node_server/node
NODE_ID=B PORT=9001 go run .
```

Or use the tmux launcher: `bash scripts/start_nodes_tmux.sh -n 4`

Inside a node prompt:

```
SEND:<hops>:<ip>:<port>:<message>               # auto route
SSEND:<group>:<hops>:<ip>:<port>:<message>      # super-node mode
LIST:                                            # show registered nodes
REGEN:                                           # rotate RSA key
QUIT:                                            # leave
```

## Environment variables

| Variable | Default | Effect |
|---|---|---|
| `NODE_ID` | - | Node identifier (required) |
| `PORT` | random | TCP port the node listens on |
| `SERVER_ADDR` | `localhost:8080` | Directory server address |
| `NETWORK_PROFILE` | - | `server`, `laptop_WIFI7`, `smartphone_4G`, `smartphone_2G` |
| `ENABLE_WEB` | - | Set to `1` to start the web UI on port 9090 |
| `WEB_PORT` | `9090` | Web UI port |
| `DASHBOARD_URL` | `http://localhost:8888` | Telemetry dashboard endpoint |
| `SIM_LATENCY` | - | Simulate network latency (max ms) |

## Protocol

```
INIT:id:port:pubkey_b64:sa:sn  ->  directory server (register)
GET_LIST                        ->  directory server (fetch peers)
GET_KEY:ip:port                 ->  directory server (fetch pubkey)
QUIT:id                         ->  directory server (unregister)
GET_PUBKEY                      ->  node (direct pubkey fetch)
NACK:msgid                      ->  node (failure propagation)
<encrypted onion>               ->  node (relay or final)
```

Each onion layer: `base64(RSA-OAEP(aesKey)):base64(AES-GCM(payload))`.
Layer fields (pipe-separated): `Type|MsgID|Next|From|Data|Message`.

## Running tests

```bash
cd node_server/model
go test ./...
```

## Security

See [SECURITY.md](SECURITY.md) for threat model and known limitations.
