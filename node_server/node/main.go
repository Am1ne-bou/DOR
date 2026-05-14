package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"strings"

	"project/node_server/model"
)

func NewNode(id string, serverAddr string) (*model.Node, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	publicKey := privateKey.PublicKey

	addr := fmt.Sprintf("0.0.0.0:%d", 0)
	os_port := os.Getenv("PORT")
	if os_port != "" {
		addr = fmt.Sprintf("0.0.0.0:%s", os_port)
	}
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}

	return &model.Node{
		ID:            id,
		Port:          listener.Addr().(*net.TCPAddr).Port,
		PrivateKey:    privateKey,
		PublicKey:     &publicKey,
		Listener:      listener,
		ServerAddr:    serverAddr,
		PendingACKs:   make(map[string]chan bool),
		PendingRelays: make(map[string]model.Nackstruct),
	}, nil
}

// TODO: replace fmt.Println with slog (stdlib since Go 1.21) -- add log levels and JSON output mode
func main() {
	publicKeys := newKeyCache()

	var id string
	if len(os.Args) >= 2 && !strings.HasPrefix(os.Args[1], "--") {
		id = os.Args[1]
	} else {
		id = os.Getenv("NODE_ID")
	}
	if id == "" {
		fmt.Println("Usage: go run main.go <id>  OR set NODE_ID env variable")
		return
	}
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:8080"
	}

	node, err := NewNode(id, serverAddr)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	go node.StartNode()

	profile := os.Getenv("NETWORK_PROFILE")
	sa, sn := GetScoresFromProfile(profile)

	err = node.JoinServerList(serverAddr, sa, sn)
	if err != nil {
		fmt.Println("Error joining server:", err)
	}

	if os.Getenv("ENABLE_WEB") == "1" {
		startStdoutBridge()
		go startWebUI(node, serverAddr, publicKeys)
	}

	runCLI(node, serverAddr, publicKeys)
}
