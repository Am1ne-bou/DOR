package model

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// ParseAddresses split Next field on commas
func ParseAddresses(next string) []string {
	if next == "" {
		return []string{}
	}
	return strings.Split(next, ",")
}

func JoinAddresses(addrs []string) string {
	return strings.Join(addrs, ",")
}

// BroadcastEncrypt broadcast version of encryptForNode
// same AES key encrypted with each RSA pubkey in the group
// format: "rsaKey1;rsaKey2;rsaKeyN:aesPayload" (all base64)
func BroadcastEncrypt(plaintext []byte, pubKeys []*rsa.PublicKey) (string, error) {
	if len(pubKeys) == 0 {
		return "", fmt.Errorf("pas de clé publique")
	}

	aesKey := make([]byte, 32)
	io.ReadFull(rand.Reader, aesKey)

	encPayload, err := EncryptAES(aesKey, plaintext)
	if err != nil {
		return "", err
	}

	// encrypt AES key once per group member
	var encKeys []string
	for _, pk := range pubKeys {
		encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pk, aesKey, nil)
		if err != nil {
			return "", err
		}
		encKeys = append(encKeys, base64.StdEncoding.EncodeToString(encKey))
	}

	return strings.Join(encKeys, ";") + ":" + base64.StdEncoding.EncodeToString(encPayload), nil
}

// BroadcastDecrypt try each RSA key until one works, also handles old single-key format
func BroadcastDecrypt(message string, privateKey *rsa.PrivateKey) ([]byte, error) {
	parts := strings.SplitN(message, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("format invalide")
	}

	encPayload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	encKeyStrings := strings.Split(parts[0], ";")

	var aesKey []byte
	for _, encKeyB64 := range encKeyStrings {
		encKey, err := base64.StdEncoding.DecodeString(encKeyB64)
		if err != nil {
			continue
		}
		key, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encKey, nil)
		if err == nil {
			aesKey = key
			break
		}
	}
	if aesKey == nil {
		return nil, fmt.Errorf("aucune clé correspondante")
	}

	return DecryptAES(aesKey, encPayload)
}
