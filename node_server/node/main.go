package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64" //Ce package va servir a stoker les clés (pour faire la diff entre \n et un octet qui prendrais la valeur associé à \n), idem pour ":"
	"fmt"
	"io"
	mrand "math/rand"
	"net"
	"os"
	"project/node_server/model"
	"strconv"
	"strings"
	"time"
)

func NewNode(id string, serverAddr string) (*model.Node, error) {

	// Génération d'une clé privée RSA 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Générer une clé public à partir de la clé privée
	publicKey := privateKey.PublicKey

	addr := fmt.Sprintf("0.0.0.0:%d", 0)
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}

	return &model.Node{
		ID:            id,
		Port:          listener.Addr().(*net.TCPAddr).Port,
		PrivateKey:    privateKey,
		PublicKey:     &publicKey,
		Listener:      listener,
		ServerAddr:    serverAddr,
		PendingACKs:   make(map[string]chan bool),
		PendingRelays: make(map[string]model.Nackstruct),
	}, nil

}

func FetchKeyFromServer(addr string, serverAddr string) (*rsa.PublicKey, error) {
	conn, err := model.DialDirectoryServer(serverAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.Write([]byte(fmt.Sprintf("GET_KEY:%s\n", addr)))

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	response = strings.TrimSpace(response)

	if strings.HasPrefix(response, "ERROR:") {
		return nil, fmt.Errorf(response)
	}

	parts := strings.SplitN(response, ":", 2)
	if len(parts) != 2 || parts[0] != "KEY" {
		return nil, fmt.Errorf("invalid response")
	}

	// Base64 to bytes
	publicBytes, _ := base64.StdEncoding.DecodeString(parts[1])
	// bytes to publicKey
	pubKey, _ := x509.ParsePKIXPublicKey(publicBytes)
	return pubKey.(*rsa.PublicKey), nil
}

type CachedKey struct {
	Key       *rsa.PublicKey
	ExpiresAt time.Time
}

func main() {
	publicKeys := make(map[string]CachedKey)

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <id>")
		fmt.Println("Exemple: go run main.go node-1")
		return
	}

	id := os.Args[1]
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:8080"
	}

	node, err := NewNode(id, serverAddr)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	go node.StartNode()

	err = node.JoinServerList(serverAddr)
	if err != nil {
		fmt.Println("Error joining server:", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("  FETCH:<ip>:<port>                              - Récupérer la clé publique d'un noeud")
	fmt.Println("  MSG:<ip>:<port>:<message>                      - Message direct")
	fmt.Println("  RELAY:<ip>:<port>,<ip>:<port>,...,<message>    - Relai multi-hop (route manuelle)")
	fmt.Println("  SEND:<nbr>:<ip>:<port>:<message>              - Envoi auto (route aléatoire)")
	fmt.Println("  REGEN:                                         - Régénère la clé RSA du noeud")
	fmt.Println("  QUIT:                                          - Quitter")
	fmt.Println("  LIST:                                          - Afficher la liste des noeuds enregistrés")
	fmt.Println()

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		parts := strings.SplitN(input, ":", 2)
		cmd := strings.ToUpper(parts[0]) //commande marche en minuscule aussi

		var data string
		if len(parts) > 1 {
			data = parts[1]
		}

		switch cmd {

		case "FETCH":
			targetAddr := data // data contient "ip:port"

			conn, err := net.Dial("tcp", targetAddr)
			if err != nil {
				fmt.Println("Erreur connexion:", err)
				continue
			}

			//on demande sa clé publique
			conn.Write([]byte("GET_PUBKEY\n"))

			//on lit la rép
			reader := bufio.NewReader(conn)
			pubBase64, _ := reader.ReadString('\n')
			conn.Close()

			pubBytes, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(pubBase64)) //on décode le sérial de la clé qui est en base 64
			pubKeyInterface, err := x509.ParsePKIXPublicKey(pubBytes)                    //on décode le sérial en x509 pour retrouver la clé RSA
			if err != nil {
				fmt.Println("Erreur decodage clé:", err)
				continue
			}

			// on peut ensuite la stoquer dans le dico annuaire port -> clé pub
			if pubKey, ok := pubKeyInterface.(*rsa.PublicKey); ok {
				publicKeys[targetAddr] = CachedKey{
					Key:       pubKey,
					ExpiresAt: time.Now().Add(30 * time.Second),
				}
				fmt.Printf("Enregistrement de la clé (publique) de %s réalisé avec succès!\n", targetAddr)
			}

		case "MSG":
			// Format: MSG:<ip>:<port>:<message>
			// NO ACK
			subParts := strings.SplitN(data, ":", 3)
			if len(subParts) < 3 {
				fmt.Println("Invalid MSG format. Use MSG:<port>:<message>")
				continue
			}
			dstAddr := subParts[0] + ":" + subParts[1]
			msg := subParts[2]
			// no ACK returnRoute =nil no need for nack
			nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)
			onion, _, _, err := Encapsulator_func(msg, []string{dstAddr}, nil, publicKeys, serverAddr, nodeAddr)
			if err != nil {
				fmt.Println("Erreur Encapsulator_func:", err)
				continue
			}

			err = node.SendTo(dstAddr, onion)
			if err != nil {
				fmt.Println("Error sending message:", err)
			}
		case "RELAY":
			// Format: RELAY:<ip1>:<port1>,<ip2>:<port2>,...,<message>
			// no ACk
			lastComma := strings.LastIndex(data, ",")
			if lastComma == -1 {
				fmt.Println("Format: RELAY:<ip>:<port>,<ip>:<port>,...,<message>")
				continue
			}

			addrsStr := strings.Split(data[:lastComma], ",")
			message := data[lastComma+1:]

			var route []string
			for _, addr := range addrsStr {
				route = append(route, strings.TrimSpace(addr))
			}
			//no ACK nor nack
			nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)
			onion, _, _, err := Encapsulator_func(message, route, nil, publicKeys, serverAddr, nodeAddr)
			if err != nil {
				fmt.Println("Erreur Encapsulator_func:", err)
				continue
			}

			err = node.SendTo(route[0], onion)
			if err != nil {
				fmt.Println("Erreur:", err)
			}

		case "SEND":
			// Format: SEND:<nbr_relays>:<ip>:<port>:<message>
			// Build an auto route rand from a nbr of relays
			subParts := strings.SplitN(data, ":", 4)
			if len(subParts) < 4 {
				fmt.Println("Format: SEND:<nbr_relays>:<ip>:<port>:<message>")
				continue
			}

			numRelays, err := strconv.Atoi(subParts[0])
			if err != nil {
				fmt.Println("Error parsing relay number:", err)
				continue
			}

			destAddr := subParts[1] + ":" + subParts[2] // ip:port
			message := subParts[3]

			go SendWithRetry(node, serverAddr, destAddr, message, numRelays, publicKeys, 3, 0, time.Now())

		case "BENCH":
			subParts := strings.SplitN(data, ":", 5)
			if len(subParts) < 5 {
				fmt.Println("Format: BENCH:<nbr_messages>:<nbr_relays>:<maxRetries>:<ip>:<port>")
				continue
			}
			nbrMsg, _ := strconv.Atoi(subParts[0])
			numRelays, _ := strconv.Atoi(subParts[1])
			maxRetries, _ := strconv.Atoi(subParts[2])
			destAddr := subParts[3] + ":" + subParts[4]

			for i := 0; i < nbrMsg; i++ {
				msg := fmt.Sprintf("bench-msg-%d", i)
				go SendWithRetry(node, serverAddr, destAddr, msg, numRelays, publicKeys, maxRetries, 0, time.Now())
				time.Sleep(500 * time.Millisecond)
			}

		////
		case "LIST":
			list, err := node.GetNodesList()
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println(list)
			}
		/////

		case "REGEN":
			err := node.RegenerateKeys()
			if err != nil {
				fmt.Println("Erreur lors de la régénération:", err)
			}

		case "QUIT":
			fmt.Println("Shutting down node...")
			node.Stop()
			return

		default:
			fmt.Println("Unknown command. Use MSG or RELAY.")
		}
	}
}

