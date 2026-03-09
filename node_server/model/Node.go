package model

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64" //Ce package va servir a stoker les clés (pour faire la diff entre \n et un octet qui prendrais la valeur associé à \n, idem pour ":")
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type Node struct {
	ID         string
	Port       int
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Listener   net.Listener
}

// fonction quasi-reprise de l'exemple : https://pkg.go.dev/crypto/cipher#NewGCM
func EncryptAES(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	//pour info : https://pkg.go.dev/crypto/cipher#pkg-types
	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	//notre ciphertext est la concaténation de : [ le nonce (K octets) ] + [ msg chiffré ] + [ tag (une sorte de checksum pr l'intégrité)].
	return ciphertext, nil
}

// fonction quasi-reprise de l'exemple : https://pkg.go.dev/crypto/cipher#NewGCM
func DecryptAES(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize() //on recup la taille du nonce

	nonce := ciphertext[:nonceSize] //pour ensuite pouvoir séparer nonce et message
	Ciphertext_real := ciphertext[nonceSize:]

	//déchiffrement (et vérif d'intégrité d'ailleur aussi grâce au tag)
	plaintext, err := aesgcm.Open(nil, nonce, Ciphertext_real, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, err
}

func (n *Node) StartNode() {
	fmt.Printf("[%s] Started in port : %d\n", n.ID, n.Port)
	for {
		conn, err := n.Listener.Accept()
		if err != nil {
			return
		}

		go n.handlerroutine(conn)
	}

}

// ///
func (n *Node) GetNodesList() (string, error) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.Write([]byte("GET_LIST\n"))

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

////

func (n *Node) handlerroutine(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, _ := reader.ReadString('\n')

	line = strings.TrimSpace(line)
	if line == "GET_PUBKEY" {
		pubBytes, _ := x509.MarshalPKIXPublicKey(n.PublicKey)
		pubBase64 := base64.StdEncoding.EncodeToString(pubBytes)
		conn.Write([]byte(pubBase64 + "\n"))
		return
	}

	// Séparation clé AES (chiffré via RSA) et payload chiffré via la dite clé AES
	partsAES := strings.SplitN(line, ":", 2)
	if len(partsAES) < 2 {
		return
	}

	// Déchiffrement RSA (pour récup la clé AES) :
	//on doit s'abord décoder la base 64 avant de déchiffrer le message via RSA :
	encKey, err := base64.StdEncoding.DecodeString(partsAES[0])
	if err != nil {
		return
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, n.PrivateKey, encKey, nil)
	if err != nil {
		fmt.Println("Erreur déchiffrement RSA (Clé AES)")
		return
	}

	//déchiffrement AES (pour récup le le payload en clair)
	//on doit s'abord décoder la base 64 avant de déchiffrer le message via AES :
	encPayload, err := base64.StdEncoding.DecodeString(partsAES[1])
	if err != nil {
		return
	}

	decrypted, err := DecryptAES(aesKey, encPayload)
	if err != nil {
		fmt.Println("Erreur déchiffrement AES")
		return
	}

	line_decrypted := string(decrypted)
	parts := strings.SplitN(line_decrypted, ":", 2)

	if len(parts) < 2 {
		return
	}

	cmd := parts[0]
	data := parts[1]

	switch cmd {
	case "MSG":
		// Direct message
		fmt.Printf("[%s] Message reçu: \"%s\"\n", n.ID, data)

	case "RELAY":
		// Format: RELAY:<nextPort>:<rest>
		// <rest> can be "MSG:message" or "RELAY:<port>:..."
		subParts := strings.SplitN(data, ":", 2)
		if len(subParts) < 2 {
			fmt.Printf("[%s] RELAY format invalide\n", n.ID)
			return
		}

		nextPort, err := strconv.Atoi(subParts[0])
		if err != nil {
			fmt.Printf("[%s] Port invalide: %s\n", n.ID, subParts[0])
			return
		}

		payload := subParts[1]
		fmt.Printf("[%s] Relai vers :%d\n", n.ID, nextPort)

		// Send the palyoad
		err = n.SendTo(nextPort, payload)
		if err != nil {
			fmt.Printf("[%s] Erreur relai: %s\n", n.ID, err)
		}
	default:
		fmt.Printf("[%s] Commande inconnue: %s\n", n.ID, cmd)

	}
}

func (n *Node) SendTo(targetPort int, message string) error {

	addr := fmt.Sprintf("localhost:%d", targetPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	defer conn.Close()

	_, err = conn.Write([]byte(message + "\n"))
	return err
}

// Close the node
func (n *Node) Stop() {
	fmt.Printf("[%s] Node stopped\n", n.ID)

	// Send QUIT to server to leave the list
	// TODO: Implement QUIT message to directory server
	conn, err := net.Dial("tcp", "localhost:8080")
	if err == nil {
		msg := fmt.Sprintf("QUIT:%s\n", n.ID)
		conn.Write([]byte(msg))
		conn.Close()
	}

	n.Listener.Close()
	fmt.Printf("[%s] Node stopped\n", n.ID)

}

func (n *Node) JoinServerList(addrlist string) error {
	conn, err := net.Dial("tcp", addrlist)
	if err != nil {
		return err
	}
	defer conn.Close()

	//Pr envoyer la clé publique sur le réseau (format "reconnu" partout)
	// on utilise le format PKIX (encodage en ASN.1 DER).
	//on appelle cette etape la serialisation
	pubBytes, err := x509.MarshalPKIXPublicKey(n.PublicKey)
	if err != nil {
		return fmt.Errorf("erreur sérialisation clé: %v", err)
	}

	//ensuite on utilise la base 64 et pas le binaire pour le pb des \n
	pubBase64 := base64.StdEncoding.EncodeToString(pubBytes)

	// Send: INIT:id:port:key
	msg := fmt.Sprintf("INIT:%s:%d:%s\n", n.ID, n.Port, pubBase64)
	_, err = conn.Write([]byte(msg))
	if err != nil {
		return err
	}

	// READ INIT_ACK
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "INIT_ACK") {
		fmt.Printf("[%s] Registered to directory server\n", n.ID)
		return nil
	}

	return fmt.Errorf("registration failed: %s", response)
}
