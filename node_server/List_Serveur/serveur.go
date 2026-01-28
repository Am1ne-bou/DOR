package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"project/node_server/model"
)

var mu sync.Mutex
var nodes = make(map[net.Conn]model.Node)
var nbrNodes int = 0

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listen:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Directory Server on port 8080")
	fmt.Println("\nCommandes disponibles:")
	fmt.Println("  LIST - Afficher les noeuds connectés")
	fmt.Println("  QUIT - Arrêter le serveur")
	fmt.Println()

	// Listen for incoming connections
	go acceptConnections(listener)

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

		mu.Lock()
		nodes[conn] = info
		nbrNodes++
		count := nbrNodes
		mu.Unlock()

		fmt.Printf("[+] Node %s registered (Port: %d, Total: %d)\n", id, port, count)
		conn.Write([]byte("INIT_ACK:" + id + "\n"))

	case "GET_LIST":
		conn.Write([]byte(getNodesList()))

	case "QUIT":
		// Format: QUIT:id
		if len(parts) < 2 {
			return
		}
		id := parts[1]

		mu.Lock()
		for conn, info := range nodes {
			if info.ID == id {
				delete(nodes, conn)
				nbrNodes--
				break
			}
		}
		mu.Unlock()

		fmt.Printf("[-] Node %s unregistered\n", id)

	default:
		conn.Write([]byte("ERROR:Unknown command\n"))
		return
	}

}

func getNodesList() string {
	mu.Lock()
	defer mu.Unlock()

	if nbrNodes == 0 {
		return "LIST:empty\n"
	}

	var result strings.Builder
	result.WriteString("LIST:")

	first := true
	for _, info := range nodes {
		if !first {
			result.WriteString(",")
		}
		// Format: id|port|key
		result.WriteString(fmt.Sprintf("%s|%d|%d", info.ID, info.Port, info.Key))
		first = false
	}
	result.WriteString("\n")

	return result.String()
}

func showNodes() {
	mu.Lock()
	defer mu.Unlock()

	fmt.Println("\n=== Noeuds connectés ===")
	if nbrNodes == 0 {
		fmt.Println("  (aucun)")
	} else {
		for _, info := range nodes {
			fmt.Printf("  . %s - Port: %d, Key: %d\n", info.ID, info.Port, info.Key)
		}
	}
	fmt.Printf("Total: %d\n\n", nbrNodes)
}
