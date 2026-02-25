package data

import (
	"database/sql"
	//"os"

	"project/node_server/model"

	_ "github.com/mattn/go-sqlite3"
)

var Db *sql.DB = nil

func Connect(path string) error {

	db, err := sql.Open("sqlite3", path)

	Db = db
	if err != nil {
		return err
	}

	return nil
}

func InitTable() error {
	sqlStmt := `
    CREATE TABLE IF NOT EXISTS nodes (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uuid TEXT,
        name TEXT,
		ip INTEGER,
		port TEXT,
		publicKey TEXT
    );
    `
	_, err := Db.Exec(sqlStmt)

	return err
}

func AddNode(node *model.NodeInfo) error {
	uuid := node.Uuid
	name := node.Name
	ip := node.Ip
	port := node.Port
	key := node.PublicKey

	_, err := Db.Exec("INSERT INTO nodes(uuid, name, ip, port, publicKey) VALUES(?, ?, ?, ?, ?)", uuid, name, ip, port, key)
	if err != nil {
		return err
	}

	return nil
}

func GetNodesList() ([]model.NodeInfo, error) {

	var nodes []model.NodeInfo

	rows, err := Db.Query("SELECT name, port, key FROM nodes")
	if err != nil {
		return []model.NodeInfo{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var name string
		var port int
		var key string
		err = rows.Scan(&name, &port, &key)

		if err != nil {
			return []model.NodeInfo{}, err
		}

		nodes = append(nodes, model.NodeInfo{Name: name, Port: port, PublicKey: key})
	}

	if err = rows.Err(); err != nil {
		return []model.NodeInfo{}, err
	}

	return nodes, nil
}

func RemoveNode(nodeID string) error {

	_, err := Db.Exec("DELETE FROM nodes WHERE name = ?", nodeID)

	return err
}

func ClearTable() error {
	Db.Exec("DELETE FROM nodes")
	_, err := Db.Exec("DELETE FROM sqlite_sequence WHERE name='nodes'")
	return err
}

func Close() {
	if Db != nil {
		Db.Close()
		//os.Remove("test.db")
	}
}
