package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"project/node_server/model"
)

func TestGetScoresFromProfile(t *testing.T) {
	cases := []struct {
		profile string
		wantSa  int
		wantSn  int
	}{
		{"server", 100, 100},
		{"laptop_WIFI7", 60, 85},
		{"smartphone_4G", 30, 75},
		{"smartphone_2G", 20, 10},
		{"unknown", 15, 15},
		{"", 15, 15},
	}
	for _, c := range cases {
		sa, sn := GetScoresFromProfile(c.profile)
		if sa != c.wantSa || sn != c.wantSn {
			t.Errorf("profile=%q: got (%d,%d) want (%d,%d)", c.profile, sa, sn, c.wantSa, c.wantSn)
		}
	}
}

func TestEncryptForNode_FormatAndRoundtrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("hello onion")

	result, err := encryptForNode(plaintext, &priv.PublicKey)
	if err != nil {
		t.Fatalf("encryptForNode: %v", err)
	}

	// format: base64(encKey):base64(encData)
	parts := strings.SplitN(result, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("expected colon separator, got: %q", result)
	}

	encKeyBytes, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("encKey base64: %v", err)
	}
	encDataBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("encData base64: %v", err)
	}

	// decrypt AES key with RSA private key
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKeyBytes, nil)
	if err != nil {
		t.Fatalf("RSA decrypt: %v", err)
	}

	// decrypt payload with AES key
	got, err := model.DecryptAES(aesKey, encDataBytes)
	if err != nil {
		t.Fatalf("AES decrypt: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Errorf("roundtrip: got %q want %q", got, plaintext)
	}
}
