#!/bin/bash
# === CONFIG ===
N_RELAYS=${1:-6}
KILL_INTERVAL=${2:-5}
DEATH_TIME=${3:-2}
MAX_KILLS=${4:-1}

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_DIR="$SCRIPT_DIR/../List_Serveur"
NODE_DIR="$SCRIPT_DIR/../node"
LOG_DIR="$SCRIPT_DIR/logs"
SIM_LATENCY=${SIM_LATENCY:-0}

mkdir -p $LOG_DIR
rm -f $LOG_DIR/*.log

pkill -f "relay-" 2>/dev/null
pkill -f "bench-" 2>/dev/null
pkill -f "node-receiver" 2>/dev/null
pkill -f "__go_build" 2>/dev/null
pkill -f "go run serveur.go" 2>/dev/null
fuser -k 8080/tcp 2>/dev/null
sleep 3

echo "=== DOR Benchmark ==="
echo "Relais: $N_RELAYS | Kill: ${KILL_INTERVAL}s | Dead: ${DEATH_TIME}s | MaxKills: $MAX_KILLS | Latency: ${SIM_LATENCY}ms"
echo ""

# === SERVEUR ===
echo "[1] Lancement du serveur..."
cd $SERVER_DIR
sleep infinity | go run serveur.go > $LOG_DIR/server.log 2>&1 &
SERVER_PID=$!
cd - > /dev/null
sleep 2

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "ERREUR: Serveur pas démarré"
    exit 1
fi
echo "    Serveur OK"

# === RECEIVER ===
echo "[2] Lancement du receiver..."
cd $NODE_DIR
sleep infinity | SIM_LATENCY=$SIM_LATENCY go run main.go node-receiver > $LOG_DIR/node-receiver.log 2>&1 &
RECEIVER_PID=$!
cd - > /dev/null
sleep 2

RECEIVER_PORT=$(grep -oP 'Started in port : \K[0-9]+' $LOG_DIR/node-receiver.log)
RECEIVER_IP=$(grep -oP 'IP: \K[0-9.]+' $LOG_DIR/node-receiver.log)
RECEIVER_IP=${RECEIVER_IP:-127.0.0.1}

if [ -z "$RECEIVER_PORT" ]; then
    echo "ERREUR: Receiver pas trouvé"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

echo ""
echo "=========================================="
echo "  RECEIVER: $RECEIVER_IP:$RECEIVER_PORT"
echo "=========================================="
echo ""

# === RELAIS ===
echo "[3] Lancement de $N_RELAYS relais..."
RELAY_PIDS=()
for i in $(seq 1 $N_RELAYS); do
    cd $NODE_DIR
    sleep infinity | SIM_LATENCY=$SIM_LATENCY go run main.go relay-$i > $LOG_DIR/relay-$i.log 2>&1 &
    RELAY_PIDS+=($!)
    cd - > /dev/null
    sleep 1
    echo "    relay-$i OK"
done

echo ""
echo "[4] Chaos dans 10s..."
sleep 10

# === KILL LOOP ===
cleanup() {
    echo ""
    echo "=== Nettoyage ==="
    kill $SERVER_PID 2>/dev/null
    kill $RECEIVER_PID 2>/dev/null
    for pid in "${RELAY_PIDS[@]}"; do
        kill $pid 2>/dev/null
    done
    pkill -f "relay-" 2>/dev/null
    pkill -f "__go_build" 2>/dev/null
    echo "Process arrêtés."
    exit 0
}
trap cleanup SIGINT SIGTERM

while true; do
    sleep $KILL_INTERVAL

    NB_KILLS=$((RANDOM % MAX_KILLS + 1))
    KILLED=()
    for k in $(seq 1 $NB_KILLS); do
        INDEX=$((RANDOM % N_RELAYS))
        VICTIM_NAME="relay-$((INDEX + 1))"
        if [[ " ${KILLED[@]} " =~ " $INDEX " ]]; then
            continue
        fi
        KILLED+=($INDEX)
        pkill -f "$VICTIM_NAME" 2>/dev/null
        echo "[$(date +%H:%M:%S)] KILL $VICTIM_NAME"
    done

    sleep $DEATH_TIME

    for INDEX in "${KILLED[@]}"; do
        VICTIM_NAME="relay-$((INDEX + 1))"
        cd $NODE_DIR
        sleep infinity | SIM_LATENCY=$SIM_LATENCY go run main.go $VICTIM_NAME > $LOG_DIR/$VICTIM_NAME.log 2>&1 &
        RELAY_PIDS[$INDEX]=$!
        cd - > /dev/null
        echo "[$(date +%H:%M:%S)] RELAUNCHED $VICTIM_NAME"
    done
done