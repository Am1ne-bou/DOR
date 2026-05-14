package data

import (
	"path/filepath"
	"testing"

	"project/node_server/model"
)

// Full SQLite roundtrip: add -> list -> update key -> remove.
func TestDataLayerRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	if err := Connect(path); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer Close()

	if err := InitTable(); err != nil {
		t.Fatalf("InitTable: %v", err)
	}

	node := &model.NodeInfo{
		Uuid: "uuid-1", Name: "alpha", Ip: "127.0.0.1",
		Port: 9000, PublicKey: "key1", AvailabilityScore: 80, NetworkScore: 90,
	}
	if err := AddNode(node); err != nil {
		t.Fatalf("AddNode: %v", err)
	}

	nodes, err := GetNodesList(10)
	if err != nil {
		t.Fatalf("GetNodesList: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Name != "alpha" {
		t.Fatalf("expected 1 node 'alpha', got %v", nodes)
	}

	if err := UpdateNodeKey("alpha", "key2"); err != nil {
		t.Fatalf("UpdateNodeKey: %v", err)
	}
	nodes, _ = GetNodesList(10)
	if nodes[0].PublicKey != "key2" {
		t.Errorf("UpdateNodeKey: expected key2, got %s", nodes[0].PublicKey)
	}

	if err := RemoveNode("alpha"); err != nil {
		t.Fatalf("RemoveNode: %v", err)
	}
	nodes, _ = GetNodesList(10)
	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes after remove, got %d", len(nodes))
	}
}

// ClearTable should reset the table and autoincrement sequence.
func TestClearTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	if err := Connect(path); err != nil {
		t.Fatal(err)
	}
	defer Close()
	if err := InitTable(); err != nil {
		t.Fatal(err)
	}

	for i, name := range []string{"n1", "n2", "n3"} {
		_ = AddNode(&model.NodeInfo{Uuid: name, Name: name, Port: 9000 + i})
	}

	if err := ClearTable(); err != nil {
		t.Fatalf("ClearTable: %v", err)
	}
	nodes, _ := GetNodesList(10)
	if len(nodes) != 0 {
		t.Fatalf("expected empty table after clear, got %d rows", len(nodes))
	}
}