func SendWithRetry(
	node *model.Node,
	serverAddr string,
	destAddr string,
	message string,
	numRelays int,
	publicKeys map[string]CachedKey,
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
	// Recuperation de la liste des nodes
	listStr, err := node.GetNodesList()
	if err != nil {
		fmt.Println("Erreur récupération liste:", err)
		return
	}

	if listStr == "LIST:empty" {
		fmt.Println("Aucun node disponible sur le réseau")
		return
	}

	// Parser la réponse LIST:name|ip|port|key,name|ip|port|key,...
	listData := strings.TrimPrefix(listStr, "LIST:")
	entries := strings.Split(listData, ",")

	var candidates []string // adresses ip:port des candidats
	destFound := false
	nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)

	for _, entry := range entries {
		fields := strings.SplitN(entry, "|", 4)
		if len(fields) < 4 {
			continue
		}
		ip := fields[1]
		port := fields[2]
		addr := ip + ":" + port

		if addr == destAddr {
			destFound = true
		}
		// Exclude this node and the destination from relay candidates
		if addr != nodeAddr && addr != destAddr {
			candidates = append(candidates, addr)
		}
	}
	if !destFound {
		//fmt.Printf("Destination %s introuvable dans le réseau\n", destAddr)
		//continue
	}

	// Select relays
	if numRelays > len(candidates) {
		numRelays = len(candidates)
	}
	if numRelays == 0 {
		fmt.Println("Pas assez de nodes pour construire une route (besoin d'au moins 1 relais)")
		return
	}

	// Shuffle candidates
	for i := len(candidates) - 1; i > 0; i-- {
		j := mrand.Intn(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}

	relays := candidates[:numRelays]

	// Build the route forward : [relays..., dest]
	route := append(relays, destAddr)
	fmt.Printf("Route forward : %v\n", route)

	// Build the return route inverse of the forward
	var returnRoute []string
	for i := len(relays) - 1; i >= 0; i-- {
		returnRoute = append(returnRoute, relays[i])
	}
	returnRoute = append(returnRoute, nodeAddr)
	fmt.Printf("Route retour:  %v\n", returnRoute)

	// Encapsulate in onion layers and send to the first node
	onion, msgID, firstNackID, err := Encapsulator_func(message, route, returnRoute, publicKeys, serverAddr, nodeAddr)
	if err != nil {
		fmt.Println("Erreur encapsulation:", err)
		return
	}

	// save in the chanel before sending
	ackChan := make(chan bool, 1)
	node.Mu.Lock()
	node.PendingACKs[msgID] = ackChan
	node.PendingACKs[firstNackID] = ackChan
	node.Mu.Unlock()

	err = node.SendTo(route[0], onion)
	if err != nil {
		fmt.Println("Erreur envoi:", err)
		node.Mu.Lock()
		delete(node.PendingACKs, msgID)
		node.Mu.Unlock()
		SendWithRetry(node, serverAddr, destAddr, message, numRelays, publicKeys, maxRetries, currentTry+1, startTime)
		return
	}

	fmt.Printf("Message envoyé (msgID: %s), attente ACK...\n\n", msgID)

	// Goroutine to wait for the ACK with timeout
	go func(id string, nackID string, ch chan bool) {
		select {
		case success := <-ch:
			elapsed := time.Since(startTime).Milliseconds()
			if success {
				fmt.Printf("ACK confirmé pour %s\n\n", id)
				fmt.Printf("RESULT|%s|ACK|%d|%dms\n", destAddr, currentTry, elapsed)
			} else {
				fmt.Printf("NACK reçu pour %s — retry...\n\n", msgID)
				node.Mu.Lock()
				delete(node.PendingACKs, msgID)
				delete(node.PendingACKs, firstNackID)
				node.Mu.Unlock()
				SendWithRetry(node, serverAddr, destAddr, message, numRelays, publicKeys, maxRetries, currentTry+1, startTime)
			}
		case <-time.After(time.Second * 8):
			elapsed := time.Since(startTime).Milliseconds()
			fmt.Printf("RESULT|%s|TIMEOUT|%d|%dms\n", destAddr, currentTry, elapsed)
			// timeout
			fmt.Printf("Timeout du ACK pour %s\n\n", id)
			node.Mu.Lock()
			delete(node.PendingACKs, id)
			delete(node.PendingACKs, nackID)
			node.Mu.Unlock()
		}
	}(msgID, firstNackID, ackChan)

}

