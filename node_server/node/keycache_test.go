package main

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

func TestKeyCacheGetSet(t *testing.T) {
	c := newKeyCache()

	_, ok := c.get("missing")
	if ok {
		t.Fatal("expected miss on empty cache")
	}

	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	v := CachedKey{Key: &priv.PublicKey, ExpiresAt: time.Now().Add(time.Minute)}
	c.set("addr1", v)

	got, ok := c.get("addr1")
	if !ok {
		t.Fatal("expected hit after set")
	}
	if got.Key != v.Key {
		t.Error("returned wrong key")
	}
}

func TestKeyCacheConcurrent(t *testing.T) {
	c := newKeyCache()
	priv, _ := rsa.GenerateKey(rand.Reader, 1024)
	v := CachedKey{Key: &priv.PublicKey, ExpiresAt: time.Now().Add(time.Minute)}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			c.set("k", v)
			c.get("k")
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
