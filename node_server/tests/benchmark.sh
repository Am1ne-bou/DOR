#!/bin/bash

# === CONFIG ===
N_RELAYS=${1:-6}         # nombre de relais (defaut 6)
KILL_INTERVAL=${2:-5}    # kill toutes les X secondes (defaut 5)
DEATH_TIME=${3:-2}
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_DIR="$SCRIPT_DIR/../List_Serveur"
NODE_DIR="$SCRIPT_DIR/../node"
LOG_DIR="$SCRIPT_DIR/logs"

# === CLEANUP ===
mkdir -p $LOG_DIR
rm -f $LOG_DIR/*.log

# Kill tout ce qui reste d'un ancien test
pkill -f "go run main.go node-" 2>/dev/null
pkill -f "go run serveur.go" 2>/dev/null
sleep 1

echo "=== DOR Benchmark ==="
echo "Relais: $N_RELAYS"
echo "Kill interval: ${KILL_INTERVAL}s"
echo ""

# === 1. LANCER LE SERVEUR ===
echo "[1] Lancement du serveur..."
cd $SERVER_DIR
sleep infinity | go run serveur.go > $LOG_DIR/server.log 2>&1 &
SERVER_PID=$!
cd - > /dev/null
sleep 2

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "ERREUR: Le serveur n'a pas démarré. Vérifie $LOG_DIR/server.log"
    exit 1
fi
echo "    Serveur lancé (PID: $SERVER_PID)"

# === 2. LANCER LE RECEIVER (protégé, jamais kill) ===
echo "[2] Lancement du receiver (node-receiver)..."
cd $NODE_DIR
sleep infinity | go run main.go node-receiver > $LOG_DIR/node-receiver.log 2>&1 &
RECEIVER_PID=$!
cd - > /dev/null
sleep 2

# Extraire le port du receiver depuis ses logs
RECEIVER_PORT=$(grep -oP 'Started in port : \K[0-9]+' $LOG_DIR/node-receiver.log)

if [ -z "$RECEIVER_PORT" ]; then
    echo "ERREUR: Port du receiver pas trouvé. Vérifie $LOG_DIR/node-receiver.log"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# Extraire l'IP du receiver
RECEIVER_IP=$(grep -oP 'IP: \K[0-9.]+' $LOG_DIR/node-receiver.log)
if [ -z "$RECEIVER_IP" ]; then
    RECEIVER_IP="127.0.0.1"
fi

echo "    Receiver lancé (PID: $RECEIVER_PID)"
echo ""
echo "=========================================="
echo "  RECEIVER: $RECEIVER_IP:$RECEIVER_PORT"
echo "=========================================="
echo ""

# === 3. LANCER LES RELAIS ===
echo "[3] Lancement de $N_RELAYS relais..."
RELAY_PIDS=()
for i in $(seq 1 $N_RELAYS); do
    cd $NODE_DIR
    sleep infinity | go run main.go relay-$i > $LOG_DIR/relay-$i.log 2>&1 &
    RELAY_PIDS+=($!)
    cd - > /dev/null
    sleep 1
    echo "    relay-$i lancé (PID: ${RELAY_PIDS[$i-1]})"
done

echo ""
echo "[4] Tous les noeuds sont lancés."
echo "    Lance TON node dans un autre terminal:"
echo ""
echo "    cd $NODE_DIR && go run main.go mon-node"
echo ""
echo "    Puis envoie avec:"
echo "    SEND:3:$RECEIVER_IP:$RECEIVER_PORT:hello"
echo "    ou BENCH:50:3:$RECEIVER_IP:$RECEIVER_PORT"
echo ""
echo "[5] Début du chaos dans 10s... (Ctrl+C pour tout arrêter)"
sleep 10

# === 4. KILL LOOP ===
cleanup() {
    echo ""
    echo "=== Nettoyage ==="
    kill $SERVER_PID 2>/dev/null
    kill $RECEIVER_PID 2>/dev/null
    for pid in "${RELAY_PIDS[@]}"; do
        kill $pid 2>/dev/null
    done
    echo "Tous les process arrêtés."
    echo "Logs dans $LOG_DIR/"
    exit 0
}
trap cleanup SIGINT SIGTERM

while true; do
    sleep $KILL_INTERVAL

    # Pick un relai random
    INDEX=$((RANDOM % N_RELAYS))
    VICTIM_NAME="relay-$((INDEX + 1))"
    VICTIM_PID=${RELAY_PIDS[$INDEX]}

    # Vérifie si le process est encore vivant
    if kill -0 $VICTIM_PID 2>/dev/null; then
        echo "[$(date +%H:%M:%S)] KILL $VICTIM_NAME (PID: $VICTIM_PID)"
        pkill -f "$VICTIM_NAME" 2>/dev/null
    fi
    sleep $DEATH_TIME

    # Relance
    cd $NODE_DIR
    sleep infinity | go run main.go $VICTIM_NAME > $LOG_DIR/$VICTIM_NAME.log 2>&1 &
    RELAY_PIDS[$INDEX]=$!
    cd - > /dev/null
    echo "[$(date +%H:%M:%S)] RELAUNCHED $VICTIM_NAME (PID: ${RELAY_PIDS[$INDEX]})"
done