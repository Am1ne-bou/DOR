package data

import (
	"database/sql"
	"project/node_server/model"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB = nil

func Connect(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}

	defer db.Close()

	return db, nil
}

func InitTable(db *sql.DB) error {
	sqlStmt := `
    CREATE TABLE IF NOT EXISTS nodes (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        name TEXT,
		port INTEGER,
		KEY INTEGER
    );
    `
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}

	return nil
}

func AddNode(node *model.Node) error {
	id := node.ID
	port := node.Port
	key := 12345

	_, err := db.Exec("INSERT INTO nodes(name, port, key) VALUES(?, ?, ?)", id, port, key)
	if err != nil {
		return err
	}

	return nil
}
