package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"project/node_server/data"
	"project/node_server/model"

	"github.com/google/uuid"
)

const MaxSampleNodes = 500

// TODO: add heartbeat loop -- periodically evict nodes that haven't pinged in > N seconds
func main() {

	//chargement du certificat
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		slog.Error("load certificate", "err", err)
		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	//écoute en tls
	listener, err := tls.Listen("tcp4", "0.0.0.0:8080", config)
	if err != nil {
		slog.Error("listen failed", "err", err)
		return
	}

	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("Error closing listener:", err)
		}
	}(listener)

	// Initialize the database
	err = data.Connect("dor_nodes.db") // Open the database
	if err != nil {
		slog.Error("db connect", "err", err)
		return
	}

	defer data.Close() // Ensure the database is closed on exit and DELETED to change after

	err = data.InitTable() // Initialize the nodes table if not exists
	if err != nil {
		slog.Error("init table", "err", err)
		return
	}

	err = data.ClearTable() // Clear existing nodes on startup
	if err != nil {
		slog.Error("clear table", "err", err)
		return
	}

	slog.Info("directory server started", "port", 8080)
	fmt.Println("\nCommandes disponibles:")
	fmt.Println("  LIST - Afficher les noeuds connectés")
	fmt.Println("  QUIT - Arrêter le serveur")
	fmt.Println()

	// Listen for incoming connections
	go acceptConnections(listener)
	go TestPing()
	// Command from stdin
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		switch strings.ToUpper(input) {
		case "LIST":
			showNodes()
		case "QUIT":
			fmt.Println("Shutting down server...")
			return
		default:
			fmt.Println("Commande inconnue. Utilise LIST ou QUIT")
		}
	}
}

func acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		// Handle each connection in a new goroutine
		go handleConnection(conn)
	}
}

// TODO: rate-limit incoming connections per IP to prevent directory server flooding
func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	line = strings.TrimSpace(line)
	parts := strings.Split(line, ":")

	if len(parts) < 1 {
		return
	}

	cmd := parts[0]

	switch cmd {
	case "INIT":
		// Format v2: INIT:id:port:key:Sa:Sn
		if len(parts) < 6 {
			_, err = conn.Write([]byte("ERROR:Invalid format\n"))
			return
		}

		name := parts[1]
		port, _ := strconv.Atoi(parts[2])
		key := parts[3]
		sa, _ := strconv.Atoi(parts[4]) // Score de disponibilité
    	sn, _ := strconv.Atoi(parts[5]) // Score réseau

		ip_string := conn.RemoteAddr().String()
		host := os.Getenv("NODE_ADDR")
		if host == "" {
			host, _, _ = net.SplitHostPort(ip_string)
		}

		info := model.NodeInfo{
			Uuid:      uuid.New().String(),
			Name:      name,
			Ip:        host,
			Port:      port,
			PublicKey: key,
			AvailabilityScore: sa,
        	NetworkScore: sn,
		}

		// Ajout dans BDD
		err = data.AddNode(&info)
		if err != nil {
			slog.Error("add node", "name", name, "err", err)
			return
		}

		slog.Info("node registered", "name", name, "port", port, "sa", sa, "sn", sn)
		_, err = conn.Write([]byte("INIT_ACK:" + name + ":" + host + "\n"))
		if err != nil {
			return
		}

	case "UPDATE_KEY":
		// Format: UPDATE_KEY:id:key
		if len(parts) < 3 {
			_, err := conn.Write([]byte("ERROR:Invalid format\n"))
			if err != nil {
				return
			}
			return
		}
		id := parts[1]
		key := parts[2]

		err := data.UpdateNodeKey(id, key)
		if err != nil {
			slog.Error("update key", "id", id, "err", err)
		} else {
			slog.Info("key updated", "id", id)
		}

	case "GET_LIST":
		_, err := conn.Write([]byte(getNodesList()))
		if err != nil {
			return
		}

	case "GET_KEY":
		// Format GET_KEY:<ip>:<port>
		if len(parts) < 3 {
			_, err := conn.Write([]byte("ERROR:Invalid format\n"))
			if err != nil {
				return
			}
			return
		}

		ip := parts[1]
		port, err := strconv.Atoi(parts[2])
		if err != nil {
			_, err := conn.Write([]byte("ERROR:Invalid port\n"))
			if err != nil {
				return
			}
			return
		}

		nodes, _ := data.GetNodesList(MaxSampleNodes)

		for _, node := range nodes {
			if node.Ip == ip && node.Port == port {
				_, err = conn.Write([]byte("KEY:" + node.PublicKey + "\n"))
				if err != nil {
					return
				}
				return
			}
		}

		_, err = conn.Write([]byte("ERROR:Node not found\n"))
		if err != nil {
			return
		}

	case "QUIT":
		// Format: QUIT:id
		if len(parts) < 2 {
			return
		}
		id := parts[1]

		err := data.RemoveNode(id)
		if err != nil {
			slog.Error("remove node", "id", id, "err", err)
			return
		}

		slog.Info("node unregistered", "id", id)

	default:
		_, err := conn.Write([]byte("ERROR:Unknown command\n"))
		if err != nil {
			return
		}
		return
	}
}

func getNodesList() string {
	// Utiliser data.GetNodesList() à la place
	nodes, err := data.GetNodesList(MaxSampleNodes)
	if err != nil || len(nodes) == 0 {
		return "LIST:empty\n"
	}

	var result strings.Builder
	result.WriteString("LIST:")

	for i, info := range nodes {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(fmt.Sprintf("%s|%s|%d|%s|%d|%d", info.Name, info.Ip, info.Port, info.PublicKey, info.AvailabilityScore, info.NetworkScore))
	}
	result.WriteString("\n")

	return result.String()
}

func showNodes() {
	nodes, err := data.GetNodesList(MaxSampleNodes)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("\n=== Noeuds connectés ===")
	if len(nodes) == 0 {
		fmt.Println("  (aucun)")
	} else {
		for _, info := range nodes {
			fmt.Printf("  . %s - Addr: %s:%d, Key: %s, [Sa:%d Sn:%d]\n", info.Name, info.Ip, info.Port, info.PublicKey, info.AvailabilityScore, info.NetworkScore)
		}
	}
	fmt.Printf("Total: %d\n\n", len(nodes))
}

func TestPing() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		nodes, err := data.GetNodesList(MaxSampleNodes)
		if err != nil {
			slog.Error("TestPing get nodes", "err", err)
			continue
		}

		for _, node := range nodes {
			addr := net.JoinHostPort(node.Ip, strconv.Itoa(node.Port))
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				newScore := node.AvailabilityScore / 2
				if newScore < 5 {
					if err := data.RemoveNode(node.Name); err != nil {
						slog.Error("evict node", "name", node.Name, "err", err)
						continue
					}
					slog.Info("node evicted", "name", node.Name)
				} else {
					if err := data.UpdateAvailabilityScore(node.Name, newScore); err != nil {
						slog.Error("update score", "name", node.Name, "err", err)
					}
					slog.Info("node degraded", "name", node.Name, "score", node.AvailabilityScore, "new", newScore)
				}
			} else {
				conn.Close()
				if node.AvailabilityScore < 100 {
					newScore := min(node.AvailabilityScore+5, 100)
					if err := data.UpdateAvailabilityScore(node.Name, newScore); err != nil {
						slog.Error("update score", "name", node.Name, "err", err)
					}
				}
			}
		}
	}
}
