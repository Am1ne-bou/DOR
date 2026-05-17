package model

import (
	"strings"
	"testing"
)

func TestStringToOnionLayer_Valid(t *testing.T) {
	str := "RELAY|msg-123|192.168.1.1:80|192.168.1.2:80|datadata|hello"
	ol, err := StringToOnionLayer(str)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ol.Type != "RELAY" || ol.MsgID != "msg-123" || ol.Next != "192.168.1.1:80" ||
		ol.From != "192.168.1.2:80" || ol.Data != "datadata" || ol.Message != "hello" {
		t.Errorf("Parsed layer fields do not match expected: %+v", ol)
	}
}

func TestStringToOnionLayer_Invalid(t *testing.T) {
	str := "RELAY|msg-123|192.168.1.1:80" // Not enough parts
	_, err := StringToOnionLayer(str)
	if err == nil {
		t.Fatalf("Expected error for invalid string split, got nil")
	}
}

func TestOnionlayerToString(t *testing.T) {
	ol := OnionLayer{
		Type:    "FINAL",
		MsgID:   "msg-456",
		Next:    "",
		From:    "192.168.1.2:80",
		Data:    "",
		Message: "Decrypted message",
	}

	expected := "FINAL|msg-456||192.168.1.2:80|||Decrypted message"
	if out := ol.OnionlayerToString(); out != expected {
		t.Errorf("Expected %s, got %s", expected, out)
	}
}

func TestGenerateMsgID(t *testing.T) {
	id1 := GenerateMsgID()
	if !strings.HasPrefix(id1, "msg-") {
		t.Errorf("Expected ID to start with 'msg-', got %s", id1)
	}

	id2 := GenerateMsgID("test")
	if !strings.HasPrefix(id2, "test-") {
		t.Errorf("Expected ID to start with 'test-', got %s", id2)
	}

	if len(id2) != 15 { // "test-" is 5 chars + 10 digits = 15
		t.Errorf("Expected ID length of 11, got %d", len(id2))
	}
}

func TestGenerateMsgID_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := GenerateMsgID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate MsgID generated: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestStringToOnionLayer_PipeInMessage(t *testing.T) {
	// SplitN(6) leaves everything after the 5th pipe in parts[5]
	str := "FINAL|msg-1|next|from|data||hello|world|pipe" // empty Frag, Message contains pipes
	ol, err := StringToOnionLayer(str)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ol.Message != "hello|world|pipe" {
		t.Errorf("expected message with pipes, got '%s'", ol.Message)
	}
}

// TODO: add benchmark for AES enc/dec and RSA enc/dec -- go test -bench=.

func FuzzStringToOnionLayer(f *testing.F) {
	f.Add("RELAY|msg-123|192.168.1.1:80|192.168.1.2:80|datadata|hello")
	f.Add("FINAL|msg-456||192.168.1.2:80||hello world")
	f.Add("ACK|msg-789||||")
	f.Add("")
	f.Add("|||||||")
	f.Fuzz(func(t *testing.T, s string) {
		ol, err := StringToOnionLayer(s)
		if err != nil {
			return
		}
		// round-trip must not panic
		_ = ol.OnionlayerToString()
	})
}
