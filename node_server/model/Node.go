package model

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64" //Ce package va servir a stoker les clés (pour faire la diff entre \n et un octet qui prendrais la valeur associé à \n, idem pour ":")
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

//ATTENTION LA LIGNE EN DESSOUS N'EST PAS UN COMMENTAIRE
//go:embed cert.pem
//ATTENTION ne pas enlever les // ou supprimer la ligne au dessus !
//(c'est pour la compil pour intégrer le fichier),

var serverCert []byte

func DialDirectoryServer(addr string) (*tls.Conn, error) {
	certPool := x509.NewCertPool() //liste de certificats (vide pr l'instant)
	certPool.AppendCertsFromPEM(serverCert)

	config := &tls.Config{
		RootCAs:            certPool, //notre liste de certificat de confiance
		InsecureSkipVerify: true,     // TODO: générer un certificat avec les bonnes IP/SAN
	}

	return tls.Dial("tcp", addr, config) //comme tcp mais avec ajout config certificat
}

type Nackstruct struct {
	PrevNackID   string // id to send to the prevnode
	PrevNodeAddr string
}
type Node struct {
	ID            string
	Port          int
	PrivateKey    *rsa.PrivateKey
	PublicKey     *rsa.PublicKey
	Listener      net.Listener
	ServerAddr    string                // Adresse du serveur d'annuaire (ex: "192.168.1.10:8080")
	NodeIP        string                // IP du nœud vue par le serveur
	PendingACKs   map[string]chan bool  // msgID  canal de notification
	PendingRelays map[string]Nackstruct // recievedNackID  Nackstruct
	Mu            sync.Mutex            // protège PendingACKs
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
	conn, err := DialDirectoryServer(n.ServerAddr)
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

	// NACK:msgid
	if strings.HasPrefix(line, "NACK:") {
		msgId := line[len("NACK:"):]
		fmt.Printf("[%s] NACK received for %s\n", n.ID, msgId)

		n.Mu.Lock()
		//if its the sender
		if ch, ok := n.PendingACKs[msgId]; ok {
			ch <- false
			delete(n.PendingACKs, msgId)
			n.Mu.Unlock()
			return
		}
		Nack, exists := n.PendingRelays[msgId]
		delete(n.PendingRelays, msgId)
		n.Mu.Unlock()
		if exists {
			fmt.Printf("[%s] Propagating NACK for %s to %s\n", n.ID, msgId, Nack.PrevNodeAddr)
			n.SendTo(Nack.PrevNodeAddr, fmt.Sprintf("NACK:%s", Nack.PrevNackID))
		}
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

	// onion layer
	layer, err := StringToOnionLayer(string(decrypted))
	if err != nil {
		fmt.Println(err)
		return
	}
	switch layer.Type {
	case "RELAY":
		fmt.Printf("[%s] Received relay layer - Next from layer: '%s'\n", n.ID, layer.Next)
		fmt.Printf("[%s] Full layer content: Type=%s, MsgID=%s, Next=%s, From=%s \n",
			n.ID, layer.Type, layer.MsgID, layer.Next, layer.From)
		//node relay
		fmt.Printf("[%s] Relai vers %s\n", n.ID, layer.Next)

		// Check if the address includes a port
		if !strings.Contains(layer.Next, ":") {
			fmt.Printf("[%s] Erreur: Adresse sans port: %s\n", n.ID, layer.Next)
			return
		}
		parts := strings.Split(layer.MsgID, ":")
		tosend := parts[0]
		toreceive := parts[1]

		n.Mu.Lock()
		n.PendingRelays[toreceive] = Nackstruct{PrevNackID: tosend, PrevNodeAddr: layer.From}
		n.Mu.Unlock()

		err = n.SendTo(layer.Next, layer.Data)
		if err != nil {
			fmt.Printf("[%s] Erreur relai: %s\n", n.ID, err)
			n.SendTo(layer.From, fmt.Sprintf("NACK:%s", tosend))
		}

	case "FINAL":
		//node final the destination
		fmt.Printf("[%s] Message recu (MsgID : %s): \"%s\"\n", n.ID, layer.MsgID, layer.Message)
		if layer.Next != "" && layer.Data != "" {
			fmt.Printf("[%s] Envoi ACK pour %s via %s\n", n.ID, layer.MsgID, layer.Next)
			err = n.SendTo(layer.Next, layer.Data)
			if err != nil {
				fmt.Printf("[%s] Erreur envoi ACK: %s\n", n.ID, err)
				n.SendTo(layer.From, fmt.Sprintf("NACK:%s", layer.MsgID))
			}
		}
	case "ACK":
		// node sender
		n.Mu.Lock()
		if ch, ok := n.PendingACKs[layer.MsgID]; ok {
			ch <- true
			delete(n.PendingACKs, layer.MsgID)
		}
		n.Mu.Unlock()
	default:
		fmt.Printf("[%s] Type inconnue: %s\n", n.ID, layer.Type)

	}
}

func (n *Node) SendTo(targetAddr string, message string) error {

	conn, err := net.Dial("tcp", targetAddr)
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
	conn, err := DialDirectoryServer(n.ServerAddr)
	if err == nil {
		msg := fmt.Sprintf("QUIT:%s\n", n.ID)
		conn.Write([]byte(msg))
		conn.Close()
	}

	n.Listener.Close()
	fmt.Printf("[%s] Node stopped\n", n.ID)

}

func (n *Node) JoinServerList(addrlist string) error {
	conn, err := DialDirectoryServer(addrlist)
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
		ackParts := strings.SplitN(response, ":", 3)
		if len(ackParts) >= 3 {
			n.NodeIP = ackParts[2]
		}
		fmt.Printf("[%s] Registered to directory server (IP: %s)\n", n.ID, n.NodeIP)
		return nil
	}

	return fmt.Errorf("registration failed: %s", response)
}
