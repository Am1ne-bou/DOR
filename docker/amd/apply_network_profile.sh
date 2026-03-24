#!/bin/sh

PROFILE=$1
IFACE=eth0

tc qdisc del dev $IFACE root 2>/dev/null

GREEN="\033[32m"
RED="\033[31m"
RESET="\033[0m"

case "$PROFILE" in
  smartphone_EDGE)
    tc qdisc add dev $IFACE root handle 1: tbf rate 200kbit burst 8kbit latency 800ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 400ms 100ms loss 5%
    printf "${GREEN}EDGE config applied (0.2 Mbps) !${RESET}\n"
    ;;

  smartphone_2G)
    tc qdisc add dev $IFACE root handle 1: tbf rate 100kbit burst 8kbit latency 1000ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 600ms 150ms loss 8%
    printf "${GREEN}2G config applied (0.1 Mbps) !${RESET}\n"
    ;;

  smartphone_3G)
    tc qdisc add dev $IFACE root handle 1: tbf rate 1mbit burst 32kbit latency 400ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 150ms 50ms loss 2%
    printf "${GREEN}3G config applied (1 Mbps) !${RESET}\n"
    ;;

  smartphone_4G)
    tc qdisc add dev $IFACE root handle 1: tbf rate 20mbit burst 64kbit latency 200ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 50ms 20ms loss 0.5%
    printf "${GREEN}4G config applied (20 Mbps) !${RESET}\n"
    ;;

  smartphone_5G)
    tc qdisc add dev $IFACE root handle 1: tbf rate 150mbit burst 128kbit latency 100ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 20ms 5ms loss 0.1%
    printf "${GREEN}5G config applied (150 Mbps) !${RESET}\n"
    ;;

  laptop_WIFI3)
    tc qdisc add dev $IFACE root handle 1: tbf rate 11mbit burst 32kbit latency 100ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 30ms 10ms loss 1%
    printf "${GREEN}WiFi 3 (802.11g) config applied !${RESET}\n"
    ;;

  laptop_WIFI4)
    tc qdisc add dev $IFACE root handle 1: tbf rate 50mbit burst 64kbit latency 80ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 20ms 5ms loss 0.5%
    printf "${GREEN}WiFi 4 (802.11n) config applied !${RESET}\n"
    ;;

  laptop_WIFI5)
    tc qdisc add dev $IFACE root handle 1: tbf rate 200mbit burst 128kbit latency 50ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 10ms 3ms loss 0.2%
    printf "${GREEN}WiFi 5 (802.11ac) config applied !${RESET}\n"
    ;;

  laptop_WIFI6)
    tc qdisc add dev $IFACE root handle 1: tbf rate 400mbit burst 256kbit latency 30ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 8ms 2ms loss 0.1%
    printf "${GREEN}WiFi 6 (802.11ax) config applied !${RESET}\n"
    ;;

  laptop_WIFI7)
    tc qdisc add dev $IFACE root handle 1: tbf rate 1gbit burst 512kbit latency 20ms
    tc qdisc add dev $IFACE parent 1:1 handle 10: netem delay 5ms 1ms loss 0.05%
    printf "${GREEN}WiFi 7 config applied (1 Gbps) !${RESET}\n"
    ;;

  server)
    printf "${GREEN}Server's config applied !${RESET}\n"
    ;;

  *)
    printf "${RED}Nothing changed ! Profile not found !${RESET}\n"
    ;;
esac