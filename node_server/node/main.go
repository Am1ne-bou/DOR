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
	"github.com/google/uuid"
	"project/node_server/model"
	"strconv"
	"strings"
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
		ID:         id,
		Port:       listener.Addr().(*net.TCPAddr).Port,
		PrivateKey: privateKey,
		PublicKey:  &publicKey,
		Listener:   listener,
		ServerAddr: serverAddr,
	}, nil

}

func FetchKeyFromServer(addr string, serverAddr string) (*rsa.PublicKey, error) {
	conn, err := model.DialDirectoryServer(serverAddr);
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

func main() {
	publicKeys := make(map[string]*rsa.PublicKey)

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
	fmt.Println("  QUIT:                                          - Quitter")
	fmt.Println("  LIST:                                          - Afficher la liste des noeuds enregistrés")
	fmt.Println()

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text());

		if input == "" {
            continue;
        }

		parts := strings.SplitN(input, ":", 2)
		cmd := strings.ToUpper(parts[0]); //commande marche en minuscule aussi
		
		var data string;
        if len(parts) > 1 {
            data = parts[1];
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
				publicKeys[targetAddr] = pubKey
				fmt.Printf("Enregistrement de la clé (publique) de %s réalisé avec succès!\n", targetAddr)
			}

		case "MSG":
			// Format: MSG:<ip>:<port>:<message>
			subParts := strings.SplitN(data, ":", 3)
			if len(subParts) < 3 {
				fmt.Println("Invalid MSG format. Use MSG:<port>:<message>")
				continue
			}
			dstAddr := subParts[0] + ":" + subParts[1]
			msg := subParts[2]
			onion, err := Encapsulator_func(msg, []string{dstAddr}, publicKeys, serverAddr)
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

			onion, err := Encapsulator_func(message, route, publicKeys, serverAddr)
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

			// Recuperation de la liste des nodes
			listStr, err := node.GetNodesList()
			if err != nil {
				fmt.Println("Erreur récupération liste:", err)
				continue
			}

			if listStr == "LIST:empty" {
				fmt.Println("Aucun node disponible sur le réseau")
				continue
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
				fmt.Printf("Destination %s introuvable dans le réseau\n", destAddr)
				continue
			}

			// Select relays
			if numRelays > len(candidates) {
				numRelays = len(candidates)
			}
			if numRelays == 0 {
				fmt.Println("Pas assez de nodes pour construire une route (besoin d'au moins 1 relais)")
				continue
			}

			// Shuffle candidates
			for i := len(candidates) - 1; i > 0; i-- {
				j := mrand.Intn(i + 1)
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}

			relays := candidates[:numRelays]

			// Build the route : [relays..., dest]
			route := append(relays, destAddr)
			fmt.Printf("Route automatique: %v\n", route)

			// Encapsulate in onion layers and send to the first node
			onion, err := Encapsulator_func(message, route, publicKeys, serverAddr)
			if err != nil {
				fmt.Println("Erreur encapsulation:", err)
				continue
			}

			err = node.SendTo(route[0], onion)
			if err != nil {
				fmt.Println("Erreur envoi:", err)
			} else {
				fmt.Println("Message envoyé via route automatique !")
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

		case "QUIT":
			fmt.Println("Shutting down node...")
			node.Stop()
			return

		default:
			fmt.Println("Unknown command. Use MSG or RELAY.")
		}
	}
}

// Encapsulator_func wraps the message in multiple encryption layers
func Encapsulator_func(message string, route []string, publicKeys map[string]*rsa.PublicKey, serverAddr string) (string, error) {

	//Fetching keys if needed
	for _, port := range route {
		if _, ok := publicKeys[port]; !ok {
			fmt.Println("Key not found searching for it ...")
			key, err := FetchKeyFromServer(port, serverAddr)
			if err != nil {
				return "", fmt.Errorf("error fetching public key for %s: %v", port, err)
			}
			publicKeys[port] = key
			fmt.Println("Found public key for ", port)
		}
	}

	currentPayload := "MSG:" + strings.ReplaceAll(uuid.New().String(), ":", "-") + ":" + message

	for i := len(route) - 1; i >= 0; i-- {
		targetAddr := route[i]
		pubKey, ok := publicKeys[targetAddr]
		if !ok {
			return "", fmt.Errorf("clé publique manquante pour %s", targetAddr)
		}

		//chiffrement "duo" (AES puis RSA):

		//Géneration clé AES aléatoire (32 octet)
		aesKey := make([]byte, 32)
		io.ReadFull(rand.Reader, aesKey)

		//chiffrement du mess en clair (payload) avec AES :
		encPayload, _ := model.EncryptAES(aesKey, []byte(currentPayload))

		//chiffrement de la clé AES avec RSA:
		encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, aesKey, nil)
		if err != nil {
			return "", err
		}

		// Format fusion : base64(clé_AES_chiffrée_via_RSA):base64(payload_chiffre_via_AES)
		encodedKey := base64.StdEncoding.EncodeToString(encKey)
		encodedPayload := base64.StdEncoding.EncodeToString(encPayload)
		encoded := encodedKey + ":" + encodedPayload

		if i > 0 { //si ce n'est pas le premier saut, il faut mettre un "header" (RELAY:IP:PORT:msg_encrypted)
			currentPayload = fmt.Sprintf("RELAY:%s:%s", targetAddr, encoded)
		} else { //si c'est le 1er saut (noyaux du message):
			currentPayload = encoded
		}
	}

	return currentPayload, nil
}