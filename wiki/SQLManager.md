# Wiki SQLManager

SQLManager.go implements multiples functions to manipulate data inside a SQLite data base.

Each node is identified using a UUID (Length = 128 bits, stored as a string).

# Establish a connection

```go
func Connect(path string) error
```
Establishes connection using a *sql.DB object.

Arguments:
- path (string), path of .db file. If not exists, it will be created. 

# Initialize the main table for node storage

```go
func InitTable() error
```
'nodes' table will be created only if doesn't exist. 
Table content:
- id, Integer autoincrement primary key
- name, Text
- port, Integer
- key, Integer public key

Arguments:
- None

# Add a node into the 'nodes'

```go
func AddNode(node *model.Node) error
```
Adds a model.Node object into 'nodes'.

Arguments:
- node (*model.Node)

# Remove a node into the 'nodes'

```go
func RemoveNode(nodeID string) error
```
Removes a model.Node object from 'nodes' using UUID.

Arguments:
- nodeID (string)

# Get all nodes stored inside 'nodes'

```go
func GetNodesList() ([]model.Node, error) 
```
Returns []model.Node with all data.

Arguments:
- None

# Close the connection

```go
func Close()
```
Closes the connection to the data base.

Arguments:
- None
