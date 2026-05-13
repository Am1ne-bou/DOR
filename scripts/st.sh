#!/bin/bash
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
i=${1:-1} ENABLE_WEB="1" NODE_ID="node${i}" WEB_PORT="959${i}" SERVER_ADDR=localhost:8080 go run "$ROOT/node_server/node/"
