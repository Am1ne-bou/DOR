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

	expected := "FINAL|msg-456||192.168.1.2:80||Decrypted message"
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
	
	if len(id2) != 11 { // "test-" is 5 chars + 6 digits = 11
		t.Errorf("Expected ID length of 11, got %d", len(id2))
	}
}
