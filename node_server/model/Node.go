package model

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Node struct {
	ID       string
	Port     int
	Key      int
	Listener net.Listener
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
