package main

import (
	"crypto/rsa"
	"fmt"
	mrand "math/rand"
	"project/node_server/model"
	"strings"
	"time"
)

// LayerGroup represents a set of relay addresses and their associated public keys.
type LayerGroup struct {
	Addrs   []string
	PubKeys []*rsa.PublicKey
}

// PickLayer selects up to groupSize candidates and returns the selected group plus the remaining candidates.
func PickLayer(addrs []string, keys []*rsa.PublicKey, groupSize int) (LayerGroup, []string, []*rsa.PublicKey) {
	if groupSize > len(addrs) {
		groupSize = len(addrs)
	}
	group := LayerGroup{
		Addrs:   addrs[:groupSize],
		PubKeys: keys[:groupSize],
	}
	// return remaining candidates
	return group, addrs[groupSize:], keys[groupSize:]
}

// encryptOnionLayersGroup encrypts onion layers for a route made of relay groups.
func encryptOnionLayersGroup(
	innerLayer *model.OnionLayer,
	route []LayerGroup,
	senderAddr string,
	nackArray []string,
) (string, error) {
	innerStr := innerLayer.OnionlayerToString()

	// encrypt for last group
	payload, err := model.BroadcastEncrypt([]byte(innerStr), route[len(route)-1].PubKeys)
	if err != nil {
		return "", err
	}

	// layer by layer from end
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

// Encapsulator_func_super builds the forward and return onion payloads for grouped relays.
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
		innerLayer = &model.OnionLayer{
			Type: "FINAL", MsgID: msgID, Message: message,
		}
	}

	forwardPayload, err := encryptOnionLayersGroup(innerLayer, route, senderAddr, nackArray)
	if err != nil {
		return "", "", "", err
	}
	return forwardPayload, msgID, nackArray[0], nil
}

// SendWithRetrySuper sends a message through grouped relays and retries on failure.
func SendWithRetrySuper(
	node *model.Node,
	serverAddr string,
	destAddr string,
	message string,
	numHops int,
	groupSize int,
	publicKeys map[string]CachedKey,
	maxRetries int,
	currentTry int,
	startTime time.Time,
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

	// parse node list + fetch keys
	listData := strings.TrimPrefix(listStr, "LIST:")
	entries := strings.Split(listData, ",")
	nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)

	var candAddrs []string
	var candKeys []*rsa.PublicKey

	for _, entry := range entries {
		fields := strings.SplitN(entry, "|", 4)
		if len(fields) < 4 {
			continue
		}
		addr := fields[1] + ":" + fields[2]
		if addr == nodeAddr || addr == destAddr {
			continue
		}
		cached, ok := publicKeys[addr]
		if !ok || time.Now().After(cached.ExpiresAt) {
			key, err := FetchKeyFromServer(addr, serverAddr)
			if err != nil {
				continue
			}
			publicKeys[addr] = CachedKey{Key: key, ExpiresAt: time.Now().Add(30 * time.Second)}
			cached = publicKeys[addr]
		}
		candAddrs = append(candAddrs, addr)
		candKeys = append(candKeys, cached.Key)
	}

	if len(candAddrs) < numHops {
		fmt.Println("Pas assez de nodes pour la route")
		return
	}

	// fetch dest key
	cachedDest, ok := publicKeys[destAddr]
	if !ok || time.Now().After(cachedDest.ExpiresAt) {
		key, err := FetchKeyFromServer(destAddr, serverAddr)
		if err != nil {
			fmt.Println("Erreur clé destination:", err)
			return
		}
		publicKeys[destAddr] = CachedKey{Key: key, ExpiresAt: time.Now().Add(30 * time.Second)}
		cachedDest = publicKeys[destAddr]
	}

	// shuffle all candidates
	perm := mrand.Perm(len(candAddrs))
	shuffledAddrs := make([]string, len(candAddrs))
	shuffledKeys := make([]*rsa.PublicKey, len(candKeys))
	for i, j := range perm {
		shuffledAddrs[i] = candAddrs[j]
		shuffledKeys[i] = candKeys[j]
	}

	// build groups for each hop
	remaining := shuffledAddrs
	remainingKeys := shuffledKeys
	var relayGroups []LayerGroup
	for h := 0; h < numHops; h++ {
		if len(remaining) == 0 {
			break
		}
		var g LayerGroup
		g, remaining, remainingKeys = PickLayer(remaining, remainingKeys, groupSize)
		relayGroups = append(relayGroups, g)
	}

	// dest = single node group
	destGroup := LayerGroup{Addrs: []string{destAddr}, PubKeys: []*rsa.PublicKey{cachedDest.Key}}

	// forward route: [relayGroups..., destGroup]
	route := append(relayGroups, destGroup)

	// return route: reverse relay groups + sender
	node.KeyMu.RLock()
	senderPub := node.PublicKey
	node.KeyMu.RUnlock()
	senderGroup := LayerGroup{Addrs: []string{nodeAddr}, PubKeys: []*rsa.PublicKey{senderPub}}

	var returnRoute []LayerGroup
	for i := len(relayGroups) - 1; i >= 0; i-- {
		returnRoute = append(returnRoute, relayGroups[i])
	}
	returnRoute = append(returnRoute, senderGroup)

	fmt.Printf("Route forward (super) : ")
	for _, g := range route {
		fmt.Printf("%v ", g.Addrs)
	}
	fmt.Println()
	fmt.Printf("Route retour (super)  : ")
	for _, g := range returnRoute {
		fmt.Printf("%v ", g.Addrs)
	}
	fmt.Println()

	// encapsulate
	onion, msgID, firstNackID, err := Encapsulator_func_super(message, route, returnRoute, nodeAddr)
	if err != nil {
		fmt.Println("Erreur encapsulation:", err)
		return
	}

	// register ACK/NACK channels
	ackChan := make(chan bool, 1)
	node.Mu.Lock()
	node.PendingACKs[msgID] = ackChan
	node.PendingACKs[firstNackID] = ackChan
	node.Mu.Unlock()

	// send to first group, try each candidate
	firstGroup := route[0]
	sent := false
	for _, addr := range firstGroup.Addrs {
		if node.SendTo(addr, onion) == nil {
			sent = true
			break
		}
		fmt.Printf("Candidat %s injoignable\n", addr)
	}
	if !sent {
		fmt.Println("Erreur envoi: tout le premier groupe offline")
		node.Mu.Lock()
		delete(node.PendingACKs, msgID)
		delete(node.PendingACKs, firstNackID)
		node.Mu.Unlock()
		SendWithRetrySuper(node, serverAddr, destAddr, message, numHops, groupSize, publicKeys, maxRetries, currentTry+1, startTime)
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
			} else {
				fmt.Printf("NACK reçu pour %s — retry...\n\n", id)
				node.Mu.Lock()
				delete(node.PendingACKs, id)
				delete(node.PendingACKs, nackID)
				node.Mu.Unlock()
				SendWithRetrySuper(node, serverAddr, destAddr, message, numHops, groupSize, publicKeys, maxRetries, currentTry+1, startTime)
			}
		case <-time.After(8 * time.Second):
			elapsed := time.Since(startTime).Milliseconds()
			fmt.Printf("Timeout ACK pour %s\n\n", id)
			fmt.Printf("RESULT_SUPER|%s|TIMEOUT|%d|%dms\n", destAddr, currentTry, elapsed)
			node.Mu.Lock()
			delete(node.PendingACKs, id)
			delete(node.PendingACKs, nackID)
			node.Mu.Unlock()
		}
	}(msgID, firstNackID, ackChan)
}
