package main

import (
	"bufio"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"project/node_server/model"
)

// fragChunkSize: large enough for typical messages, small enough to cap per-hop TCP writes
const fragChunkSize = 4096

// splitMessage splits msg into fixed-size byte chunks for fragmented sending
func splitMessage(msg string, size int) []string {
	var chunks []string
	for len(msg) > 0 {
		if len(msg) <= size {
			chunks = append(chunks, msg)
			break
		}
		chunks = append(chunks, msg[:size])
		msg = msg[size:]
	}
	return chunks
}

func runCLI(node *model.Node, serverAddr string, publicKeys *KeyCache) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("  FETCH:<ip>:<port>                              - Récupérer la clé publique d'un noeud")
	fmt.Println("  MSG:<ip>:<port>:<message>                      - Message direct")
	fmt.Println("  RELAY:<ip>:<port>,<ip>:<port>,...,<message>    - Relai multi-hop (route manuelle)")
	fmt.Println("  SEND:<nbr>:<ip>:<port>:<message>               - Envoi auto (route aléatoire)")
	fmt.Println("  REGEN:                                         - Régénère la clé RSA du noeud")
	fmt.Println("  SSEND:<grp>:<nbr>:<ip>:<port>:<message>        - Envoi super-node (broadcast enc)")
	fmt.Println("  SBENCH:<msgs>:<grp>:<nbr>:<retries>:<ip>:<port> - Bench super-node")
	fmt.Println("  QUIT:                                          - Quitter")
	fmt.Println("  LIST:                                          - Afficher la liste des noeuds enregistrés")
	fmt.Println()

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.SplitN(input, ":", 2)
		cmd := strings.ToUpper(parts[0])

		var data string
		if len(parts) > 1 {
			data = parts[1]
		}

		switch cmd {

		case "FETCH":
			targetAddr := data
			conn, err := net.Dial("tcp", targetAddr)
			if err != nil {
				fmt.Println("Erreur connexion:", err)
				continue
			}
			if _, err := conn.Write([]byte("GET_PUBKEY\n")); err != nil {
				fmt.Println("Erreur FETCH write:", err)
				conn.Close()
				continue
			}
			reader := bufio.NewReader(conn)
			pubBase64, err := reader.ReadString('\n')
			conn.Close()
			if err != nil {
				fmt.Println("Erreur FETCH read:", err)
				continue
			}

			pubBytes, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(pubBase64))
			pubKeyInterface, err := x509.ParsePKIXPublicKey(pubBytes)
			if err != nil {
				fmt.Println("Erreur decodage clé:", err)
				continue
			}
			if pubKey, ok := pubKeyInterface.(*rsa.PublicKey); ok {
				publicKeys.set(targetAddr, CachedKey{
					Key:       pubKey,
					ExpiresAt: time.Now().Add(30 * time.Second),
				})
				fmt.Printf("Enregistrement de la clé (publique) de %s réalisé avec succès!\n", targetAddr)
			}

		case "MSG":
			subParts := strings.SplitN(data, ":", 3)
			if len(subParts) < 3 {
				fmt.Println("Invalid MSG format. Use MSG:<port>:<message>")
				continue
			}
			dstAddr := subParts[0] + ":" + subParts[1]
			msg := subParts[2]
			nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)
			onion, _, _, err := Encapsulator_func(msg, "", []string{dstAddr}, nil, publicKeys, serverAddr, nodeAddr)
			if err != nil {
				fmt.Println("Erreur Encapsulator_func:", err)
				continue
			}
			err = node.SendTo(dstAddr, onion)
			if err != nil {
				fmt.Println("Error sending message:", err)
			}

		case "RELAY":
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
			nodeAddr := fmt.Sprintf("%s:%d", node.NodeIP, node.Port)
			onion, _, _, err := Encapsulator_func(message, "", route, nil, publicKeys, serverAddr, nodeAddr)
			if err != nil {
				fmt.Println("Erreur Encapsulator_func:", err)
				continue
			}
			err = node.SendTo(route[0], onion)
			if err != nil {
				fmt.Println("Erreur:", err)
			}

		case "SEND":
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
			destAddr := subParts[1] + ":" + subParts[2]
			message := subParts[3]
			// fragment large messages; each chunk is sent as an independent onion
			if len(message) <= fragChunkSize {
				go SendWithRetry(node, serverAddr, destAddr, message, "", numRelays, publicKeys, 3, 0, time.Now())
			} else {
				fragID := model.GenerateMsgID("frag")
				chunks := splitMessage(message, fragChunkSize)
				for i, chunk := range chunks {
					frag := fmt.Sprintf("%s:%d/%d", fragID, i+1, len(chunks))
					go SendWithRetry(node, serverAddr, destAddr, chunk, frag, numRelays, publicKeys, 3, 0, time.Now())
				}
			}

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
				go SendWithRetry(node, serverAddr, destAddr, msg, "", numRelays, publicKeys, maxRetries, 0, time.Now())
				time.Sleep(500 * time.Millisecond)
			}

		case "LIST":
			list, err := node.GetNodesList()
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println(list)
			}

		case "REGEN":
			err := node.RegenerateKeys()
			if err != nil {
				fmt.Println("Erreur lors de la régénération:", err)
			}

		case "SSEND":
			subParts := strings.SplitN(data, ":", 5)
			if len(subParts) < 5 {
				fmt.Println("Format: SSEND:<group_size>:<nbr_relays>:<ip>:<port>:<message>")
				continue
			}
			groupSize, _ := strconv.Atoi(subParts[0])
			numRelays, _ := strconv.Atoi(subParts[1])
			destAddr := subParts[2] + ":" + subParts[3]
			message := subParts[4]
			go SendWithRetrySuper(node, serverAddr, destAddr, message, numRelays, groupSize, publicKeys, 3, 0, time.Now(), []string{})

		case "SBENCH":
			subParts := strings.SplitN(data, ":", 6)
			if len(subParts) < 6 {
				fmt.Println("Format: SBENCH:<nbr_messages>:<group_size>:<nbr_relays>:<maxRetries>:<ip>:<port>")
				continue
			}
			nbrMsg, _ := strconv.Atoi(subParts[0])
			groupSize, _ := strconv.Atoi(subParts[1])
			numRelays, _ := strconv.Atoi(subParts[2])
			maxRetries, _ := strconv.Atoi(subParts[3])
			destAddr := subParts[4] + ":" + subParts[5]
			for i := 0; i < nbrMsg; i++ {
				msg := fmt.Sprintf("sbench-msg-%d", i)
				go SendWithRetrySuper(node, serverAddr, destAddr, msg, numRelays, groupSize, publicKeys, maxRetries, 0, time.Now(), []string{})
				time.Sleep(500 * time.Millisecond)
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
