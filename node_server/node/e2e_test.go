package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"testing"
	"time"

	"project/node_server/model"
)

func makeTestNode(t *testing.T) (*model.Node, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	n := &model.Node{
		ID:            addr,
		Port:          port,
		PrivateKey:    priv,
		PublicKey:     &priv.PublicKey,
		NodeIP:        "127.0.0.1",
		Listener:      ln,
		ServerAddr:    "127.0.0.1:0", // unused, keys pre-cached
		PendingACKs:   make(map[string]chan bool),
		PendingRelays: make(map[string]model.Nackstruct),
	}
	go n.StartNode()
	t.Cleanup(func() { ln.Close() })
	return n, addr
}

func TestSendACKRoundtrip(t *testing.T) {
	sender, senderAddr := makeTestNode(t)
	relay1, relay1Addr := makeTestNode(t)
	relay2, relay2Addr := makeTestNode(t)
	dest, destAddr := makeTestNode(t)

	keys := newKeyCache() // no directory server needed
	for _, n := range []*model.Node{sender, relay1, relay2, dest} {
		a := fmt.Sprintf("127.0.0.1:%d", n.Port)
		keys.set(a, CachedKey{Key: n.PublicKey, ExpiresAt: time.Now().Add(time.Minute)})
	}

	route := []string{relay1Addr, relay2Addr, destAddr}
	returnRoute := []string{relay2Addr, relay1Addr, senderAddr} // reversed for ACK

	onion, msgID, firstNackID, err := Encapsulator_func("hello world", route, returnRoute, keys, "", senderAddr)
	if err != nil {
		t.Fatalf("Encapsulator_func: %v", err)
	}

	ackChan := make(chan bool, 1)
	sender.Mu.Lock()
	sender.PendingACKs[msgID] = ackChan
	sender.PendingACKs[firstNackID] = ackChan // nack hits same chan
	sender.Mu.Unlock()

	if err := sender.SendTo(relay1Addr, onion); err != nil {
		t.Fatalf("SendTo: %v", err)
	}

	select {
	case ok := <-ackChan:
		if !ok {
			t.Fatal("got NACK, want ACK")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for ACK")
	}
}

func TestSSendACKRoundtrip(t *testing.T) {
	sender, senderAddr := makeTestNode(t)
	relay1, relay1Addr := makeTestNode(t)
	relay2, relay2Addr := makeTestNode(t)
	dest, destAddr := makeTestNode(t)

	group := func(n *model.Node, addr string) LayerGroup { // single-node cluster
		return LayerGroup{Addrs: []string{addr}, PubKeys: []*rsa.PublicKey{n.PublicKey}}
	}

	route := []LayerGroup{group(relay1, relay1Addr), group(relay2, relay2Addr), group(dest, destAddr)}
	returnRoute := []LayerGroup{group(relay2, relay2Addr), group(relay1, relay1Addr), group(sender, senderAddr)} // reversed for ACK

	onion, msgID, firstNackID, err := Encapsulator_func_super("hello world", route, returnRoute, senderAddr)
	if err != nil {
		t.Fatalf("Encapsulator_func_super: %v", err)
	}

	ackChan := make(chan bool, 1)
	sender.Mu.Lock()
	sender.PendingACKs[msgID] = ackChan
	sender.PendingACKs[firstNackID] = ackChan // nack hits same chan
	sender.Mu.Unlock()

	if err := sender.SendTo(relay1Addr, onion); err != nil {
		t.Fatalf("SendTo: %v", err)
	}

	select {
	case ok := <-ackChan:
		if !ok {
			t.Fatal("got NACK, want ACK")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for ACK")
	}
}