// Encapsulator_func wraps the message in multiple encryption layers
func Encapsulator_func(
	message string,
	route []string, // [R1, R2, dest]
	returnRoute []string, // [R2, R1, sender] — nil si pas de ACK
	publicKeys map[string]CachedKey,
	serverAddr string,
	senderAddr string,
) (string, string, string, error) { // data,msgid,firstNackId,err

	//Fetching keys if needed
	allNodes := append([]string{}, route...)
	if returnRoute != nil {
		allNodes = append(allNodes, returnRoute...)
	}
	for _, port := range allNodes {
		cached, ok := publicKeys[port]
		if !ok || time.Now().After(cached.ExpiresAt) {
			if !ok {
				fmt.Println("Key not found searching for it ...")
			} else {
				fmt.Println("Key expired, searching for it ...")
			}
			key, err := FetchKeyFromServer(port, serverAddr)
			if err != nil {
				return "", "", "", fmt.Errorf("error fetching public key for %s: %v", port, err)
			}
			publicKeys[port] = CachedKey{
				Key:       key,
				ExpiresAt: time.Now().Add(30 * time.Second),
			}
			fmt.Println("Found public key for ", port)
		}
	}

	msgID := model.GenerateMsgID() //original one seen by src and dst
	// array for the nacks
	nackArray := []string{}
	for _ = range len(route) {
		nackArray = append(nackArray, model.GenerateMsgID("nack"))
	}

	var returnOnion string
	var firstReturnHop string

	if returnRoute != nil && len(returnRoute) > 0 {
		firstReturnHop = returnRoute[0]
		// the most inner layer return ACK for the sender
		innerLayer := &model.OnionLayer{
			Type:  "ACK",
			MsgID: msgID,
		}
		invNackArray := []string{}
		for i := range len(route) {
			invNackArray = append(invNackArray, nackArray[len(nackArray)-i-1])
		}
		// encrypt layer by layer from the sender
		returnPayload, err := encryptOnionLayers(innerLayer, returnRoute, publicKeys, returnRoute[len(returnRoute)-1], invNackArray)
		if err != nil {
			return "", "", "", fmt.Errorf("error building return onion: %v", err)
		}
		//
		returnOnion = returnPayload
	}

	//building the onion of tha payload
	var innerLayer *model.OnionLayer

	if returnRoute != nil {
		innerLayer = &model.OnionLayer{
			Type:    "FINAL",
			MsgID:   msgID,
			Next:    firstReturnHop,
			Data:    returnOnion,
			Message: message,
		}
	} else {
		innerLayer = &model.OnionLayer{
			Type:    "FINAL",
			MsgID:   msgID,
			Message: message,
		}
	}

	// encrypt layer by layer from the destination
	forwardPayload, err := encryptOnionLayers(innerLayer, route, publicKeys, senderAddr, nackArray)
	if err != nil {
		return "", "", "", fmt.Errorf("error building forward onion: %v", err)
	}

	return forwardPayload, msgID, nackArray[0], nil

}

