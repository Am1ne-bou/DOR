package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
)

func main() {
	// Hashage SHA 256
	h := sha256.New()
	h.Write([]byte("hello\n"))
	//fmt.Printf("Hash-1: %x\n", h.Sum(nil))

	h.Write([]byte("world\n"))
	//fmt.Printf("Hash-2: %x\n", h.Sum(nil))

	// Générer une clé 128 bits  /!\ À éviter car facile à trouver
	//key := rand.Text()
	//fmt.Printf("Clé 128 bits: %v\n", key)

	// Générer une clé privée RSA 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating RSA key: %s\n", err)
		return
	}
	//fmt.Printf("Private: %v\n", privateKey)

	// Générer une clé public à partir de la clé privée
	publicKey := privateKey.PublicKey
	//mt.Printf("Public: %v\n", publicKey)

	// Chiffrer un message avec RSA
	msg := "Ceci est un message secret"
	fmt.Println("Message à chiffrer:", msg)

	encryptedBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &publicKey, []byte(msg), nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Encrypted message: %s\n", encryptedBytes)

	// Déchiffer un message RSA
	decryptedBytes, err := privateKey.Decrypt(nil, encryptedBytes, &rsa.OAEPOptions{Hash: crypto.SHA256})
	if err != nil {
		panic(err)
	}
	fmt.Println("decrypted message: ", string(decryptedBytes))
}
