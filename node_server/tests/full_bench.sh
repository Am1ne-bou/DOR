#!/bin/bash
# =============================================================================
# DOR Full Benchmark
#
# Usage: ./full_bench.sh <nb_relais> <kill_interval> <dead_time>
# Exemple: ./full_bench.sh 5 3 5
#
# Fait exactement ce que tu fais manuellement:
# 1. Lance benchmark.sh
# 2. Lance sender avec maxRetries=3, tape BENCH
# 3. Attend les résultats
# 4. Kill tout
# 5. Relance benchmark.sh
# 6. Lance sender avec maxRetries=1, tape BENCH
# 7. Attend les résultats
# 8. Kill tout
# 9. Analyse
# =============================================================================

if [ $# -lt 3 ]; then
    echo "Usage: ./full_bench.sh <nb_relais> <kill_interval> <dead_time>"
    echo "Exemple: ./full_bench.sh 5 3 5"
    exit 1
fi

NB_RELAIS=$1
KILL_INT=$2
DEAD_TIME=$3

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NODE_DIR="$SCRIPT_DIR/../node"
CONFIG_NAME="${NB_RELAIS}_${KILL_INT}_${DEAD_TIME}"
BASE_DIR=${4:-$SCRIPT_DIR}
OUTPUT_DIR="$BASE_DIR/$CONFIG_NAME"

NBR_MESSAGES=300
NBR_RELAYS_ROUTE=3

mkdir -p "$OUTPUT_DIR"

kill_everything() {
    pkill -f "go run main.go" 2>/dev/null
    pkill -f "go run serveur.go" 2>/dev/null
    pkill -f "__go_build" 2>/dev/null
    pkill -f "relay-" 2>/dev/null
    pkill -f "bench-" 2>/dev/null
    pkill -f "node-receiver" 2>/dev/null
    fuser -k 8080/tcp 2>/dev/null
    sleep 8
    # tuer tout ce qui reste
    pkill -f "relay-" 2>/dev/null
    pkill -f "bench-" 2>/dev/null
    fuser -k 8080/tcp 2>/dev/null
    while fuser 8080/tcp > /dev/null 2>&1; do
        sleep 2
    done
    sleep 2
}

wait_for_receiver() {
    local LOG_FILE=$1
    for i in $(seq 1 30); do
        ADDR=$(grep -oP 'RECEIVER: \K[0-9.:]+' "$LOG_FILE" 2>/dev/null)
        if [ -n "$ADDR" ]; then
            echo "$ADDR"
            return 0
        fi
        sleep 1
    done
    return 1
}

run_one() {
    local MAX_RETRIES=$1
    local LOG_FILE=$2
    local LABEL=$3

    echo ""
    echo "--- $LABEL (maxRetries=$MAX_RETRIES) ---"

    # Vider le log
    > "$LOG_FILE"

    # Kill tout avant de commencer
    kill_everything

    # Lancer benchmark.sh (serveur + receiver + relais + chaos)
    BENCH_LOG="/tmp/dor_bench_output_$$.log"
    > "$BENCH_LOG"
    "$SCRIPT_DIR/benchmark.sh" $NB_RELAIS $KILL_INT $DEAD_TIME > "$BENCH_LOG" 2>&1 &
    BENCH_PID=$!

    # Attendre le receiver
    echo "  Attente du receiver..."
    RECEIVER_ADDR=$(wait_for_receiver "$BENCH_LOG")
    if [ -z "$RECEIVER_ADDR" ]; then
        echo "  ERREUR: Receiver pas trouvé"
        cat "$BENCH_LOG"
        kill $BENCH_PID 2>/dev/null
        kill_everything
        return 1
    fi
    echo "  Receiver: $RECEIVER_ADDR"

    # Attendre que le chaos fasse des dégâts
    echo "  Attente 20s pour que le chaos s'installe..."
    sleep 20

    # Lancer le sender avec BENCH
    BENCH_CMD="BENCH:${NBR_MESSAGES}:${NBR_RELAYS_ROUTE}:${MAX_RETRIES}:${RECEIVER_ADDR}"
    echo "  Sender: $BENCH_CMD"

    cd "$NODE_DIR"
    (sleep 2 && echo "$BENCH_CMD" && sleep infinity) | go run main.go bench-sender > "$LOG_FILE" 2>&1 &
    SENDER_PID=$!
    cd "$SCRIPT_DIR"

    # Attendre les résultats
    MAX_WAIT=180
    ELAPSED=0
    while [ $ELAPSED -lt $MAX_WAIT ]; do
        sleep 5
        ELAPSED=$((ELAPSED + 5))
        NB_RESULTS=$(grep -c "^RESULT|" "$LOG_FILE" 2>/dev/null || echo "0")
        if [ "$NB_RESULTS" -ge "$NBR_MESSAGES" ]; then
            break
        fi
    done
    echo "  $NB_RESULTS/$NBR_MESSAGES résultats en ${ELAPSED}s"

    # Kill tout
    kill $SENDER_PID 2>/dev/null
    kill $BENCH_PID 2>/dev/null
    kill_everything

    echo "  $LABEL terminé"
}

# =============================================================================
# MAIN
# =============================================================================

echo "=============================================="
echo "  DOR Benchmark: $CONFIG_NAME"
echo "  $(date)"
echo "  $NB_RELAIS relais, kill=${KILL_INT}s, dead=${DEAD_TIME}s"
echo "=============================================="

# Run 1: AVEC retry
WITH_LOG="$SCRIPT_DIR/sender_with_retry.log"
run_one 3 "$WITH_LOG" "AVEC RETRY"

# Run 2: SANS retry
WITHOUT_LOG="$SCRIPT_DIR/sender_without_retry.log"
run_one 1 "$WITHOUT_LOG" "SANS RETRY"

# Analyse
echo ""
echo "=== ANALYSE ==="
"$SCRIPT_DIR/run_analysis.sh" "$OUTPUT_DIR" "$WITH_LOG" "$WITHOUT_LOG"

echo ""
echo "Résultats dans: $OUTPUT_DIR/"