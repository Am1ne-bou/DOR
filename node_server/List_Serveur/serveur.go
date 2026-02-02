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
	defer listener.Close()

	// Initialize the database
	data.Connect("test.db") // Open the database
	defer data.Close()      // Ensure the database is closed on exit and DELETED to change after
	data.InitTable()        // Initialize the nodes table if not exists
	data.ClearTable()       // Clear existing nodes on startup

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
		// Format: INIT:id:port:key
		if len(parts) < 4 {
			conn.Write([]byte("ERROR:Invalid format\n"))
			return
		}

		id := parts[1]
		port, _ := strconv.Atoi(parts[2])
		key, _ := strconv.Atoi(parts[3])

		info := model.Node{
			ID:       id,
			Port:     port,
			Key:      key,
			Listener: nil,
		}

		// Ajout dans BDD
		data.AddNode(&info)

		// Lecture des noeuds dans bdd
		nodesSQL, _ := data.GetNodesList()
		for _, node := range nodesSQL {
			fmt.Printf("%s : %d : %d\n", node.ID, node.Port, node.Key)
		}

		fmt.Printf("[+] Node %s registered (Port: %d, Total: %d)\n", id, port, len(nodesSQL))
		conn.Write([]byte("INIT_ACK:" + id + "\n"))

	case "GET_LIST":
		conn.Write([]byte(getNodesList()))

	case "QUIT":
		// Format: QUIT:id
		if len(parts) < 2 {
			return
		}
		id := parts[1]

		data.RemoveNode(id)

		// Lecture des noeuds dans bdd
		//nodesSQL, _ := data.GetNodesList()
		//for _, node := range nodesSQL {
		//	fmt.Printf("%s : %d : %d\n", node.ID, node.Port, node.Key)
		//}

		fmt.Printf("[-] Node %s unregistered\n", id)

	default:
		conn.Write([]byte("ERROR:Unknown command\n"))
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
		result.WriteString(fmt.Sprintf("%s|%d|%d", info.ID, info.Port, info.Key))
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
			fmt.Printf("  . %s - Port: %d, Key: %d\n", info.ID, info.Port, info.Key)
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
            continue
        }

        for _, node := range nodes {
            addr := fmt.Sprintf("localhost:%d", node.Port)
            conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
            if err != nil {
                data.RemoveNode(node.ID)
                fmt.Printf("Node %s removed\n", node.ID)
            } else {
                conn.Close()
            }
        }
    }
}
