#!/bin/bash
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
N=5

gnome-terminal -- bash -c "cd '$ROOT/node_server/List_Serveur' && go run serveur.go; bash" &
sleep 2
for i in $(seq 1 $N); do
  gnome-terminal -- bash -c "cd '$ROOT/node_server/node' && go run . node-$i; bash" &
done