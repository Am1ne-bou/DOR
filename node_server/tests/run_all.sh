#!/bin/bash
# Lance full_bench.sh pour chaque config
# Usage: ./run_all.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUN_DIR="$SCRIPT_DIR/results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RUN_DIR"

CONFIGS=(
    "5 1 5"
    "3 1 8"
    "3 2 10"
    "4 2 8"
    "4 3 5"
    "3 3 5"
    "6 3 5"
    "10 3 5"
    "5 10 5"
    "5 3 5"
    "5 5 3"
    "5 5 10"
    "3 2 15"
    "5 5 20"
)

echo "=============================================="
echo "  DOR Full Benchmark - Toutes configs"
echo "  $(date)"
echo "  Résultats: $RUN_DIR"
echo "=============================================="

for config in "${CONFIGS[@]}"; do
    read -r A B C <<< "$config"
    echo ""
    echo ">>> Config: $A $B $C"
    "$SCRIPT_DIR/full_bench.sh" $A $B $C "$RUN_DIR"
    pkill -f "relay-" 2>/dev/null
    pkill -f "bench-" 2>/dev/null
    pkill -f "node-receiver" 2>/dev/null
    pkill -f "__go_build" 2>/dev/null
    fuser -k 8080/tcp 2>/dev/null
    sleep 15
done

echo ""
echo "=============================================="
echo "  RÉSUMÉ"
echo "=============================================="
echo ""
printf "%-15s | %12s | %12s | %8s\n" "Config" "Avec retry" "Sans retry" "Gain"
printf "%-15s-+-%12s-+-%12s-+-%8s\n" "---------------" "------------" "------------" "--------"

for config in "${CONFIGS[@]}"; do
    read -r A B C <<< "$config"
    DIR="$RUN_DIR/${A}_${B}_${C}"
    if [ -f "$DIR/report.txt" ]; then
        WITH=$(grep "Taux de livraison" "$DIR/report.txt" | grep -oP '[0-9.]+%' | head -1)
        WITHOUT=$(grep "Taux de livraison" "$DIR/report.txt" | grep -oP '[0-9.]+%' | tail -1)
        W=$(echo "$WITH" | tr -d '%')
        WO=$(echo "$WITHOUT" | tr -d '%')
        GAIN=$(echo "$W - $WO" | bc 2>/dev/null || echo "?")
        printf "%-15s | %12s | %12s | %7s%%\n" "${A}_${B}_${C}" "$WITH" "$WITHOUT" "+$GAIN"
    else
        printf "%-15s | %12s | %12s | %8s\n" "${A}_${B}_${C}" "erreur" "erreur" "-"
    fi
done

# Sauvegarder le résumé
echo "" > "$RUN_DIR/summary.txt"
printf "%-15s | %12s | %12s | %8s\n" "Config" "Avec retry" "Sans retry" "Gain" >> "$RUN_DIR/summary.txt"
printf "%-15s-+-%12s-+-%12s-+-%8s\n" "---------------" "------------" "------------" "--------" >> "$RUN_DIR/summary.txt"

for config in "${CONFIGS[@]}"; do
    read -r A B C <<< "$config"
    DIR="$RUN_DIR/${A}_${B}_${C}"
    if [ -f "$DIR/report.txt" ]; then
        WITH=$(grep "Taux de livraison" "$DIR/report.txt" | grep -oP '[0-9.]+%' | head -1)
        WITHOUT=$(grep "Taux de livraison" "$DIR/report.txt" | grep -oP '[0-9.]+%' | tail -1)
        W=$(echo "$WITH" | tr -d '%')
        WO=$(echo "$WITHOUT" | tr -d '%')
        GAIN=$(echo "$W - $WO" | bc 2>/dev/null || echo "?")
        printf "%-15s | %12s | %12s | %7s%%\n" "${A}_${B}_${C}" "$WITH" "$WITHOUT" "+$GAIN" >> "$RUN_DIR/summary.txt"
    fi
done

echo ""
echo "Résumé sauvegardé dans $RUN_DIR/summary.txt"
echo "Résultats dans: $RUN_DIR/"