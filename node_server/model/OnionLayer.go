package model

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// TODO: add TTL field to prevent infinite relay loops if a node routes back to itself
type OnionLayer struct {
	Type    string // RELAY, FINAL, ACK
	MsgID   string
	Next    string // RELAY  FINAL
	From    string // RELAY
	Data    string // RELAY  FINAL
	Message string // FINAL seulement
}

// TODO: add message fragmentation -- RSA-OAEP(2048) caps plaintext at ~214 bytes, large messages silently fail
func (layer OnionLayer) OnionlayerToString() string {
	str := fmt.Sprintf("%s|%s|%s|%s|%s|%s", layer.Type, layer.MsgID, layer.Next, layer.From, layer.Data, layer.Message)
	return str
}

func StringToOnionLayer(str string) (OnionLayer, error) {
	parts := strings.SplitN(str, "|", 6)
	if len(parts) != 6 {
		return OnionLayer{}, fmt.Errorf("OnionLayer StringToOnionLayer Error Split")
	}
	ol := OnionLayer{
		Type:    parts[0],
		MsgID:   parts[1],
		Next:    parts[2],
		From:    parts[3],
		Data:    parts[4],
		Message: parts[5],
	}
	return ol, nil
}

// TODO: switch to crypto/rand 128-bit UUID -- 10 decimal digits still has birthday collision risk at scale
func GenerateMsgID(prefix ...string) string {
	str := "msg-"
	if len(prefix) > 0 && prefix[0] != "" {
		str = prefix[0] + "-"
	}
	for i := 0; i < 10; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		str += string('0' + byte(n.Int64()))
	}
	return str
}
