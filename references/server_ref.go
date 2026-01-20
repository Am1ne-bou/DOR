// TCP

package main

import (
	"fmt"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server listening on port 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}

	message := string(buffer[:n])
	fmt.Println("Received:", message)

	response := "Received: " + message
	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Write error:", err)
	}
}

// UDP

package main

import (
 "net"
 "fmt"
)

func main() {
 // Listen for incoming UDP packets on port 8080.
 conn, err := net.ListenPacket("udp", ":8080")
 if err != nil {
  fmt.Println(err)
  return
 }
 defer conn.Close()

 buffer := make([]byte, 1024)
 // Read the incoming packet data into the buffer.
 n, addr, err := conn.ReadFrom(buffer)
 if err != nil {
  fmt.Println(err)
  return
 }
 fmt.Println("Received: ", string(buffer[:n]))
 // Write a response to the client's address.
 conn.WriteTo([]byte("Message received!"), addr)
}

// Multiple clients TCP

package main

import (
 "net"
 "fmt"
)

func main() {
 // Listen on TCP port 8080.
 listener, err := net.Listen("tcp", ":8080")
 if err != nil {
  fmt.Println(err)
  return
 }
 defer listener.Close()

 for {
  // Accept a connection.
  conn, err := listener.Accept()
  if err != nil {
   fmt.Println(err)
   continue
  }
  // Handle the connection in a new goroutine.
  go handleConnection(conn)
 }
}

func handleConnection(conn net.Conn) {
 defer conn.Close()
 buffer := make([]byte, 1024)
 // Read the incoming connection.
 conn.Read(buffer)
 fmt.Println("Received:", string(buffer))
 // Respond to the client.
 conn.Write([]byte("Message received!"))
}

// HTTP Server with Gorilla Mux

package main

import (
 "fmt"
 "github.com/gorilla/mux"
 "net/http"
)

func main() {
 // Create a new router.
 r := mux.NewRouter()
 // Register a handler function for the root path.
 r.HandleFunc("/", homeHandler)
 http.ListenAndServe(":8080", r)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
 // Respond with a welcome message.
 fmt.Fprint(w, "Welcome to Home!")
}

// HTTPS Server

package main

import (
 "net/http"
 "log"
)

func main() {
 http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
  // Respond with a message.
  w.Write([]byte("Hello, this is an HTTPS server!"))
 })
 // Use the cert.pem and key.pem files to secure the server.
 log.Fatal(http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil))
}

// Current protocol over TCP

package main

import (
 "net"
 "strings"
)

func main() {
 // Listen on TCP port 8080.
 listener, err := net.Listen("tcp", ":8080")
 if err != nil {
  panic(err)
 }
 defer listener.Close()

 for {
  // Accept a connection.
  conn, err := listener.Accept()
  if err != nil {
   panic(err)
  }
  // Handle the connection in a new goroutine.
  go handleConnection(conn)
 }
}

func handleConnection(conn net.Conn) {
 defer conn.Close()
 buffer := make([]byte, 1024)
 // Read the incoming connection.
 conn.Read(buffer)
 // Process custom protocol command.
 cmd := strings.TrimSpace(string(buffer))
 if cmd == "TIME" {
  conn.Write([]byte("The current time is: " + time.Now().String()))
 } else {
  conn.Write([]byte("Unknown command"))
 }
}

// WebSockets with Gorilla WebSocket

package main

import (
 "github.com/gorilla/websocket"
 "net/http"
)

var upgrader = websocket.Upgrader{
 ReadBufferSize:  1024,
 WriteBufferSize: 1024,
}

func handler(w http.ResponseWriter, r *http.Request) {
 conn, err := upgrader.Upgrade(w, r, nil)
 if err != nil {
  http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
  return
 }
 defer conn.Close()

 for {
  messageType, p, err := conn.ReadMessage()
  if err != nil {
   return
  }
  // Echo the message back to the client.
  conn.WriteMessage(messageType, p)
 }
}

func main() {
 http.HandleFunc("/", handler)
 http.ListenAndServe(":8080", nil)
}

// Connections Timeout

package main

import (
 "context"
 "fmt"
 "net"
 "time"
)

func main() {
 // Create a context with a timeout of 2 seconds
 ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
 defer cancel()

 // Dialer using the context
 dialer := net.Dialer{}
 conn, err := dialer.DialContext(ctx, "tcp", "localhost:8080")
 if err != nil {
  panic(err)
 }

 buffer := make([]byte, 1024)
 _, err = conn.Read(buffer)
 if err == nil {
  fmt.Println("Received:", string(buffer))
 } else {
  fmt.Println("Connection error:", err)
 }
}

// Rate limiting with golang.org/x/time/rate

package main

import (
 "golang.org/x/time/rate"
 "net/http"
 "time"
)

// Define a rate limiter allowing two requests per second with a burst capacity of five.
var limiter = rate.NewLimiter(2, 5)

func handler(w http.ResponseWriter, r *http.Request) {
 // Check if request is allowed by the rate limiter.
 if !limiter.Allow() {
  http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
  return
 }
 w.Write([]byte("Welcome!"))
}

func main() {
 http.HandleFunc("/", handler)
 http.ListenAndServe(":8080", nil)
}