package main

import (
	"fmt"
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
		fmt.Printf("Abandon après %d tentatives pour %s\n\n", maxRetries, destAddr)
		elapsed := time.Since(startTime).Milliseconds()
		fmt.Printf("RESULT|%s|ABANDON|%d|%dms\n", destAddr, maxRetries, elapsed)
		return
	}

	if currentTry > 0 {
		fmt.Printf("Retry %d/%d pour %s\n", currentTry, maxRetries, destAddr)
	}

	listStr, err := node.GetNodesList()
	if err != nil {
		fmt.Println("Erreur récupération liste:", err)
		return
	}
	if listStr == "LIST:empty" {
		fmt.Println("Aucun node disponible sur le réseau")
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
		fmt.Println("Pas assez de nodes pour construire une route (besoin d'au moins 1 relais)")
		return
	}

	for i := len(candidates) - 1; i > 0; i-- {
		j := mrand.Intn(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}

	relays := candidates[:numRelays]
	route := append(relays, destAddr)
	fmt.Printf("Route forward : %v\n", route)

	var returnRoute []string
	for i := len(relays) - 1; i >= 0; i-- {
		returnRoute = append(returnRoute, relays[i])
	}
	returnRoute = append(returnRoute, nodeAddr)
	fmt.Printf("Route retour:  %v\n", returnRoute)

	onion, msgID, firstNackID, err := Encapsulator_func(message, route, returnRoute, publicKeys, serverAddr, nodeAddr)
	if err != nil {
		fmt.Println("Erreur encapsulation:", err)
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
		fmt.Println("Erreur envoi:", err)
		node.Mu.Lock()
		delete(node.PendingACKs, msgID)
		delete(node.PendingACKs, firstNackID)
		node.Mu.Unlock()
		SendWithRetry(node, serverAddr, destAddr, message, numRelays, publicKeys, maxRetries, currentTry+1, startTime)
		return
	}

	fmt.Printf("Message envoyé (msgID: %s), attente ACK...\n\n", msgID)

	go func(id string, nackID string, ch chan bool) {
		select {
		case success := <-ch:
			elapsed := time.Since(startTime).Milliseconds()
			if success {
				fmt.Printf("ACK confirmé pour %s\n\n", id)
				fmt.Printf("RESULT|%s|ACK|%d|%dms\n", destAddr, currentTry, elapsed)
				Telemetry(destAddr, nodeAddr, "ACK", "", exid, 1)
			} else {
				fmt.Printf("NACK reçu pour %s — retry...\n\n", msgID)
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
			fmt.Printf("Timeout du ACK pour %s\n\n", id)
			Telemetry(route[0], nodeAddr, "NACK", "timeout", exid, 1)
			node.Mu.Lock()
			delete(node.PendingACKs, id)
			delete(node.PendingACKs, nackID)
			node.Mu.Unlock()
		}
	}(msgID, firstNackID, ackChan)
}
