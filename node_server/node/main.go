package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"project/node_server/model"
	"strconv"
	"strings"
	"crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "encoding/base64" //Ce package va servir a stoker les clés (pour faire la diff entre \n et un octet qui prendrais la valeur associé à \n)
	"crypto/x509"
)

func NewNode(id string, port int) (*model.Node, error) {

	// Génération d'une clé privée RSA 16384 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 10000)
    if err != nil {
        return nil, err;
    }

	// Générer une clé public à partir de la clé privée
	publicKey := privateKey.PublicKey


	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &model.Node{
		ID:       id,
		Port:     port,
		PrivateKey: privateKey,
        PublicKey:  &publicKey,
		Listener: listener,
	}, nil

}

func main() {
	// Annuaire local qui a partir d'1 port donne la clé publique (partie qui sera dans le serveur plus tard)
	publicKeys := make(map[int]*rsa.PublicKey);

	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <id> <port>")
		fmt.Println("Exemple: go run main.go node-1 9010")
		return
	}

	id := os.Args[1]
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Error parsing port:", err)
		return
	}

	node, err := NewNode(id, port)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	go node.StartNode()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nCommandes disponibles:")
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
			targetPort, err := strconv.Atoi(data);
			if err != nil {
				fmt.Println("Port invalide");
				continue;
			}

			//Connexion au nœud avec le port specifiée
			addr := fmt.Sprintf("localhost:%d", targetPort);
			conn, err := net.Dial("tcp", addr);
			if err != nil {
				fmt.Println("Erreur connexion:", err);
				continue;
			}

			//on demande sa clé publique
			conn.Write([]byte("GET_PUBKEY\n"));

			//on lit la rép
			reader := bufio.NewReader(conn);
			pubBase64, _ := reader.ReadString('\n');
			conn.Close();

			//on décode la clé qui est en base 64 et encode en x509
			pubBytes,_ := base64.StdEncoding.DecodeString(strings.TrimSpace(pubBase64));
			pubKeyInterface, err := x509.ParsePKIXPublicKey(pubBytes);
			if err != nil {
				fmt.Println("Erreur decodage clé:", err);
				continue;
			}

			// on peut ensuite la stoquer dans le dico annuaire port -> clé pub
			if pubKey, ok := pubKeyInterface.(*rsa.PublicKey);
			ok {
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

			onion, err := AL1(data, []int{dst}, publicKeys)
			if err != nil {
				fmt.Println("Erreur AL1:", err)
				continue
			}

			//TODO : chiffrement du message ici avant envoi via clé publique du dst
			err = node.SendTo(dst, onion)
			if err != nil {
				fmt.Println("Error sending message:", err)
			}
		case "RELAY":
			// Format of data : RELAY:<port1>:<port2>:...:<message>
			// Build the relay chain

			//On récup qui qu'il y a après le dernier : càd le msg (supposons que le message n'a pas de :)
			lastColon :=strings.LastIndex(data, ":");
    		if lastColon == -1 {
				continue
			}

			portsStr := strings.Split(data[:lastColon], ":");
    		message := data[lastColon+1:];


			var route []int; //tableau de ports (sous forme d'entier et pas de stp)
			for _, pStr := range portsStr {
				p, _ := strconv.Atoi(pStr);
				route = append(route, p);
			}

			onion, err := AL1(message, route, publicKeys)
			if err != nil {
				fmt.Println("Erreur AL1:", err)
				continue
			}

			err = node.SendTo(route[0], onion)
			if err != nil {
			fmt.Println("Erreur:", err)
			}

			//ancien code :
			// subParts := strings.SplitN(data, ":", 2)
			// if len(subParts) < 2 {
			// 	fmt.Println("Format: RELAY:<port>:<port>:...:<message>")
			// 	continue
			// }
			// firstPort, err := strconv.Atoi(subParts[0])
			// if err != nil {
			// 	fmt.Println("Port invalide")
			// 	continue
			// }
			// rest := subParts[1]

			// // Build the relay chain message
			// msgToSend := buildRelayChain(rest)
			// err = node.SendTo(firstPort, msgToSend)
			// if err != nil {
			// 	fmt.Println("Erreur:", err)
			// }

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

func buildRelayChain(data string) string {
	//TODO : plus besoin de cette fonction comme dit sur whatsapp
	parts := strings.SplitN(data, ":", 2)

	if len(parts) < 2 {
		// No more ports, just a message
		return "MSG:" + data
	}

	//Test if parts[0] is a port number (Commentaire : moyen qui peut être amélioré, là on vérifie juste que c'est un nombre)
	_, err := strconv.Atoi(parts[0])
	if err != nil {
		// Not a port, it's a message
		return "MSG:" + data
	}

	//is a port
	port := parts[0]
	rest := parts[1]

	return "RELAY:" + port + ":" + buildRelayChain(rest)
}

func AL1(message string, route []int, publicKeys map[int]*rsa.PublicKey) (string, error) {

    currentPayload := "MSG:" + message; //encapsulation du mess final

    for i := len(route) - 1; i >= 0; i-- {
        targetPort := route[i]; //on recup le port de noeud
        pubKey, ok := publicKeys[targetPort]; //sa clé publique pour le chiffrement
        if !ok {
            return "", fmt.Errorf("clé publique manquante pour le port %d", targetPort);
        }

		//chiffrement :
        encryptedBytes,err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(currentPayload), nil);
        if err != nil {
            return "", err;
        }

        encoded := base64.StdEncoding.EncodeToString(encryptedBytes);

        if i > 0 { //si ce n'est pas le premier saut, il faut mettre un "header" (RELAY:PORT:msg_encrypted)
            currentPayload = fmt.Sprintf("RELAY:%d:%s", targetPort, encoded);
        } else { //si c'est le 1er saut (noyaux du message):
            currentPayload = encoded;
        }
    }

    return currentPayload, nil;
}