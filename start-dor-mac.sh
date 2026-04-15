#!/bin/bash
# start-dor-mac.sh
# Lance le projet DOR et ouvre 4 terminaux interactifs (macOS)

set -e

cd "$(dirname "$0")"

echo "=== Demarrage du projet DOR ==="

echo "[1/3] docker compose up -d..."
docker compose up -d

echo "[2/3] Attente des enregistrements (~30s)..."
WAITED=0
MAX_WAIT=90
while [ $WAITED -lt $MAX_WAIT ]; do
    REGISTERED=$(docker compose logs directory 2>&1 | grep -c "registered" || true)
    if [ "$REGISTERED" -ge 3 ]; then
        echo "  -> 3 noeuds enregistres !"
        break
    fi
    sleep 2
    WAITED=$((WAITED + 2))
    echo "  ($REGISTERED/3 noeuds enregistres, ${WAITED}s)"
done

echo "[3/3] Ouverture de 4 terminaux (Terminal.app)..."

open_terminal() {
    local container=$1
    osascript <<EOF
tell application "Terminal"
    do script "echo '=== Attached to $container ==='; echo 'Pour detacher : Ctrl+P puis Ctrl+Q'; echo; docker attach $container"
    set custom title of front window to "DOR - $container"
end tell
EOF
}

for c in directory node1 node2 node3; do
    open_terminal $c
    sleep 0.3
done

echo
echo "=== Pret ==="
echo "4 terminaux ouverts."
echo "Pour tout arreter : docker compose down"
