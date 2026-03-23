#!/bin/bash
# =============================================================================
# DOR Full Benchmark - Multi-sender + latency
#
# Usage: ./full_bench.sh <nb_relais> <kill_interval> <dead_time> [output_dir]
# Env:   SIM_LATENCY=80 NBR_SENDERS=3 NBR_MESSAGES=100 MAX_KILLS=3 ./full_bench.sh 20 3 8
# =============================================================================

if [ $# -lt 3 ]; then
    echo "Usage: ./full_bench.sh <nb_relais> <kill_interval> <dead_time> [output_dir]"
    echo "Env: SIM_LATENCY=80 NBR_SENDERS=3 MAX_KILLS=3 ./full_bench.sh 20 3 8"
    exit 1
fi

NB_RELAIS=$1
KILL_INT=$2
DEAD_TIME=$3

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NODE_DIR="$SCRIPT_DIR/../node"
# Configurable via variables d'environnement
NBR_MESSAGES=${NBR_MESSAGES:-100}
NBR_RELAYS_ROUTE=${NBR_RELAYS_ROUTE:-3}
NBR_SENDERS=${NBR_SENDERS:-1}
MAX_KILLS=${MAX_KILLS:-1}
SIM_LATENCY=${SIM_LATENCY:-0}

CONFIG_NAME="${NB_RELAIS}r_${KILL_INT}k_${DEAD_TIME}d_${MAX_KILLS}mk_${NBR_SENDERS}s_${SIM_LATENCY}ms"

BASE_DIR=${4:-$SCRIPT_DIR}
OUTPUT_DIR="$BASE_DIR/$CONFIG_NAME"

mkdir -p "$OUTPUT_DIR"

kill_everything() {
    pkill -f "go run main.go" 2>/dev/null
    pkill -f "go run serveur.go" 2>/dev/null
    pkill -f "__go_build" 2>/dev/null
    pkill -f "relay-" 2>/dev/null
    pkill -f "bench-sender" 2>/dev/null
    pkill -f "node-receiver" 2>/dev/null
    fuser -k 8080/tcp 2>/dev/null
    sleep 8
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
    echo "--- $LABEL (maxRetries=$MAX_RETRIES, senders=$NBR_SENDERS, latency=${SIM_LATENCY}ms) ---"

    > "$LOG_FILE"

    kill_everything

    # Lancer benchmark.sh
    BENCH_LOG="/tmp/dor_bench_output_$$.log"
    > "$BENCH_LOG"
    SIM_LATENCY=$SIM_LATENCY "$SCRIPT_DIR/benchmark.sh" $NB_RELAIS $KILL_INT $DEAD_TIME $MAX_KILLS > "$BENCH_LOG" 2>&1 &
    BENCH_PID=$!

    echo "  Attente du receiver..."
    RECEIVER_ADDR=$(wait_for_receiver "$BENCH_LOG")
    if [ -z "$RECEIVER_ADDR" ]; then
        echo "  ERREUR: Receiver pas trouvé"
        kill $BENCH_PID 2>/dev/null
        kill_everything
        return 1
    fi
    echo "  Receiver: $RECEIVER_ADDR"

    echo "  Attente 20s pour le chaos..."
    sleep 20

    # Lancer les senders
    BENCH_CMD="BENCH:${NBR_MESSAGES}:${NBR_RELAYS_ROUTE}:${MAX_RETRIES}:${RECEIVER_ADDR}"
    echo "  Commande: $BENCH_CMD x $NBR_SENDERS senders"

    SENDER_PIDS=()
    for s in $(seq 1 $NBR_SENDERS); do
        cd "$NODE_DIR"
        (sleep 2 && echo "$BENCH_CMD" && sleep infinity) | SIM_LATENCY=$SIM_LATENCY go run main.go bench-sender-$s > "${LOG_FILE}.${s}" 2>&1 &
        SENDER_PIDS+=($!)
        cd "$SCRIPT_DIR"
        sleep 1
    done

    # Attendre les résultats (total = NBR_MESSAGES * NBR_SENDERS)
    TOTAL_EXPECTED=$((NBR_MESSAGES * NBR_SENDERS))
    MAX_WAIT=300
    ELAPSED=0
    while [ $ELAPSED -lt $MAX_WAIT ]; do
        sleep 5
        ELAPSED=$((ELAPSED + 5))
        # Compter les résultats de tous les senders
        NB_RESULTS=0
        for s in $(seq 1 $NBR_SENDERS); do
            R=$(grep -c "^RESULT|" "${LOG_FILE}.${s}" 2>/dev/null || echo "0")
            NB_RESULTS=$((NB_RESULTS + R))
        done
        if [ "$NB_RESULTS" -ge "$TOTAL_EXPECTED" ]; then
            break
        fi
    done
    echo "  $NB_RESULTS/$TOTAL_EXPECTED résultats en ${ELAPSED}s"

    # Merge les logs
    cat "${LOG_FILE}".* > "$LOG_FILE" 2>/dev/null
    rm -f "${LOG_FILE}".* 2>/dev/null

    # Cleanup
    for pid in "${SENDER_PIDS[@]}"; do
        kill $pid 2>/dev/null
    done
    kill $BENCH_PID 2>/dev/null
    kill_everything

    echo "  $LABEL terminé"
}

# === MAIN ===

echo "=============================================="
echo "  DOR Benchmark: $CONFIG_NAME"
echo "  $(date)"
echo "  Relais=$NB_RELAIS Kill=${KILL_INT}s Dead=${DEAD_TIME}s"
echo "  Senders=$NBR_SENDERS Msg/sender=$NBR_MESSAGES MaxKills=$MAX_KILLS Latency=${SIM_LATENCY}ms"
echo "=============================================="

WITH_LOG="$SCRIPT_DIR/sender_with_retry.log"
run_one 3 "$WITH_LOG" "AVEC RETRY"

WITHOUT_LOG="$SCRIPT_DIR/sender_without_retry.log"
run_one 1 "$WITHOUT_LOG" "SANS RETRY"

echo ""
echo "=== ANALYSE ==="
"$SCRIPT_DIR/run_analysis.sh" "$OUTPUT_DIR" "$WITH_LOG" "$WITHOUT_LOG"

echo ""
echo "Résultats dans: $OUTPUT_DIR/"