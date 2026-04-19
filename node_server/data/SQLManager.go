package data

import (
	"database/sql"
	"strconv"
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
	return err
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
	rows, err := Db.Query("SELECT uuid, name, ip, port, publicKey, availability_score, network_score FROM nodes ORDER BY RANDOM() LIMIT ?", limit)
	if err != nil {
		return []model.NodeInfo{}, err
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

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

func getAddr(nodeId string) (string, error) {
	var out string

	rows, err := Db.Query("SELECT DISTINCT ip, port FROM nodes WHERE name = ?", nodeId)
	if err != nil {
		return "", err
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	for rows.Next() {
		var ip string
		var port int

		err = rows.Scan(&ip, &port)
		out = ip + ":" + strconv.Itoa(port)
	}

	if err = rows.Err(); err != nil {
		return "", err
	}

	return out, nil
}

func UpdateNodeKey(name string, newKey string) error {
	_, err := Db.Exec("UPDATE nodes SET publicKey = ? WHERE name = ?", newKey, name)
	return err
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
		err := Db.Close()
		if err != nil {
			return
		}
	}
}
