package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"project/node_server/model"
)

func GetScoresFromProfile(profile string) (int, int) {
	switch profile {
	case "server":
		return 100, 100
	case "laptop_WIFI7":
		return 60, 85
	case "smartphone_4G":
		return 30, 75
	case "smartphone_2G":
		return 20, 10
	default:
		return 15, 15
	}
}

func Encapsulator_func(
	message string,
	route []string,
	returnRoute []string,
	publicKeys *KeyCache,
	serverAddr string,
	senderAddr string,
) (string, string, string, error) {
	allNodes := append([]string{}, route...)
	if returnRoute != nil {
		allNodes = append(allNodes, returnRoute...)
	}
	for _, port := range allNodes {
		cached, ok := publicKeys.get(port)
		if !ok || time.Now().After(cached.ExpiresAt) {
			key, err := FetchKeyFromServer(port, serverAddr)
			if err != nil {
				return "", "", "", fmt.Errorf("error fetching public key for %s: %v", port, err)
			}
			publicKeys.set(port, CachedKey{Key: key, ExpiresAt: time.Now().Add(30 * time.Second)})
		}
	}

	msgID := model.GenerateMsgID()
	nackArray := []string{}
	for range len(route) {
		nackArray = append(nackArray, model.GenerateMsgID("nack"))
	}

	var returnOnion string
	var firstReturnHop string

	if returnRoute != nil && len(returnRoute) > 0 {
		firstReturnHop = returnRoute[0]
		innerLayer := &model.OnionLayer{Type: "ACK", MsgID: msgID}
		invNackArray := []string{}
		for i := range len(route) {
			invNackArray = append(invNackArray, nackArray[len(nackArray)-i-1])
		}
		returnPayload, err := encryptOnionLayers(innerLayer, returnRoute, publicKeys, returnRoute[len(returnRoute)-1], invNackArray)
		if err != nil {
			return "", "", "", fmt.Errorf("error building return onion: %v", err)
		}
		returnOnion = returnPayload
	}

	var innerLayer *model.OnionLayer
	if returnRoute != nil {
		innerLayer = &model.OnionLayer{
			Type: "FINAL", MsgID: msgID,
			Next: firstReturnHop, Data: returnOnion, Message: message,
		}
	} else {
		innerLayer = &model.OnionLayer{Type: "FINAL", MsgID: msgID, Message: message}
	}

	forwardPayload, err := encryptOnionLayers(innerLayer, route, publicKeys, senderAddr, nackArray)
	if err != nil {
		return "", "", "", fmt.Errorf("error building forward onion: %v", err)
	}
	return forwardPayload, msgID, nackArray[0], nil
}

func encryptOnionLayers(
	innerLayer *model.OnionLayer,
	route []string,
	publicKeys *KeyCache,
	senderAddr string,
	nackArray []string,
) (string, error) {
	innerLayerString := innerLayer.OnionlayerToString()
	lastCached, _ := publicKeys.get(route[len(route)-1])
	currentPayload, err := encryptForNode([]byte(innerLayerString), lastCached.Key)
	if err != nil {
		return "", err
	}
	for i := len(route) - 2; i >= 0; i-- {
		prevNode := senderAddr
		if i > 0 {
			prevNode = route[i-1]
		}
		relayLayer := &model.OnionLayer{
			Type:  "RELAY",
			MsgID: nackArray[i] + ":" + nackArray[i+1],
			Next:  route[i+1],
			From:  prevNode,
			Data:  currentPayload,
		}
		nodeCached, _ := publicKeys.get(route[i])
		currentPayload, err = encryptForNode([]byte(relayLayer.OnionlayerToString()), nodeCached.Key)
		if err != nil {
			return "", err
		}
	}
	return currentPayload, nil
}

func encryptForNode(plaintext []byte, pubKey *rsa.PublicKey) (string, error) {
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	encPlaintext, err := model.EncryptAES(aesKey, plaintext)
	if err != nil {
		return "", err
	}
	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, aesKey, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encKey) + ":" +
		base64.StdEncoding.EncodeToString(encPlaintext), nil
}
