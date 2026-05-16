package model

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64" //Ce package va servir a stoker les clés (pour faire la diff entre \n et un octet qui prendrais la valeur associé à \n, idem pour ":")
	"fmt"
	"io"
	"log/slog"
	mrand "math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//ATTENTION LA LIGNE EN DESSOUS N'EST PAS UN COMMENTAIRE
//go:embed cert.pem
//ATTENTION ne pas enlever les // ou supprimer la ligne au dessus !
//(c'est pour la compil pour intégrer le fichier),

var serverCert []byte

// TODO: replace static RSA session key with ephemeral ECDH (X25519) for forward secrecy
func DialDirectoryServer(addr string) (*tls.Conn, error) {
	certPool := x509.NewCertPool() //liste de certificats (vide pr l'instant)
	certPool.AppendCertsFromPEM(serverCert)

	config := &tls.Config{
		RootCAs:            certPool, //notre liste de certificat de confiance
		InsecureSkipVerify: true, // TODO: replace with proper cert validation once nodes have real SAN certs
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
	KeyMu         sync.RWMutex // protège PrivateKey et PublicKey
	Listener      net.Listener
	ServerAddr    string                // Adresse du serveur d'annuaire (ex: "192.168.1.10:8080")
	NodeIP        string                // IP du nœud vue par le serveur
	// TODO: PendingACKs should have TTL-based eviction, map grows unbounded under packet loss
	PendingACKs   map[string]chan bool  // msgID  canal de notification
	PendingRelays map[string]Nackstruct // recievedNackID  Nackstruct
	Mu            sync.Mutex            // protège PendingACKs
	FragBuf       map[string]map[int]string // fragID -> {chunkIndex -> chunk}
	FragTotal     map[string]int            // fragID -> expected total chunk count
	FragMu        sync.Mutex                // protège FragBuf et FragTotal
	SeenMsgs      map[string]time.Time      // msgID -> first seen; evicted after 30s
	SeenMu        sync.Mutex               // protects SeenMsgs
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

// seenMsg returns true if msgID was already processed (loop or replay), marks it if not
func (n *Node) seenMsg(msgID string) bool {
	n.SeenMu.Lock()
	defer n.SeenMu.Unlock()
	if n.SeenMsgs == nil {
		n.SeenMsgs = make(map[string]time.Time)
	}
	if _, seen := n.SeenMsgs[msgID]; seen {
		return true
	}
	n.SeenMsgs[msgID] = time.Now()
	return false
}

func (n *Node) StartNode() {
	slog.Info("node started", "id", n.ID, "port", n.Port)
	// evict seen MsgIDs older than 30s to prevent unbounded growth
	go func() {
		for {
			time.Sleep(30 * time.Second)
			n.SeenMu.Lock()
			for id, t := range n.SeenMsgs {
				if time.Since(t) > 30*time.Second {
					delete(n.SeenMsgs, id)
				}
			}
			n.SeenMu.Unlock()
		}
	}()
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

	if _, err := conn.Write([]byte("GET_LIST\n")); err != nil {
		return "", err
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

////

// TODO: add timestamp+nonce to each onion layer and reject replays within a TTL window
func (n *Node) handlerroutine(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			slog.Debug("handlerroutine read error", "id", n.ID, "err", err)
		}
		return
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if line == "GET_PUBKEY" {
		n.KeyMu.RLock()
		pubBytes, _ := x509.MarshalPKIXPublicKey(n.PublicKey)
		n.KeyMu.RUnlock()
		pubBase64 := base64.StdEncoding.EncodeToString(pubBytes)
		if _, err := conn.Write([]byte(pubBase64 + "\n")); err != nil {
			slog.Debug("GET_PUBKEY write", "id", n.ID, "err", err) // requester disconnected, skip
		}
		return
	}

	// NACK:msgid
	if strings.HasPrefix(line, "NACK:") {
		msgId := line[len("NACK:"):]
		slog.Info("NACK received", "id", n.ID, "msgID", msgId)

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
			slog.Info("propagating NACK", "id", n.ID, "msgID", msgId, "to", Nack.PrevNodeAddr)
			for _, fromAddr := range ParseAddresses(Nack.PrevNodeAddr) {
				if n.SendTo(fromAddr, fmt.Sprintf("NACK:%s", Nack.PrevNackID)) == nil {
					break
				}
			}
		}
		return
	}
	// try decrypter le message
	n.KeyMu.RLock()
	decrypted, err := BroadcastDecrypt(line, n.PrivateKey)
	n.KeyMu.RUnlock()
	if err != nil {
		slog.Error("decryption failed", "id", n.ID, "err", err)
		return
	}

	// onion layer
	layer, err := StringToOnionLayer(string(decrypted))
	if err != nil {
		slog.Error("parse onion layer", "id", n.ID, "err", err)
		return
	}
	// drop loops and replays -- NACK/ACK excluded (they must propagate freely)
	if layer.Type == "RELAY" || layer.Type == "FINAL" {
		if n.seenMsg(layer.MsgID) {
			slog.Warn("duplicate MsgID dropped", "id", n.ID, "msgID", layer.MsgID, "type", layer.Type)
			return
		}
	}
	switch layer.Type {
	case "RELAY":
		slog.Debug("relay layer", "id", n.ID, "msgID", layer.MsgID, "next", layer.Next, "from", layer.From)

		// Check if the address includes a port
		if !strings.Contains(layer.Next, ":") {
			slog.Error("address missing port", "id", n.ID, "next", layer.Next)
			return
		}
		parts := strings.SplitN(layer.MsgID, ":", 2)
		if len(parts) != 2 {
			fmt.Printf("[%s] Erreur: MsgID format invalide: %s\n", n.ID, layer.MsgID)
			return
		}
		tosend := parts[0]
		toreceive := parts[1]

		n.Mu.Lock()
		n.PendingRelays[toreceive] = Nackstruct{PrevNackID: tosend, PrevNodeAddr: layer.From}
		n.Mu.Unlock()

		// try each candidate in Next group
		nextAddrs := ParseAddresses(layer.Next)
		mrand.Shuffle(len(nextAddrs), func(i, j int) {
			nextAddrs[i], nextAddrs[j] = nextAddrs[j], nextAddrs[i]
		})
		sent := false
		for _, addr := range nextAddrs {
			if n.SendTo(addr, layer.Data) == nil {
				sent = true
				break
			}
			slog.Warn("candidate unreachable", "id", n.ID, "addr", addr)
		}
		if !sent {
			slog.Warn("all next hops offline, sending NACK", "id", n.ID, "msgID", layer.MsgID)
			// nobody will send toreceive back to us; clean up to avoid leak
			n.Mu.Lock()
			delete(n.PendingRelays, toreceive)
			n.Mu.Unlock()
			for _, fromAddr := range ParseAddresses(layer.From) {
				if n.SendTo(fromAddr, fmt.Sprintf("NACK:%s", tosend)) == nil {
					break
				}
			}
		}

	case "FINAL":
		//node final the destination
		if layer.Frag != "" {
			// buffer this chunk; when all N arrive, assemble in order and deliver
			fp := strings.SplitN(layer.Frag, ":", 2)
			fragID := fp[0]
			in := strings.Split(fp[1], "/")
			idx, _ := strconv.Atoi(in[0])
			total, _ := strconv.Atoi(in[1])
			n.FragMu.Lock()
			if n.FragBuf == nil {
				n.FragBuf = make(map[string]map[int]string)
				n.FragTotal = make(map[string]int)
			}
			if n.FragBuf[fragID] == nil {
				n.FragBuf[fragID] = make(map[int]string)
				n.FragTotal[fragID] = total
			}
			n.FragBuf[fragID][idx] = layer.Message
			if len(n.FragBuf[fragID]) == n.FragTotal[fragID] {
				// all chunks arrived: assemble in order and deliver
				var b strings.Builder
				for i := 1; i <= total; i++ { // chunks indexed from 1
					b.WriteString(n.FragBuf[fragID][i])
				}
				assembled := b.String()
				delete(n.FragBuf, fragID)
				delete(n.FragTotal, fragID)
				n.FragMu.Unlock()
				slog.Info("message reassembled", "id", n.ID, "fragID", fragID, "chunks", total, "msg", assembled)
			} else {
				n.FragMu.Unlock()
			}
		} else {
			slog.Info("message received", "id", n.ID, "msgID", layer.MsgID, "msg", layer.Message)
		}
		if layer.Next != "" && layer.Data != "" {
			slog.Info("sending ACK", "id", n.ID, "msgID", layer.MsgID, "via", layer.Next)
			nextAddrs := ParseAddresses(layer.Next)
			mrand.Shuffle(len(nextAddrs), func(i, j int) {
				nextAddrs[i], nextAddrs[j] = nextAddrs[j], nextAddrs[i]
			})
			sent := false
			for _, addr := range nextAddrs {
				if n.SendTo(addr, layer.Data) == nil {
					sent = true
					break
				}
			}
			if !sent {
				for _, fromAddr := range ParseAddresses(layer.From) {
					if n.SendTo(fromAddr, fmt.Sprintf("NACK:%s", layer.MsgID)) == nil {
						break
					}
				}
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
		slog.Warn("unknown layer type", "id", n.ID, "type", layer.Type)

	}
}

func (n *Node) SendTo(targetAddr string, message string) error {
	// Simuler la latence si configuré
	if simLatency := os.Getenv("SIM_LATENCY"); simLatency != "" {
		maxMs, _ := strconv.Atoi(simLatency)
		if maxMs > 0 {
			delay := time.Duration(10+mrand.Intn(maxMs)) * time.Millisecond
			time.Sleep(delay)
		}
	}

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
	// Send QUIT to server to leave the list
	conn, err := DialDirectoryServer(n.ServerAddr)
	if err == nil {
		msg := fmt.Sprintf("QUIT:%s\n", n.ID)
		conn.Write([]byte(msg))
		conn.Close()
	}

	n.Listener.Close()
	slog.Info("node stopped", "id", n.ID)

}

func (n *Node) JoinServerList(addrlist string, sa int, sn int) error {
	conn, err := DialDirectoryServer(addrlist)
	if err != nil {
		return err
	}
	defer conn.Close()

	//Pr envoyer la clé publique sur le réseau (format "reconnu" partout)
	// on utilise le format PKIX (encodage en ASN.1 DER).
	//on appelle cette etape la serialisation
	n.KeyMu.RLock()
	pubBytes, err := x509.MarshalPKIXPublicKey(n.PublicKey)
	n.KeyMu.RUnlock()
	if err != nil {
		return fmt.Errorf("erreur sérialisation clé: %v", err)
	}

	//ensuite on utilise la base 64 et pas le binaire pour le pb des \n
	pubBase64 := base64.StdEncoding.EncodeToString(pubBytes)

	// Send (v2): INIT:id:port:key:sa:sn
	msg := fmt.Sprintf("INIT:%s:%d:%s:%d:%d\n", n.ID, n.Port, pubBase64, sa, sn)
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
		slog.Info("registered", "id", n.ID, "ip", n.NodeIP)
		return nil
	}

	return fmt.Errorf("registration failed: %s", response)
}

func (n *Node) RegenerateKeys() error {
	newPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	newPub := newPriv.PublicKey

	pubBytes, err := x509.MarshalPKIXPublicKey(&newPub)
	if err != nil {
		return err
	}
	pubBase64 := base64.StdEncoding.EncodeToString(pubBytes)

	n.KeyMu.Lock()
	n.PrivateKey = newPriv
	n.PublicKey = &newPub
	n.KeyMu.Unlock()

	conn, err := DialDirectoryServer(n.ServerAddr)
	if err != nil {
		return fmt.Errorf("erreur de connexion à l'annuaire: %v", err)
	}
	defer conn.Close()

	msg := fmt.Sprintf("UPDATE_KEY:%s:%s\n", n.ID, pubBase64)
	_, err = conn.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("erreur d'envoi à l'annuaire: %v", err)
	}

	slog.Info("RSA key rotated", "id", n.ID)
	return nil
}
