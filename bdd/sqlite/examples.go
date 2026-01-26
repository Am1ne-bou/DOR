package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Connexion
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		panic(err)
	}

	defer db.Close()

	// Création d'une table
	sqlStmt := `
    CREATE TABLE IF NOT EXISTS nodes (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        name TEXT
    );
    `
	_, err = db.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}

	fmt.Println("Table 'nodes' created successfully")

	// Ajout de valeurs
	_, err = db.Exec("INSERT INTO nodes(name) VALUES(?)", "Exemple")
	if err != nil {
		panic(err)
	}

	fmt.Println("Value added into 'nodes' successfully")

	// Lecture valeurs
	rows, err := db.Query("SELECT id, name FROM nodes")
	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Node: %d, %s\n", id, name)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
}
