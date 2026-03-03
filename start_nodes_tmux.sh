#!/bin/bash
N=4 # par defaut
SESSION="DOR"

while getopts "n:" opt; do
  case $opt in
    n) N=$OPTARG ;;
    *) echo "Usage: $0 [-n number_of_nodes]"; exit 1 ;;
  esac
done

# permet de tuer les anciennes fenetres
lsof -ti :8080 | xargs kill -9
tmux kill-session -t $SESSION

tmux new-session -d -s $SESSION -n "server" \
"cd node_server/List_Serveur && go run serveur.go; bash"

sleep 2

for i in $(seq 1 $N); do
    tmux new-window -t $SESSION:$i -n "node-$i" \
    "cd node_server/node && go run main.go node-$i"
done

tmux select-window -t $SESSION:0
tmux attach -t $SESSION