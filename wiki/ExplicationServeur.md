# Wiki Directory Server

serveur.go implements the Directory Server that maintains a list of active nodes in the DOR network using a SQLite database.

The server listens on TCP port 8080 and handles node registration, deregistration and list requests.

# Architecture

# Architecture

The Directory Server listens on TCP port 8080 and stores active nodes in a SQLite database. Nodes connect to the server to register (INIT), unregister (QUIT) or retrieve the list of available nodes (GET_LIST).

# TCP Protocol

All messages are sent over TCP and terminated by `\n`.

## Node → Server

| Command | Format | Response |
|---------|--------|----------|
| INIT | `INIT:id:port:key\n` | `INIT_ACK:id\n` |
| QUIT | `QUIT:id\n` | *(none)* |
| GET_LIST | `GET_LIST\n` | `LIST:id\|port\|key,...\n` |

## Server Terminal Commands

| Command | Description |
|---------|-------------|
| LIST | Display registered nodes |
| QUIT | Shutdown the server |

# Server startup

```go
func main()
```

Initializes the TCP listener, SQLite database, and starts accepting connections.

Startup sequence:
1. Listen on TCP port 8080
2. Connect to SQLite database (`test.db`)
3. Initialize `nodes` table if not exists
4. Clear `nodes` table (remove stale entries from previous run)
5. Start accepting connections in a goroutine
6. Read terminal commands from stdin

# Accept connections

```go
func acceptConnections(listener net.Listener)
```

Listens for incoming TCP connections and spawns a goroutine for each one.

Arguments:
- listener (net.Listener)

# Handle a connection

```go
func handleConnection(conn net.Conn)
```

Reads one message from the connection, parses the command and executes it.

Supported commands:
- `INIT` → Adds node to database, responds with `INIT_ACK`
- `QUIT` → Removes node from database
- `GET_LIST` → Sends the list of registered nodes

Arguments:
- conn (net.Conn)

# Get nodes list as string

```go
func getNodesList() string
```

Returns a formatted string of all nodes from the database.

Format: `LIST:id1|port1|key1,id2|port2|key2\n`

If no nodes: `LIST:empty\n`

Arguments:
- None

# Show nodes in terminal

```go
func showNodes()
```

Prints all registered nodes to the server terminal.

Arguments:
- None

# Node lifecycle

```
1. Node starts       → go run main.go node1 9001
2. Node registers    → sends INIT:node1:9001:12345
3. Server stores     → SQLite INSERT
4. Server responds   → INIT_ACK:node1
5. Node is active    → can send MSG, RELAY, GET_LIST
6. Node quits        → sends QUIT:node1
7. Server removes    → SQLite DELETE
```

# How to run

## Terminal 1 - Server
```bash
cd node_server/List_Serveur
go run serveur.go
```

## Terminal 2, 3, 4... - Nodes
```bash
cd node_server/node
go run main.go node1 9001
go run main.go node2 9002
go run main.go node3 9003
```