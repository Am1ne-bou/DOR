#!/bin/bash
# Integration test: 3-node onion routing (A -> B -> C, check ACK + message delivery)
# Run from anywhere: bash node_server/tests/dor_test.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR/.."

CERTDIR="$ROOT/List_Serveur"
LOG="$SCRIPT_DIR/logs/dor_test"
NODEBIN=/tmp/dor-node
DIRBIN=/tmp/dor-dir

rm -rf "$LOG" && mkdir -p "$LOG"

cleanup() {
    kill $DIRPID $NODEBPID $NODECPID 2>/dev/null
    wait 2>/dev/null
}
trap cleanup EXIT

echo "=== [build] compiling binaries ==="
go build -o "$NODEBIN" "$ROOT/node/" || { echo "FAIL: node build"; exit 1; }
go build -o "$DIRBIN" "$ROOT/List_Serveur/" || { echo "FAIL: dir build"; exit 1; }
echo "OK"

echo "=== [1/5] directory server ==="
cd "$CERTDIR"
{ sleep 120; echo "QUIT"; } | "$DIRBIN" > "$LOG/dir.log" 2>&1 &
DIRPID=$!
sleep 2
if ! kill -0 $DIRPID 2>/dev/null; then
    echo "FAIL: directory died"; cat "$LOG/dir.log"; exit 1
fi
echo "OK PID=$DIRPID"

echo "=== [2/5] node B (relay PORT=9002) ==="
{ sleep 120; } | NODE_ID=B PORT=9002 SERVER_ADDR=localhost:8080 "$NODEBIN" > "$LOG/nodeB.log" 2>&1 &
NODEBPID=$!
sleep 2

echo "=== [3/5] node C (dest PORT=9003) ==="
{ sleep 120; } | NODE_ID=C PORT=9003 SERVER_ADDR=localhost:8080 "$NODEBIN" > "$LOG/nodeC.log" 2>&1 &
NODECPID=$!
sleep 3

echo "=== [4/5] node A sends to C via 1 relay ==="
{
    sleep 2
    echo "LIST:"
    sleep 1
    echo "SEND:1:127.0.0.1:9003:hello-from-A"
    sleep 8
    echo "QUIT:"
} | NODE_ID=A PORT=9001 SERVER_ADDR=localhost:8080 "$NODEBIN" > "$LOG/nodeA.log" 2>&1
echo "Node A done"

echo ""
echo "=== LOGS ==="
echo "--- directory ---"
cat "$LOG/dir.log"
echo ""
echo "--- node A (sender) ---"
cat "$LOG/nodeA.log"
echo ""
echo "--- node B (relay) ---"
cat "$LOG/nodeB.log"
echo ""
echo "--- node C (dest) ---"
cat "$LOG/nodeC.log"

echo ""
echo "=== VERDICT ==="
ACK=0; MSG=0; RELAY=0; ERR=0

grep -qi "ACK" "$LOG/nodeA.log" && ACK=1
grep -qi "Message recu\|hello-from-A" "$LOG/nodeC.log" && MSG=1
grep -qi "Relai\|RELAY" "$LOG/nodeB.log" && RELAY=1
grep -qi "panic\|fatal" "$LOG/nodeA.log" "$LOG/nodeB.log" "$LOG/nodeC.log" "$LOG/dir.log" && ERR=1

[ $ACK -eq 1 ] && echo "PASS -- Node A got ACK" || echo "FAIL -- No ACK on node A"
[ $MSG -eq 1 ] && echo "PASS -- Node C received message" || echo "FAIL -- No message on node C"
[ $RELAY -eq 1 ] && echo "PASS -- Node B relayed" || echo "INFO -- Could not confirm relay on B"
[ $ERR -eq 0 ] && echo "PASS -- No panics" || echo "WARN -- panic/fatal in logs"
