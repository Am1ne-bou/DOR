#!/bin/sh

source $1

docker network rm dor 2>/dev/null
docker network create dor

y=0
for PROFILE in smartphone_2G server laptop_WIFI5; do
  COUNT=$(eval echo \$$PROFILE)
  RAM=$(eval echo \$${PROFILE}_RAM)
  CPU=$(eval echo \$${PROFILE}_CPU)

  i=1
  while [ $i -le $COUNT ]; do
    docker run -it -d \
      --cap-add=NET_ADMIN \
      --network dor \
      --name "$PROFILE"$i \
      -e NETWORK_PROFILE="$PROFILE" \
      -e PORT=$((9000 + i + y)) \
      -e NODE_ADDR="host.docker.internal" \
      -p $((9000 + i + y)):$((9000 + i + y)) \
      --add-host=host.docker.internal:host-gateway \
      --rm \
      device-amd:latest

    i=$((i + 1))
  done
  y=$((y+COUNT))
done