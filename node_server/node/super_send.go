package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log/slog"
	"project/node_server/model"
	"strconv"
	"strings"
	"time"
)

const (
	WeightAvailability = 0.5
	WeightNetwork      = 0.5
	TargetClusterScore = 150
	MaxNodesPerCluster = 5
	MinClusters        = 3
)

type LayerGroup struct {
	Addrs   []string
	PubKeys []*rsa.PublicKey
}

func parsePublicKey(keyB64 string) *rsa.PublicKey {
	pubBytes, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		return nil
	}
	rsaKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil
	}
	return rsaKey
}

func PickLayer(addrs []string, keys []*rsa.PublicKey, groupSize int) (LayerGroup, []string, []*rsa.PublicKey) {
	if groupSize > len(addrs) {
		groupSize = len(addrs)
	}
	return LayerGroup{Addrs: addrs[:groupSize], PubKeys: keys[:groupSize]},
		addrs[groupSize:], keys[groupSize:]
}

func encryptOnionLayersGroup(
	innerLayer *model.OnionLayer,
	route []LayerGroup,
	senderAddr string,
	nackArray []string,
) (string, error) {
	innerStr := innerLayer.OnionlayerToString()
	payload, err := model.BroadcastEncrypt([]byte(innerStr), route[len(route)-1].PubKeys)
	if err != nil {
		return "", err
	}
	for i := len(route) - 2; i >= 0; i-- {
		prev := senderAddr
		if i > 0 {
			prev = model.JoinAddresses(route[i-1].Addrs)
		}
		layer := &model.OnionLayer{
			Type:  "RELAY",
			MsgID: nackArray[i] + ":" + nackArray[i+1],
			Next:  model.JoinAddresses(route[i+1].Addrs),
			From:  prev,
			Data:  payload,
		}
		payload, err = model.BroadcastEncrypt([]byte(layer.OnionlayerToString()), route[i].PubKeys)
		if err != nil {
			return "", err
		}
	}
	return payload, nil
}

func Encapsulator_func_super(
	message string,
	route []LayerGroup,
	returnRoute []LayerGroup,
	senderAddr string,
) (string, string, string, error) {
	msgID := model.GenerateMsgID()
	nackArray := []string{}
	for range route {
		nackArray = append(nackArray, model.GenerateMsgID("nack"))
	}

	var returnOnion string
	var firstReturnHop string

	if returnRoute != nil && len(returnRoute) > 0 {
		firstReturnHop = model.JoinAddresses(returnRoute[0].Addrs)
		innerACK := &model.OnionLayer{Type: "ACK", MsgID: msgID}
		invNack := []string{}
		for i := range route {
			invNack = append(invNack, nackArray[len(nackArray)-i-1])
		}
		var err error
		returnOnion, err = encryptOnionLayersGroup(innerACK, returnRoute, model.JoinAddresses(returnRoute[len(returnRoute)-1].Addrs), invNack)
		if err != nil {
			return "", "", "", err
		}
	}

	var innerLayer *model.OnionLayer
	if returnRoute != nil {
		innerLayer = &model.OnionLayer{
			Type: "FINAL", MsgID: msgID,
			Next: firstReturnHop, Data: returnOnion, Message: message,
		}
	} else {
		innerLayer = &model.OnionLayer{Type: "FINAL", MsgID: msgID, Message: message}
	}

	forwardPayload, err := encryptOnionLayersGroup(innerLayer, route, senderAddr, nackArray)
	if err != nil {
		return "", "", "", err
	}
	return forwardPayload, msgID, nackArray[0], nil
}

