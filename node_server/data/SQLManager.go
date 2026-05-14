package data

import (
	"database/sql"
	//"os"
	"project/node_server/model"

	_ "modernc.org/sqlite"
)

var Db *sql.DB = nil

func Connect(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	Db = db

	// WAL mode allows concurrent reads + one write, busy_timeout retries instead of failing instantly
	Db.Exec("PRAGMA journal_mode=WAL")
	Db.Exec("PRAGMA busy_timeout=5000")

	return nil
}

func InitTable() error {
	sqlStmt := `
    CREATE TABLE IF NOT EXISTS nodes (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uuid TEXT,
        name TEXT,
		ip TEXT,
		port INTEGER,
		publicKey TEXT,
		availability_score INTEGER DEFAULT 0,
		network_score INTEGER DEFAULT 0
    );
    `
	_, err := Db.Exec(sqlStmt)
	if err != nil {
		return err
	}

	// Migration : ajouter les colonnes si elles manquent (vieux .db)
	Db.Exec("ALTER TABLE nodes ADD COLUMN availability_score INTEGER DEFAULT 0")
	Db.Exec("ALTER TABLE nodes ADD COLUMN network_score INTEGER DEFAULT 0")

	return nil
}
func AddNode(node *model.NodeInfo) error {
	uuid := node.Uuid
	name := node.Name
	ip := node.Ip
	port := node.Port
	key := node.PublicKey
	availability_score := node.AvailabilityScore
	network_score := node.NetworkScore

	_, err := Db.Exec("INSERT INTO nodes(uuid, name, ip, port, publicKey, availability_score, network_score) VALUES(?, ?, ?, ?, ?, ?, ?)", uuid, name, ip, port, key, availability_score, network_score)
	if err != nil {
		return err
	}

	return nil
}

func GetNodesList(limit int) ([]model.NodeInfo, error) {
	var nodes []model.NodeInfo
	// weighted random: higher-scored nodes appear more often, entropy preserved
	rows, err := Db.Query("SELECT uuid, name, ip, port, publicKey, availability_score, network_score FROM nodes ORDER BY RANDOM() * (availability_score + network_score + 1) DESC LIMIT ?", limit)
	if err != nil {
		return []model.NodeInfo{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var n model.NodeInfo

		err = rows.Scan(&n.Uuid, &n.Name, &n.Ip, &n.Port, &n.PublicKey, &n.AvailabilityScore, &n.NetworkScore)
		if err != nil {
			continue
		}

		nodes = append(nodes, n)
	}

	if err = rows.Err(); err != nil {
		return []model.NodeInfo{}, err
	}

	return nodes, nil
}

func UpdateNodeKey(name string, newKey string) error {
	_, err := Db.Exec("UPDATE nodes SET publicKey = ? WHERE name = ?", newKey, name)
	return err
}

func RemoveNode(nodeID string) error {

	_, err := Db.Exec("DELETE FROM nodes WHERE name = ?", nodeID)

	return err
}

func UpdateAvailabilityScore(name string, score int) error {
	_, err := Db.Exec("UPDATE nodes SET availability_score = ? WHERE name = ?", score, name)
	return err
}

func ClearTable() error {
	if _, err := Db.Exec("DELETE FROM nodes"); err != nil {
		return err
	}
	_, err := Db.Exec("DELETE FROM sqlite_sequence WHERE name='nodes'")
	return err
}

func Close() {
	if Db != nil {
		err := Db.Close()
		if err != nil {
			return
		}
	}
}
