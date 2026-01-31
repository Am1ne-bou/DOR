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
        name TEXT,
		port INTEGER,
		KEY INTEGER
    );
    `
	_, err := Db.Exec(sqlStmt)
	if err != nil {
		return err
	}

	return nil
}

func AddNode(node *model.Node) error {
	id := node.ID
	port := node.Port
	key := node.Key

	_, err := Db.Exec("INSERT INTO nodes(name, port, key) VALUES(?, ?, ?)", id, port, key)
	if err != nil {
		return err
	}

	return nil
}

func GetNodesList() ([]model.Node, error) {

	var nodes []model.Node

	rows, err := Db.Query("SELECT name, port, key FROM nodes")
	if err != nil {
		return []model.Node{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var name string
		var port int
		var key int
		err = rows.Scan(&name, &port, &key)

		if err != nil {
			return []model.Node{}, err
		}

		nodes = append(nodes, model.Node{ID: name, Port: port, Key: key})
	}

	if err = rows.Err(); err != nil {
		return []model.Node{}, err
	}

	return nodes, nil
}

func RemoveNode(nodeID string) error {

	_, err := Db.Exec("DELETE FROM nodes WHERE name = ?", nodeID)
	if err != nil {
		return err
	}

	return nil
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
