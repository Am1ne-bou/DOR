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
	"net"
	"os"
	"project/node_server/model"
	"strconv"
	"strings"
)

func NewNode(id string) (*model.Node, error) {

	// Génération d'une clé privée RSA 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Générer une clé public à partir de la clé privée
	publicKey := privateKey.PublicKey

	addr := fmt.Sprintf(":%d", 0)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &model.Node{
		ID:         id,
		Port:       listener.Addr().(*net.TCPAddr).Port,
		PrivateKey: privateKey,
		PublicKey:  &publicKey,
		Listener:   listener,
	}, nil

}

func FetchKeyFromServer(port int) (*rsa.PublicKey, error) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.Write([]byte(fmt.Sprintf("GET_KEY:%d\n", port)))

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
	// Annuaire local qui a partir d'1 port donne la clé publique (partie qui sera rempli grâce au serveur plus tard)
	publicKeys := make(map[int]*rsa.PublicKey)

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <id>")
		fmt.Println("Exemple: go run main.go node-1")
		return
	}

	id := os.Args[1]

	node, err := NewNode(id)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	go node.StartNode()

	err = node.JoinServerList("localhost:8080")
	if err != nil {
		fmt.Println("Error joining server:", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nCommandes disponibles:")
	fmt.Println("  FETCH:<port>                      - Récupérer la clé publique d'un noeud")
	fmt.Println("  MSG:<port>:<message>              - Message direct")
	fmt.Println("  RELAY:<port>:<port>:...:<message> - Relai multi-hop")
	fmt.Println("  QUIT:                             - Quitter")
	fmt.Println("  LIST:                             - Afficher la liste des noeuds enregistrés")
	fmt.Println()

	for scanner.Scan() {
		input := scanner.Text()
		parts := strings.SplitN(input, ":", 2)
		if len(parts) < 2 {
			fmt.Println("Invalid format. Use MSG:<message> or RELAY:<port>:<message>")
			continue
		}

		cmd := parts[0]
		data := parts[1]

		switch cmd {

		case "FETCH":
			// Format: FETCH:<port>
			// Plus tard on demandera au serveur cette info
			targetPort, err := strconv.Atoi(data)
			if err != nil {
				fmt.Println("Port invalide")
				continue
			}

			//Connexion au nœud avec le port specifiée
			addr := fmt.Sprintf("localhost:%d", targetPort)
			conn, err := net.Dial("tcp", addr)
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
				publicKeys[targetPort] = pubKey
				fmt.Printf("Enregistrement de la clé (publique) du port réalisé avec succée %d!\n", targetPort)
			}

		case "MSG":
			// Format: MSG:<port>:<message>
			subParts := strings.SplitN(data, ":", 2)
			if len(subParts) < 2 {
				fmt.Println("Invalid MSG format. Use MSG:<port>:<message>")
				continue
			}
			dst, err := strconv.Atoi(subParts[0])
			if err != nil {
				fmt.Println("Error parsing destination port:", err)
				continue
			}
			data = subParts[1]

			onion, err := Encapsulator_func(data, []int{dst}, publicKeys)
			if err != nil {
				fmt.Println("Erreur Encapsulator_func:", err)
				continue
			}

			err = node.SendTo(dst, onion)
			if err != nil {
				fmt.Println("Error sending message:", err)
			}
		case "RELAY":
			// Format of data : RELAY:<port1>:<port2>:...:<message>
			//On récup qui qu'il y a après le dernier : càd le msg
			//Tel qu'il est implémenté, si le message contient :, ça ne marche pas
			//(il faudrait parcourir tout les bouts et vérif si c'est un num de port possible
			//ou non et ensuite reconstruire le message en fonction... flemme pour l'instant)

			lastColon := strings.LastIndex(data, ":")
			if lastColon == -1 {
				continue
			}

			portsStr := strings.Split(data[:lastColon], ":")
			message := data[lastColon+1:]

			var route []int //tableau de ports (sous forme d'entier et pas de stp)
			for _, pStr := range portsStr {
				p, _ := strconv.Atoi(pStr)
				route = append(route, p)
			}

			onion, err := Encapsulator_func(message, route, publicKeys)
			if err != nil {
				fmt.Println("Erreur Encapsulator_func:", err)
				continue
			}

			err = node.SendTo(route[0], onion)
			if err != nil {
				fmt.Println("Erreur:", err)
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

func Encapsulator_func(message string, route []int, publicKeys map[int]*rsa.PublicKey) (string, error) {

	//Fetching keys if needed
	for _, port := range route {
		if _, ok := publicKeys[port]; !ok {
			fmt.Println("Key not found searching for it ...")
			key, err := FetchKeyFromServer(port)
			if err != nil {
				return "", fmt.Errorf("error fetching public key for port %d: %v", port, err)
			}
			publicKeys[port] = key
			fmt.Println("Found public key for port ", port)
		}
	}

	currentPayload := "MSG:" + message //encapsulation du mess final

	for i := len(route) - 1; i >= 0; i-- {
		targetPort := route[i]               //on recup le port de noeud
		pubKey, ok := publicKeys[targetPort] //sa clé publique pour le chiffrement
		if !ok {
			return "", fmt.Errorf("clé publique manquante pour le port %d", targetPort)
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

		if i > 0 { //si ce n'est pas le premier saut, il faut mettre un "header" (RELAY:PORT:msg_encrypted)
			currentPayload = fmt.Sprintf("RELAY:%d:%s", targetPort, encoded)
		} else { //si c'est le 1er saut (noyaux du message):
			currentPayload = encoded
		}
	}

	return currentPayload, nil
}
