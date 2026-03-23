#!/bin/bash
# Lance full_bench.sh pour chaque config
# Usage: ./run_all.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$SCRIPT_DIR/results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RUN_DIR"

# === CONFIGS ===
# Format: "relais kill dead max_kills senders messages latency"
CONFIGS=(

    "10 3 5 2 1 200 50"  # latence faible : 50ms/hop (~150ms par message)
    "10 3 5 2 1 200 100" # latence moyenne : 100ms/hop (~300ms par message)
    "10 3 5 2 1 200 200" # latence forte : 200ms/hop (~600ms par message)

)

echo "=============================================="
echo "  DOR Full Benchmark - $(date)"
echo "  ${#CONFIGS[@]} configs"
echo "  Résultats: $RUN_DIR"
echo "=============================================="

for config in "${CONFIGS[@]}"; do
    read -r RELAIS KILL DEAD MKILLS SENDERS MSGS LAT <<< "$config"
    echo ""
    echo ">>> Config: ${RELAIS}r ${KILL}k ${DEAD}d ${MKILLS}mk ${SENDERS}s ${MSGS}m ${LAT}ms"

    export NBR_MESSAGES=$MSGS
    export NBR_SENDERS=$SENDERS
    export MAX_KILLS=$MKILLS
    export SIM_LATENCY=$LAT

    "$SCRIPT_DIR/full_bench.sh" $RELAIS $KILL $DEAD "$RUN_DIR"

    # Force cleanup entre configs
    pkill -f "relay-" 2>/dev/null
    pkill -f "bench-" 2>/dev/null
    pkill -f "node-receiver" 2>/dev/null
    pkill -f "__go_build" 2>/dev/null
    fuser -k 8080/tcp 2>/dev/null
    sleep 15
done

# === RÉSUMÉ ===
echo ""
echo "=============================================="
echo "  RÉSUMÉ"
echo "=============================================="
echo ""
printf "%-20s | %8s | %12s | %12s | %8s\n" "Config" "Latency" "Avec retry" "Sans retry" "Gain"
printf "%-20s-+-%8s-+-%12s-+-%12s-+-%8s\n" "--------------------" "--------" "------------" "------------" "--------"

SUMMARY=""
for config in "${CONFIGS[@]}"; do
    read -r RELAIS KILL DEAD MKILLS SENDERS MSGS LAT <<< "$config"
    CONFIG_NAME="${RELAIS}r_${KILL}k_${DEAD}d_${MKILLS}mk_${SENDERS}s_${LAT}ms"
    DIR="$RUN_DIR/$CONFIG_NAME"

    LABEL="${RELAIS}r/${SENDERS}s/${MKILLS}mk"

    if [ -f "$DIR/report.txt" ]; then
        WITH=$(grep "Taux de livraison" "$DIR/report.txt" | grep -oP '[0-9.]+%' | head -1)
        WITHOUT=$(grep "Taux de livraison" "$DIR/report.txt" | grep -oP '[0-9.]+%' | tail -1)
        W=$(echo "$WITH" | tr -d '%')
        WO=$(echo "$WITHOUT" | tr -d '%')
        GAIN=$(echo "$W - $WO" | bc 2>/dev/null || echo "?")
        LINE=$(printf "%-20s | %7sms | %12s | %12s | %7s%%\n" "$LABEL" "$LAT" "$WITH" "$WITHOUT" "+$GAIN")
    else
        LINE=$(printf "%-20s | %7sms | %12s | %12s | %8s\n" "$LABEL" "$LAT" "erreur" "erreur" "-")
    fi
    echo "$LINE"
    SUMMARY+="$LINE"$'\n'
done

echo ""
echo "$SUMMARY" > "$RUN_DIR/summary.txt"
echo "Résumé sauvegardé dans $RUN_DIR/summary.txt"
echo "Résultats dans: $RUN_DIR/"