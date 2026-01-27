package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Node struct {
	ID       string
	Port     int
	Listener net.Listener
}

func NewNode(id string, port int) (*Node, error) {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Node{
		ID:       id,
		Port:     port,
		Listener: listener,
	}, nil

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

func (n *Node) handlerroutine(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	line = strings.TrimSpace(line)
	parts := strings.SplitN(line, ":", 2)

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

func (n *Node) JoinServerList(addrlist string, key int) error {
	conn, err := net.Dial("tcp", addrlist)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send: INIT:id:port:key
	msg := fmt.Sprintf("INIT:%s:%d:%d\n", n.ID, n.Port, key)
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

func main() {
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

	//===========
	key := 12345 // TODO: générer une vraie clé
	err = node.JoinServerList("localhost:8080", key)
	if err != nil {
		fmt.Println("Warning: Could not register to directory:", err)
		fmt.Println("(Continuing without registration)")
	}
	//===========

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nCommandes disponibles:")
	fmt.Println("  MSG:<port>:<message>              - Message direct")
	fmt.Println("  RELAY:<port>:<port>:...:<message> - Relai multi-hop")
	fmt.Println("  QUIT                              - Quitter")
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
			err = node.SendTo(dst, "MSG:"+data)
			if err != nil {
				fmt.Println("Error sending message:", err)
			}
		case "RELAY":
			// Format: RELAY:<port1>:<port2>:...:<message>
			// Build the relay chain
			subParts := strings.SplitN(data, ":", 2)
			if len(subParts) < 2 {
				fmt.Println("Format: RELAY:<port>:<port>:...:<message>")
				continue
			}
			firstPort, err := strconv.Atoi(subParts[0])
			if err != nil {
				fmt.Println("Port invalide")
				continue
			}
			rest := subParts[1]

			// Build the relay chain message
			msgToSend := buildRelayChain(rest)
			err = node.SendTo(firstPort, msgToSend)
			if err != nil {
				fmt.Println("Erreur:", err)
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

func buildRelayChain(data string) string {
	parts := strings.SplitN(data, ":", 2)

	if len(parts) < 2 {
		// No more ports, just a message
		return "MSG:" + data
	}

	//Test if parts[0] is a port number
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
