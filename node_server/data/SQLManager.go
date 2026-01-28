package data

import (
	"database/sql"
	"fmt"
	"os"

	"project/node_server/model"

	_ "github.com/mattn/go-sqlite3"
)

var Db *sql.DB = nil

func Connect(path string) {

	///fmt.Println(&Db, Db)

	db, err := sql.Open("sqlite3", path)

	Db = db
	if err != nil {
		fmt.Println(err)
	}

	//defer db.Close()

	return
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
	key := 12345

	_, err := Db.Exec("INSERT INTO nodes(name, port, key) VALUES(?, ?, ?)", id, port, key)
	if err != nil {
		return err
	}

	return nil
}

func Close() {
	if Db != nil {
		Db.Close()
		os.Remove("test.db")
	}
}
