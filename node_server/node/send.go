package main

import (
	"fmt"
	"log/slog"
	mrand "math/rand"
	"strings"
	"time"

	"project/node_server/model"
)

// TODO: make MaxRetries and RetryDelay configurable via env vars instead of hardcoded
func SendWithRetry(
	node *model.Node,
	serverAddr string,
	destAddr string,
	message string,
	numRelays int,
	publicKeys *KeyCache,
	maxRetries int,
	currentTry int,
	startTime time.Time,
) {
	if currentTry >= maxRetries {
		slog.Warn("abandon after max retries", "dest", destAddr, "tries", maxRetries)
		elapsed := time.Since(startTime).Milliseconds()
		fmt.Printf("RESULT|%s|ABANDON|%d|%dms\n", destAddr, maxRetries, elapsed)
		return
	}

	if currentTry > 0 {
		slog.Info("retrying", "dest", destAddr, "try", currentTry, "max", maxRetries)
	}

	listStr, err := node.GetNodesList()
	if err != nil {
		slog.Error("get nodes list", "err", err)
		return
	}
	if listStr == "LIST:empty" {
		slog.Warn("no nodes available")
		return
	}

	listData := strings.TrimPrefix(listStr, "LIST:")
	entries := strings.Split(listData, ",")

	var candidates []string
	nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)

	for _, entry := range entries {
		fields := strings.SplitN(entry, "|", 4)
		if len(fields) < 4 {
			continue
		}
		ip := fields[1]
		port := fields[2]
		addr := ip + ":" + port
		if addr != nodeAddr && addr != destAddr {
			candidates = append(candidates, addr)
		}
	}

	if numRelays > len(candidates) {
		numRelays = len(candidates)
	}
	if numRelays == 0 {
		slog.Warn("not enough relay nodes")
		return
	}

	for i := len(candidates) - 1; i > 0; i-- {
		j := mrand.Intn(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}

	relays := candidates[:numRelays]
	route := append(relays[:len(relays):len(relays)], destAddr)
	slog.Debug("route built", "forward", route)

	var returnRoute []string
	for i := len(relays) - 1; i >= 0; i-- {
		returnRoute = append(returnRoute, relays[i])
	}
	returnRoute = append(returnRoute, nodeAddr)
	slog.Debug("route built", "return", returnRoute)

	onion, msgID, firstNackID, err := Encapsulator_func(message, route, returnRoute, publicKeys, serverAddr, nodeAddr)
	if err != nil {
		slog.Error("encapsulation failed", "err", err)
		return
	}

	ackChan := make(chan bool, 1)
	node.Mu.Lock()
	node.PendingACKs[msgID] = ackChan
	node.PendingACKs[firstNackID] = ackChan
	node.Mu.Unlock()

	// Telemetry forward
	exid := msgID
	go func(r []string) {
		Telemetry(nodeAddr, r[0], "SEND", message, exid, 1)
		for i := 0; i < len(r)-1; i++ {
			time.Sleep(50 * time.Millisecond)
			Telemetry(r[i], r[i+1], "RELAY", "", exid, i+2)
		}
	}(append([]string{}, route...))

	err = node.SendTo(route[0], onion)
	if err != nil {
		slog.Error("send failed", "err", err)
		node.Mu.Lock()
		delete(node.PendingACKs, msgID)
		delete(node.PendingACKs, firstNackID)
		node.Mu.Unlock()
		SendWithRetry(node, serverAddr, destAddr, message, numRelays, publicKeys, maxRetries, currentTry+1, startTime)
		return
	}

	slog.Info("message sent, waiting ACK", "msgID", msgID)

	go func(id string, nackID string, ch chan bool) {
		select {
		case success := <-ch:
			elapsed := time.Since(startTime).Milliseconds()
			if success {
				slog.Info("ACK received", "msgID", id)
				fmt.Printf("RESULT|%s|ACK|%d|%dms\n", destAddr, currentTry, elapsed)
				Telemetry(destAddr, nodeAddr, "ACK", "", exid, 1)
			} else {
				slog.Warn("NACK received, retrying", "msgID", msgID)
				Telemetry(route[0], nodeAddr, "NACK", "", exid, 1)
				node.Mu.Lock()
				delete(node.PendingACKs, msgID)
				delete(node.PendingACKs, firstNackID)
				node.Mu.Unlock()
				SendWithRetry(node, serverAddr, destAddr, message, numRelays, publicKeys, maxRetries, currentTry+1, startTime)
			}
		case <-time.After(time.Second * 8):
			elapsed := time.Since(startTime).Milliseconds()
			fmt.Printf("RESULT|%s|TIMEOUT|%d|%dms\n", destAddr, currentTry, elapsed)
			slog.Warn("ACK timeout", "msgID", id)
			Telemetry(route[0], nodeAddr, "NACK", "timeout", exid, 1)
			node.Mu.Lock()
			delete(node.PendingACKs, id)
			delete(node.PendingACKs, nackID)
			node.Mu.Unlock()
		}
	}(msgID, firstNackID, ackChan)
}
