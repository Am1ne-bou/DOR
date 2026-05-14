package main

import (
	"bufio"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"project/node_server/model"
)

type CachedKey struct {
	Key       *rsa.PublicKey
	ExpiresAt time.Time
}

type KeyCache struct {
	mu sync.RWMutex
	m  map[string]CachedKey
}

func newKeyCache() *KeyCache { return &KeyCache{m: make(map[string]CachedKey)} }

func (c *KeyCache) get(k string) (CachedKey, bool) {
	c.mu.RLock()
	v, ok := c.m[k]
	c.mu.RUnlock()
	return v, ok
}

func (c *KeyCache) set(k string, v CachedKey) {
	c.mu.Lock()
	c.m[k] = v
	c.mu.Unlock()
}

func FetchKeyFromServer(addr string, serverAddr string) (*rsa.PublicKey, error) {
	conn, err := model.DialDirectoryServer(serverAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.Write([]byte(fmt.Sprintf("GET_KEY:%s\n", addr)))

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	response = strings.TrimSpace(response)

	if strings.HasPrefix(response, "ERROR:") {
		return nil, fmt.Errorf("%s", response)
	}

	parts := strings.SplitN(response, ":", 2)
	if len(parts) != 2 || parts[0] != "KEY" {
		return nil, fmt.Errorf("invalid response")
	}

	publicBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	pubKey, err := x509.ParsePKIXPublicKey(publicBytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaKey, nil
}
