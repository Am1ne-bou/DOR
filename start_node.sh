#!/bin/bash
N=5

gnome-terminal -- bash -c "cd node_server/List_Serveur && go run serveur.go; bash" &
sleep 2
for i in $(seq 1 $N); do
  gnome-terminal -- bash -c "cd node_server/node && go run . node-$i; bash" &
done