// TCP

package main

import (
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte("Hello, server!"))
	if err != nil {
		fmt.Println("Write error:", err)
		return
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}

	fmt.Println("Server says:", string(buffer[:n]))
}

// UDP

package main

import (
 "net"
 "fmt"
)

func main() {
 // Resolve the server's address.
 addr, err := net.ResolveUDPAddr("udp", "localhost:8080")
 if err != nil {
  fmt.Println(err)
  return
 }

 // Dial a connection to the resolved address.
 conn, err := net.DialUDP("udp", nil, addr)
 if err != nil {
  fmt.Println(err)
  return
 }
 defer conn.Close()

 // Write a message to the server.
 conn.Write([]byte("Hello, server!"))
 buffer := make([]byte, 1024)
 // Read the response from the server.
 conn.Read(buffer)
 fmt.Println(string(buffer))
}