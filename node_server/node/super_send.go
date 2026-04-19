package main

import (
	"crypto/rsa"
	"fmt"
	// mrand "math/rand"
	"project/node_server/model"
	"strings"
	"time"
	"sort"
	"encoding/base64"
	"crypto/x509"
	"strconv"
)

//Constantes
const (
    WeightAvailability = 0.5 // poids w1
    WeightNetwork = 0.5 // poids w2
    TargetClusterScore = 150  // Score cumulé visé par cluster
    MaxNodesPerCluster = 5 // K (limite pour éviter des oignons trop gros)
    MinClusters = 3 // valeur minimale requise : 3 pour l'anonymat
)

// LayerGroup represents a set of relay addresses and their associated public keys.
type LayerGroup struct {
	Addrs   []string
	PubKeys []*rsa.PublicKey
}

// Fonction qui décode de la base64 vers []byte puis parse en PKIX.

func parsePublicKey(keyB64 string) *rsa.PublicKey {
    pubBytes, _ := base64.StdEncoding.DecodeString(keyB64)
    pubKey, _ := x509.ParsePKIXPublicKey(pubBytes)
    return pubKey.(*rsa.PublicKey)
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
	groupSize int, //obsolète et remplacé par MaxNodesPerCluster
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

	//récup échantillon du réseau
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

	// var candAddrs []string
	// var candKeys []*rsa.PublicKey

	// for _, entry := range entries {
	// 	fields := strings.SplitN(entry, "|", 4)
	// 	if len(fields) < 4 {
	// 		continue
	// 	}
	// 	addr := fields[1] + ":" + fields[2]
	// 	if addr == nodeAddr || addr == destAddr {
	// 		continue
	// 	}
	// 	cached, ok := publicKeys[addr]
	// 	if !ok || time.Now().After(cached.ExpiresAt) {
	// 		key, err := FetchKeyFromServer(addr, serverAddr)
	// 		if err != nil {
	// 			continue
	// 		}
	// 		publicKeys[addr] = CachedKey{Key: key, ExpiresAt: time.Now().Add(30 * time.Second)}
	// 		cached = publicKeys[addr]
	// 	}
	// 	candAddrs = append(candAddrs, addr)
	// 	candKeys = append(candKeys, cached.Key)
	// }


	var candidates []model.NodeInfo
	for _, entry := range entries {
		fields := strings.Split(entry, "|")
		if len(fields) < 6 {
			continue
		}

		//on extraie les données dont les deux scores
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
		if addr == nodeAddr || addr == destAddr { //on exclut l'expéditeur et la destination de la liste des relais
			continue
		}

		//on met a jour le cache des clés publiques (pour le clustering)
		if _, ok := publicKeys[addr]; !ok {
			key := parsePublicKey(n.PublicKey)
			publicKeys[addr] = CachedKey{Key: key, ExpiresAt: time.Now().Add(1 * time.Minute)}
		}
		candidates = append(candidates, n)
	}

	if len(candidates) < numHops {
		fmt.Printf("Pas assez de nœuds (%d) pour %d hops\n", len(candidates), numHops)
		return
	}

	relayGroups, reliability := BuildSmartClusters(candidates, numHops, publicKeys)
	// Calcul et affichage de la fiabilité globale de la route (pour débug et tests)
	reliabilityPercent := (reliability / TargetClusterScore) * 100
	if reliabilityPercent > 100 {
		reliabilityPercent = 100
	}
	fmt.Printf("Route construite. Fiabilité estimée : %.1f%%\n", reliabilityPercent)

	//on prépare la destination et la route retour
	destKey, err := FetchKeyFromServer(destAddr, serverAddr)
	if err != nil {
		fmt.Println("Erreur : Clé de destination introuvable.")
		return
	}
	destGroup := LayerGroup{Addrs: []string{destAddr}, PubKeys: []*rsa.PublicKey{destKey}}

	route := append(relayGroups, destGroup) // Route Forward : [RelayGroups...] + DestGroup
 
	node.KeyMu.RLock()
	senderPub := node.PublicKey
	node.KeyMu.RUnlock()
	// Route Retour : Reverse(RelayGroups...) + SenderGroup
	senderGroup := LayerGroup{Addrs: []string{nodeAddr}, PubKeys: []*rsa.PublicKey{senderPub}}

	var returnRoute []LayerGroup
	for i := len(relayGroups) - 1; i >= 0; i-- {
		returnRoute = append(returnRoute, relayGroups[i])
	}
	returnRoute = append(returnRoute, senderGroup)


	// // fetch dest key
	// cachedDest, ok := publicKeys[destAddr]
	// if !ok || time.Now().After(cachedDest.ExpiresAt) {
	// 	key, err := FetchKeyFromServer(destAddr, serverAddr)
	// 	if err != nil {
	// 		fmt.Println("Erreur clé destination:", err)
	// 		return
	// 	}
	// 	publicKeys[destAddr] = CachedKey{Key: key, ExpiresAt: time.Now().Add(30 * time.Second)}
	// 	cachedDest = publicKeys[destAddr]
	// }

	// // shuffle all candidates
	// perm := mrand.Perm(len(candAddrs))
	// shuffledAddrs := make([]string, len(candAddrs))
	// shuffledKeys := make([]*rsa.PublicKey, len(candKeys))
	// for i, j := range perm {
	// 	shuffledAddrs[i] = candAddrs[j]
	// 	shuffledKeys[i] = candKeys[j]
	// }

	// // build groups for each hop
	// remaining := shuffledAddrs
	// remainingKeys := shuffledKeys
	// var relayGroups []LayerGroup
	// for h := 0; h < numHops; h++ {
	// 	if len(remaining) == 0 {
	// 		break
	// 	}
	// 	var g LayerGroup
	// 	g, remaining, remainingKeys = PickLayer(remaining, remainingKeys, groupSize)
	// 	relayGroups = append(relayGroups, g)
	// }

	// // dest = single node group
	// destGroup := LayerGroup{Addrs: []string{destAddr}, PubKeys: []*rsa.PublicKey{cachedDest.Key}}

	// // forward route: [relayGroups..., destGroup]
	// route := append(relayGroups, destGroup)

	// // return route: reverse relay groups + sender
	// node.KeyMu.RLock()
	// senderPub := node.PublicKey
	// node.KeyMu.RUnlock()
	// senderGroup := LayerGroup{Addrs: []string{nodeAddr}, PubKeys: []*rsa.PublicKey{senderPub}}

	// var returnRoute []LayerGroup
	// for i := len(relayGroups) - 1; i >= 0; i-- {
	// 	returnRoute = append(returnRoute, relayGroups[i])
	// }
	// returnRoute = append(returnRoute, senderGroup)

	// fmt.Printf("Route forward (super) : ")
	// for _, g := range route {
	// 	fmt.Printf("%v ", g.Addrs)
	// }
	// fmt.Println()
	// fmt.Printf("Route retour (super)  : ")
	// for _, g := range returnRoute {
	// 	fmt.Printf("%v ", g.Addrs)
	// }
	// fmt.Println()

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
		//nouvelle tentative avec une nouvelle route
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

// Fonction qui calcule le score GLOBAL d'un noeud : S_node = (w1 * Sa) + (w2 * Sn)
func calculateNodeScore(n model.NodeInfo) float64 {
    return (float64(n.AvailabilityScore) * WeightAvailability) + (float64(n.NetworkScore) * WeightNetwork)
}

// Fonction qui trie les noeuds du meilleur au moins bon (en fonction du score)
func sortNodesByScore(nodes []model.NodeInfo) []model.NodeInfo {
    sort.Slice(nodes, func(i, j int) bool {
        return calculateNodeScore(nodes[i]) > calculateNodeScore(nodes[j])
    })
    return nodes
}

// Fonction qui construit les clusters par ancrage puis remplissage (voir wiki)
func BuildSmartClusters(nodes []model.NodeInfo, numHops int, publicKeys map[string]CachedKey) ([]LayerGroup, float64) {
    sortedNodes := sortNodesByScore(nodes)
    clusters := make([]LayerGroup, numHops)
    clusterScores := make([]float64, numHops)

    //On place les meilleurs noeuds en tête de chaque cluster (ancrage)
    nodeIdx := 0
    for i := 0; i < numHops && nodeIdx < len(sortedNodes); i++ {
        n := sortedNodes[nodeIdx]
        clusters[i].Addrs = append(clusters[i].Addrs, n.Ip+":"+strconv.Itoa(n.Port))
        
        pubKey := parsePublicKey(n.PublicKey) //on convertie la clé string en rsa.PublicKey
        clusters[i].PubKeys = append(clusters[i].PubKeys, pubKey)
        clusterScores[i] += calculateNodeScore(n)
        nodeIdx++
    }

    //on distribue le reste selon le score le plus faible (remplissage)
    for nodeIdx < len(sortedNodes) {
        //on cherche le cluster avec score global le + bas
        targetIdx := 0
        minScore := clusterScores[0]
        for i, s := range clusterScores {
            if s < minScore {
                minScore = s
                targetIdx = i
            }
        }

        //vérif des conditions d'arret (si cluster dépasse score max ou nb de noeud max)
        if clusterScores[targetIdx] >= float64(TargetClusterScore) || len(clusters[targetIdx].Addrs) >= MaxNodesPerCluster {
            break
        }

        //ajout du noeud volatile
        n := sortedNodes[nodeIdx]
        clusters[targetIdx].Addrs = append(clusters[targetIdx].Addrs, n.Ip+":"+strconv.Itoa(n.Port))
        clusters[targetIdx].PubKeys = append(clusters[targetIdx].PubKeys, parsePublicKey(n.PublicKey))
        clusterScores[targetIdx] += calculateNodeScore(n)
        nodeIdx++
    }

    //avertissement si le score final est bas (f° en "mode dégradé")
    for i, s := range clusterScores {
        if s < float64(TargetClusterScore)*0.5 {
            fmt.Printf(" [!] Warning: Cluster %d is weak (Score: %.1f)\n", i+1, s)
        }
    }
   
	var totalScore float64
    for _, s := range clusterScores {
        totalScore += s
    }
    avgScore := totalScore / float64(numHops)
    return clusters, avgScore

}