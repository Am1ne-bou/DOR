package model

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

type OnionLayer struct {
	Type       string // Type of the node
	MsgID      string // Id of the msg  "msg-123456"
	NextHop    string // Next node
	ReturnAddr string // Node before
	ReturnData string // onion layer of the return encrypted
	Message    string
	Payload    string // onion layer of the next node encrypted
}

func (layer OnionLayer) OnionlayerToString() string {
	str := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", layer.Type, layer.MsgID, layer.NextHop, layer.ReturnAddr, layer.ReturnData, layer.Message, layer.Payload)
	return str
}

func StringToOnionLayer(str string) (OnionLayer, error) {
	parts := strings.SplitN(str, "|", 7)
	if len(parts) != 7 {
		return OnionLayer{}, fmt.Errorf("OnionLayer StringToOnionLayer Error Split")
	}
	ol := OnionLayer{
		Type:       parts[0],
		MsgID:      parts[1],
		NextHop:    parts[2],
		ReturnAddr: parts[3],
		ReturnData: parts[4],
		Message:    parts[5],
		Payload:    parts[6],
	}
	///fmt.Printf("Debug Serializing layer: %s\n", str) // Debug log
	return ol, nil
}

func GenerateMsgID() string {
	str := "msg-"
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		str += string('0' + byte(n.Int64()))
	}
	return str
}