// TODO: implement multi-path routing -- send via 2 independent routes simultaneously, deliver on first ACK
func SendWithRetrySuper(
	node *model.Node,
	serverAddr string,
	destAddr string,
	message string,
	numHops int,
	groupSize int,
	publicKeys *KeyCache,
	maxRetries int,
	currentTry int,
	startTime time.Time,
	failedNodes []string,
) {
	if currentTry >= maxRetries {
		slog.Warn("abandon after max retries", "dest", destAddr, "tries", maxRetries)
		elapsed := time.Since(startTime).Milliseconds()
		fmt.Printf("RESULT_SUPER|%s|ABANDON|%d|%dms\n", destAddr, maxRetries, elapsed)
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
	nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)

	var candidates []model.NodeInfo
	for _, entry := range entries {
		fields := strings.Split(entry, "|")
		if len(fields) < 6 {
			continue
		}
		port, _ := strconv.Atoi(fields[2])
		sa, _ := strconv.Atoi(fields[4])
		sn, _ := strconv.Atoi(fields[5])
		n := model.NodeInfo{
			Name:              fields[0],
			Ip:                fields[1],
			Port:              port,
			PublicKey:         fields[3],
			AvailabilityScore: sa,
			NetworkScore:      sn,
		}
		addr := fmt.Sprintf("%s:%d", n.Ip, n.Port)
		if addr == nodeAddr || addr == destAddr {
			continue
		}
		if _, ok := publicKeys.get(addr); !ok {
			key := parsePublicKey(n.PublicKey)
			if key == nil {
				slog.Warn("invalid public key, skipping", "addr", addr)
				continue
			}
			publicKeys.set(addr, CachedKey{Key: key, ExpiresAt: time.Now().Add(1 * time.Minute)})
		}
		candidates = append(candidates, n)
	}

	if numHops < MinClusters {
		slog.Warn("numHops below MinClusters, adjusted", "numHops", numHops, "min", MinClusters)
		numHops = MinClusters
	}
	if len(candidates) < numHops {
		slog.Warn("not enough nodes for requested hops", "candidates", len(candidates), "hops", numHops)
		return
	}

	relayGroups, reliability := BuildSmartClusters(candidates, numHops, failedNodes)
	reliabilityPercent := (reliability / TargetClusterScore) * 100
	if reliabilityPercent > 100 {
		reliabilityPercent = 100
	}
	slog.Info("route built", "reliability_pct", reliabilityPercent)

	destKey, err := FetchKeyFromServer(destAddr, serverAddr)
	if err != nil {
		slog.Error("destination key not found", "dest", destAddr, "err", err)
		return
	}
	destGroup := LayerGroup{Addrs: []string{destAddr}, PubKeys: []*rsa.PublicKey{destKey}}

	route := append(relayGroups, destGroup)

	node.KeyMu.RLock()
	senderPub := node.PublicKey
	node.KeyMu.RUnlock()
	senderGroup := LayerGroup{Addrs: []string{nodeAddr}, PubKeys: []*rsa.PublicKey{senderPub}}

	var returnRoute []LayerGroup
	for i := len(relayGroups) - 1; i >= 0; i-- {
		returnRoute = append(returnRoute, relayGroups[i])
	}
	returnRoute = append(returnRoute, senderGroup)

	onion, msgID, firstNackID, err := Encapsulator_func_super(message, route, returnRoute, nodeAddr)
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
	go func(r []LayerGroup) {
		Telemetry(nodeAddr, r[0].Addrs[0], "SSEND", message, exid, 1)
		for i := 0; i < len(r)-1; i++ {
			time.Sleep(50 * time.Millisecond)
			Telemetry(r[i].Addrs[0], r[i+1].Addrs[0], "RELAY", "", exid, i+2)
		}
	}(append([]LayerGroup{}, route...))

	firstGroup := route[0]
	sent := false
	for _, addr := range firstGroup.Addrs {
		if node.SendTo(addr, onion) == nil {
			sent = true
			break
		}
		slog.Warn("candidate unreachable, blacklisting", "addr", addr)
		failedNodes = append(failedNodes, addr)
	}
	if !sent {
		slog.Error("all first-group nodes offline")
		node.Mu.Lock()
		delete(node.PendingACKs, msgID)
		delete(node.PendingACKs, firstNackID)
		node.Mu.Unlock()
		SendWithRetrySuper(node, serverAddr, destAddr, message, numHops, groupSize, publicKeys, maxRetries, currentTry+1, startTime, failedNodes)
		return
	}

	slog.Info("message sent, waiting ACK", "msgID", msgID)

	go func(id string, nackID string, ch chan bool) {
		select {
		case success := <-ch:
			elapsed := time.Since(startTime).Milliseconds()
			if success {
				slog.Info("ACK received", "msgID", id)
				fmt.Printf("RESULT_SUPER|%s|ACK|%d|%dms\n", destAddr, currentTry, elapsed)
				Telemetry(destAddr, nodeAddr, "ACK", "", exid, 1)
			} else {
				slog.Warn("NACK received, retrying", "msgID", id)
				Telemetry(route[0].Addrs[0], nodeAddr, "NACK", "", exid, 1)
				node.Mu.Lock()
				delete(node.PendingACKs, id)
				delete(node.PendingACKs, nackID)
				node.Mu.Unlock()
				SendWithRetrySuper(node, serverAddr, destAddr, message, numHops, groupSize, publicKeys, maxRetries, currentTry+1, startTime, failedNodes)
			}
		case <-time.After(8 * time.Second):
			elapsed := time.Since(startTime).Milliseconds()
			slog.Warn("ACK timeout", "msgID", id)
			fmt.Printf("RESULT_SUPER|%s|TIMEOUT|%d|%dms\n", destAddr, currentTry, elapsed)
			Telemetry(route[0].Addrs[0], nodeAddr, "NACK", "timeout", exid, 1)
			node.Mu.Lock()
			delete(node.PendingACKs, id)
			delete(node.PendingACKs, nackID)
			node.Mu.Unlock()
		}
	}(msgID, firstNackID, ackChan)
}
