package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"project/node_server/model"
)

func NewNode(id string, port int) (*model.Node, error) {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &model.Node{
		ID:       id,
		Port:     port,
		Listener: listener,
	}, nil

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