// encryptOnionLayers encrypt an OnionLayer layer by layer from a route
//
// innerLayer = the final node will see (FINAL ou ACK)
// route      = [hop1, hop2, ..., hopN]  — hopN will see innerLayer
//
// Retourne a string encrypted sent to hop1
func encryptOnionLayers(
	innerLayer *model.OnionLayer,
	route []string,
	publicKeys map[string]CachedKey,
	senderAddr string,
	nackArray []string,
) (string, error) {

	// innerlayer to string
	innerLayerString := innerLayer.OnionlayerToString()

	// encrypte the last node
	currentPayload, err := encryptForNode([]byte(innerLayerString), publicKeys[route[len(route)-1]].Key)
	if err != nil {
		return "", err
	}

	// encrypte layer by layer from the end
	// i = len-2 just the middle nodes
	for i := len(route) - 2; i >= 0; i-- {
		prevNode := senderAddr
		if i > 0 {
			prevNode = route[i-1]
		}
		// build the RELAY OnionLayer
		relayLayer := &model.OnionLayer{
			Type:  "RELAY",
			MsgID: nackArray[i] + ":" + nackArray[i+1], // id of the nack between nodes nackidtosend:nackidtoreceive
			Next:  route[i+1],                          // next hop
			From:  prevNode,                            // node before
			Data:  currentPayload,                      // Onion encrypted of the inner layers
		}
		relayLayerString := relayLayer.OnionlayerToString()

		currentPayload, err = encryptForNode([]byte(relayLayerString), publicKeys[route[i]].Key)
		if err != nil {
			return "", err
		}
	}

	return currentPayload, nil
}

// encryptForNode encrypt bytes for a node (AES + RSA)
// Retourn this format "base64(key_RSA):base64(plaintext_AES)"
func encryptForNode(plaintext []byte, pubKey *rsa.PublicKey) (string, error) {
	// Generate a random AES key
	aesKey := make([]byte, 32)
	io.ReadFull(rand.Reader, aesKey)

	// encrypt the plaintext
	encPlaintext, err := model.EncryptAES(aesKey, plaintext)
	if err != nil {
		return "", err
	}

	// Encrypt the the AES key with the RSA key
	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, aesKey, nil)
	if err != nil {
		return "", err
	}

	// Format : base64(key AES encrpted by RSA):base64(plaintext encrypted by AES)
	return base64.StdEncoding.EncodeToString(encKey) + ":" +
		base64.StdEncoding.EncodeToString(encPlaintext), nil
}
