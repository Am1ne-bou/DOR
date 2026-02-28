package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"project/node_server/data"
	"project/node_server/model"

	"github.com/google/uuid"
)

//var mu sync.Mutex
//var nodes = make(map[net.Conn]model.Node)
//var nbrNodes int = 0

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listen:", err)
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
		fmt.Println("Error connecting:", err)
		return
	}

	defer data.Close() // Ensure the database is closed on exit and DELETED to change after

	err = data.InitTable() // Initialize the nodes table if not exists
	if err != nil {
		fmt.Println("Error initializing table:", err)
		return
	}

	err = data.ClearTable() // Clear existing nodes on startup
	if err != nil {
		fmt.Println("Error clearing table:", err)
		return
	}

	fmt.Println("Directory Server on port 8080")
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

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

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
		// Format: INIT:id:port:key
		if len(parts) < 4 {
			_, err := conn.Write([]byte("ERROR:Invalid format\n"))
			if err != nil {
				return
			}
			return
		}

		name := parts[1]
		port, _ := strconv.Atoi(parts[2])
		key := parts[3]

		info := model.NodeInfo{
			Uuid:      uuid.New().String(),
			Name:      name,
			Ip:        conn.RemoteAddr().String(),
			Port:      port,
			PublicKey: key,
		}

		// Ajout dans BDD
		err := data.AddNode(&info)
		if err != nil {
			fmt.Println("Error adding node:", err)
			return
		}

		fmt.Printf("[+] Node %s registered (Port: %d)\n", name, port)
		_, err = conn.Write([]byte("INIT_ACK:" + name + "\n"))
		if err != nil {
			return
		}

	case "GET_LIST":
		_, err := conn.Write([]byte(getNodesList()))
		if err != nil {
			return
		}

	case "GET_KEY":
		// Format GET_KEY:port
		if len(parts) < 2 {
			_, err := conn.Write([]byte("ERROR:Invalid format\n"))
			if err != nil {
				return
			}
			return
		}

		port, err := strconv.Atoi(parts[1])
		if err != nil {
			_, err := conn.Write([]byte("ERROR:Invalid port\n"))
			if err != nil {
				return
			}
			return
		}

		nodes, _ := data.GetNodesList()

		for _, node := range nodes {
			if node.Port == port {
				_, err := conn.Write([]byte("KEY:" + node.PublicKey + "\n"))
				if err != nil {
					return
				}
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
			return
		}

		fmt.Printf("[-] Node %s unregistered\n", id)

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
	nodes, err := data.GetNodesList()
	if err != nil || len(nodes) == 0 {
		return "LIST:empty\n"
	}

	var result strings.Builder
	result.WriteString("LIST:")

	for i, info := range nodes {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(fmt.Sprintf("%s|%d|%s", info.Name, info.Port, info.PublicKey))
	}
	result.WriteString("\n")

	return result.String()
}

func showNodes() {
	nodes, err := data.GetNodesList()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("\n=== Noeuds connectés ===")
	if len(nodes) == 0 {
		fmt.Println("  (aucun)")
	} else {
		for _, info := range nodes {
			fmt.Printf("  . %s - Port: %d, Key: %s\n", info.Name, info.Port, info.PublicKey)
		}
	}
	fmt.Printf("Total: %d\n\n", len(nodes))
}

func TestPing() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		nodes, err := data.GetNodesList()
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		for _, node := range nodes {
			addr := fmt.Sprintf("localhost:%d", node.Port)
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				err := data.RemoveNode(node.Name)
				if err != nil {
					fmt.Println("Error removing node:", err)
					return
				}
				fmt.Printf("Node %s removed\n", node.Name)
			} else {
				err := conn.Close()
				if err != nil {
					fmt.Println("Error closing connection:", err)
					return
				}
			}
		}
	}
}
