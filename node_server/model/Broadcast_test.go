package model

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func generateKeys(t *testing.T, n int) ([]*rsa.PrivateKey, []*rsa.PublicKey) {
	var privs []*rsa.PrivateKey
	var pubs []*rsa.PublicKey
	for i := 0; i < n; i++ {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("keygen %d: %v", i, err)
		}
		privs = append(privs, priv)
		pubs = append(pubs, &priv.PublicKey)
	}
	return privs, pubs
}

func TestParseAddresses_Single(t *testing.T) {
	r := ParseAddresses("192.168.1.1:9001")
	if len(r) != 1 || r[0] != "192.168.1.1:9001" {
		t.Errorf("got %v", r)
	}
}

func TestParseAddresses_Multiple(t *testing.T) {
	r := ParseAddresses("192.168.1.1:9001,192.168.1.2:9002,192.168.1.3:9003")
	if len(r) != 3 {
		t.Fatalf("expected 3, got %d", len(r))
	}
}

func TestParseAddresses_Empty(t *testing.T) {
	r := ParseAddresses("")
	if len(r) != 0 {
		t.Errorf("expected empty, got %v", r)
	}
}

func TestJoinAddresses(t *testing.T) {
	r := JoinAddresses([]string{"a:1", "b:2", "c:3"})
	if r != "a:1,b:2,c:3" {
		t.Errorf("got %s", r)
	}
}

func TestBroadcastEncryptDecrypt_SingleKey(t *testing.T) {
	privs, pubs := generateKeys(t, 1)
	msg := []byte("test single key")

	enc, err := BroadcastEncrypt(msg, pubs)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := BroadcastDecrypt(enc, privs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(msg, dec) {
		t.Errorf("got '%s'", dec)
	}
}

func TestBroadcastEncryptDecrypt_ThreeKeys(t *testing.T) {
	privs, pubs := generateKeys(t, 3)
	msg := []byte("message pour le groupe")

	enc, err := BroadcastEncrypt(msg, pubs)
	if err != nil {
		t.Fatal(err)
	}
	// each member should decrypt
	for i, priv := range privs {
		dec, err := BroadcastDecrypt(enc, priv)
		if err != nil {
			t.Fatalf("member %d: %v", i, err)
		}
		if !bytes.Equal(msg, dec) {
			t.Errorf("member %d: got '%s'", i, dec)
		}
	}
}

func TestBroadcastEncryptDecrypt_FiveKeys(t *testing.T) {
	privs, pubs := generateKeys(t, 5)
	msg := []byte("test 5 candidats")

	enc, err := BroadcastEncrypt(msg, pubs)
	if err != nil {
		t.Fatal(err)
	}
	for i, priv := range privs {
		dec, err := BroadcastDecrypt(enc, priv)
		if err != nil {
			t.Fatalf("member %d: %v", i, err)
		}
		if !bytes.Equal(msg, dec) {
			t.Errorf("member %d: got '%s'", i, dec)
		}
	}
}

func TestBroadcastDecrypt_WrongKey(t *testing.T) {
	_, pubs := generateKeys(t, 3)
	outsider, _ := generateKeys(t, 1)

	enc, err := BroadcastEncrypt([]byte("pas pour toi"), pubs)
	if err != nil {
		t.Fatal(err)
	}
	_, err = BroadcastDecrypt(enc, outsider[0])
	if err == nil {
		t.Fatal("outsider should not decrypt")
	}
}

func TestBroadcastDecrypt_LegacyFormat(t *testing.T) {
	privs, pubs := generateKeys(t, 1)
	msg := []byte("ancien format")

	enc, err := BroadcastEncrypt(msg, pubs)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := BroadcastDecrypt(enc, privs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(msg, dec) {
		t.Errorf("got '%s'", dec)
	}
}

func TestBroadcastEncrypt_NoKeys(t *testing.T) {
	_, err := BroadcastEncrypt([]byte("test"), []*rsa.PublicKey{})
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestBroadcastDecrypt_InvalidFormat(t *testing.T) {
	priv, _ := generateKeys(t, 1)
	_, err := BroadcastDecrypt("invalid", priv[0])
	if err == nil {
		t.Fatal("should fail")
	}
}
