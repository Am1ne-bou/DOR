#!/bin/bash
# start-dor.sh
# Lance le projet DOR et ouvre 4 terminaux interactifs (Linux)

set -e

cd "$(dirname "$0")"

echo -e "\033[36m=== Demarrage du projet DOR ===\033[0m"

# Detecter l'emulateur de terminal disponible
TERMINAL=""
for t in gnome-terminal konsole xfce4-terminal xterm tilix mate-terminal; do
    if command -v $t >/dev/null 2>&1; then
        TERMINAL=$t
        break
    fi
done

if [ -z "$TERMINAL" ]; then
    echo -e "\033[31mAucun emulateur de terminal trouve.\033[0m"
    echo "Installe gnome-terminal, konsole, xfce4-terminal ou xterm."
    exit 1
fi

echo -e "\033[33m[1/3] docker compose up -d...\033[0m"
docker compose up -d

echo -e "\033[33m[2/3] Attente des enregistrements (~30s)...\033[0m"
WAITED=0
MAX_WAIT=90
while [ $WAITED -lt $MAX_WAIT ]; do
    REGISTERED=$(docker compose logs directory 2>&1 | grep -c "registered" || true)
    if [ "$REGISTERED" -ge 3 ]; then
        echo -e "  \033[32m-> 3 noeuds enregistres !\033[0m"
        break
    fi
    sleep 2
    WAITED=$((WAITED + 2))
    echo "  ($REGISTERED/3 noeuds enregistres, ${WAITED}s)"
done

echo -e "\033[33m[3/3] Ouverture de 4 terminaux ($TERMINAL)...\033[0m"

open_terminal() {
    local container=$1
    local title="DOR - $container"
    local cmd="echo '=== Attached to $container ==='; echo 'Pour detacher : Ctrl+P puis Ctrl+Q'; echo; docker attach $container; exec bash"

    case $TERMINAL in
        gnome-terminal)
            gnome-terminal --title="$title" -- bash -c "$cmd" &
            ;;
        konsole)
            konsole --new-tab -p "tabtitle=$title" -e bash -c "$cmd" &
            ;;
        xfce4-terminal)
            xfce4-terminal --title="$title" -e "bash -c \"$cmd\"" &
            ;;
        tilix)
            tilix --title="$title" -e bash -c "$cmd" &
            ;;
        mate-terminal)
            mate-terminal --title="$title" -- bash -c "$cmd" &
            ;;
        xterm)
            xterm -T "$title" -e bash -c "$cmd" &
            ;;
    esac
}

for c in directory node1 node2 node3; do
    open_terminal $c
    sleep 0.3
done

echo
echo -e "\033[32m=== Pret ===\033[0m"
echo "4 terminaux ouverts."
echo "Pour tout arreter : docker compose down"
