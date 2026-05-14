package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
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
		elapsed := time.Since(startTime).Milliseconds()
		fmt.Printf("Abandon après %d tentatives pour %s\n\n", maxRetries, destAddr)
		fmt.Printf("RESULT_SUPER|%s|ABANDON|%d|%dms\n", destAddr, maxRetries, elapsed)
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
		fmt.Println("Aucun node disponible")
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
				fmt.Printf("Clé invalide pour %s, skip\n", addr)
				continue
			}
			publicKeys.set(addr, CachedKey{Key: key, ExpiresAt: time.Now().Add(1 * time.Minute)})
		}
		candidates = append(candidates, n)
	}

	if numHops < MinClusters {
		fmt.Printf("Nombre de hops %d < MinClusters %d, ajusté\n", numHops, MinClusters)
		numHops = MinClusters
	}
	if len(candidates) < numHops {
		fmt.Printf("Pas assez de noeuds (%d) pour %d hops\n", len(candidates), numHops)
		return
	}

	relayGroups, reliability := BuildSmartClusters(candidates, numHops, failedNodes)
	reliabilityPercent := (reliability / TargetClusterScore) * 100
	if reliabilityPercent > 100 {
		reliabilityPercent = 100
	}
	fmt.Printf("Route construite. Fiabilité estimée : %.1f%%\n", reliabilityPercent)

	destKey, err := FetchKeyFromServer(destAddr, serverAddr)
	if err != nil {
		fmt.Println("Erreur : Clé de destination introuvable.")
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
		fmt.Printf("Candidat %s injoignable, ajout à la blacklist\n", addr)
		failedNodes = append(failedNodes, addr)
	}
	if !sent {
		fmt.Println("Erreur envoi: tout le premier groupe offline")
		node.Mu.Lock()
		delete(node.PendingACKs, msgID)
		delete(node.PendingACKs, firstNackID)
		node.Mu.Unlock()
		SendWithRetrySuper(node, serverAddr, destAddr, message, numHops, groupSize, publicKeys, maxRetries, currentTry+1, startTime, failedNodes)
		return
	}

	fmt.Printf("Message envoyé (msgID: %s), attente ACK...\n\n", msgID)

	go func(id string, nackID string, ch chan bool) {
		select {
		case success := <-ch:
			elapsed := time.Since(startTime).Milliseconds()
			if success {
				fmt.Printf("ACK confirmé pour %s\n\n", id)
				fmt.Printf("RESULT_SUPER|%s|ACK|%d|%dms\n", destAddr, currentTry, elapsed)
				Telemetry(destAddr, nodeAddr, "ACK", "", exid, 1)
			} else {
				fmt.Printf("NACK reçu pour %s - retry...\n\n", id)
				Telemetry(route[0].Addrs[0], nodeAddr, "NACK", "", exid, 1)
				node.Mu.Lock()
				delete(node.PendingACKs, id)
				delete(node.PendingACKs, nackID)
				node.Mu.Unlock()
				SendWithRetrySuper(node, serverAddr, destAddr, message, numHops, groupSize, publicKeys, maxRetries, currentTry+1, startTime, failedNodes)
			}
		case <-time.After(8 * time.Second):
			elapsed := time.Since(startTime).Milliseconds()
			fmt.Printf("Timeout ACK pour %s\n\n", id)
			fmt.Printf("RESULT_SUPER|%s|TIMEOUT|%d|%dms\n", destAddr, currentTry, elapsed)
			Telemetry(route[0].Addrs[0], nodeAddr, "NACK", "timeout", exid, 1)
			node.Mu.Lock()
			delete(node.PendingACKs, id)
			delete(node.PendingACKs, nackID)
			node.Mu.Unlock()
		}
	}(msgID, firstNackID, ackChan)
}
