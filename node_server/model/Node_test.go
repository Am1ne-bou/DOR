package model

import (
	"bytes"
	"sync"
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

func TestSeenMsgID_NewNode(t *testing.T) {
	n := &Node{}
	if n.seenMsg("msg-1") {
		t.Fatal("fresh node: msg-1 should not be seen")
	}
	if !n.seenMsg("msg-1") {
		t.Fatal("msg-1 should be marked seen after first call")
	}
	if n.seenMsg("msg-2") {
		t.Fatal("msg-2 is a different ID, should not be seen")
	}
}

func TestSeenMsgID_Concurrent(t *testing.T) {
	// only one goroutine should win the first-seen race
	n := &Node{}
	const workers = 50
	results := make([]bool, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = n.seenMsg("shared-msg")
		}(i)
	}
	wg.Wait()
	falseCount := 0
	for _, r := range results {
		if !r {
			falseCount++
		}
	}
	if falseCount != 1 {
		t.Fatalf("expected exactly 1 goroutine to see false, got %d", falseCount)
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
