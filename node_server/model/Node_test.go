package model

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptAES(t *testing.T) {
	// Generate a valid 32-byte AES key for AES-256
	key := []byte("12345678901234567890123456789012")
	plaintext := []byte("Hello, this is a secret message.")

	ciphertext, err := EncryptAES(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAES failed: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatalf("Ciphertext should be different from plaintext")
	}

	decrypted, err := DecryptAES(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptAES failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Expected decrypted text '%s', got '%s'", string(plaintext), string(decrypted))
	}
}

func TestDecryptAES_InvalidKey(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	wrongKey := []byte("00000000000000000000000000000000") // Different key
	plaintext := []byte("Hello, this is a secret message.")

	ciphertext, err := EncryptAES(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAES failed: %v", err)
	}

	_, err = DecryptAES(wrongKey, ciphertext)
	if err == nil {
		t.Fatalf("DecryptAES should have failed with the wrong key")
	}
}
