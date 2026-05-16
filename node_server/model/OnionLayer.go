package model

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

type OnionLayer struct {
	Type    string // RELAY, FINAL, ACK
	MsgID   string
	Next    string // RELAY  FINAL
	From    string // RELAY
	Data    string // RELAY  FINAL
	Frag    string // "fragID:i/n", empty for non-fragmented
	Message string // FINAL seulement
}

func (layer OnionLayer) OnionlayerToString() string {
	str := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", layer.Type, layer.MsgID, layer.Next, layer.From, layer.Data, layer.Frag, layer.Message)
	return str
}

func StringToOnionLayer(str string) (OnionLayer, error) {
	// 6-field: old wire format (no Frag); 7-field: new format -- both accepted
	parts := strings.SplitN(str, "|", 7)
	if len(parts) < 6 {
		return OnionLayer{}, fmt.Errorf("OnionLayer StringToOnionLayer Error Split")
	}
	ol := OnionLayer{
		Type:  parts[0],
		MsgID: parts[1],
		Next:  parts[2],
		From:  parts[3],
		Data:  parts[4],
	}
	if len(parts) == 7 {
		ol.Frag = parts[5]
		ol.Message = parts[6]
	} else {
		ol.Message = parts[5] // backward compat: pre-fragmentation messages
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
