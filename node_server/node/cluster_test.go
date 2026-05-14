package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"project/node_server/model"
)

func makeNodeInfo(t *testing.T, ip string, port int, sa, sn int) model.NodeInfo {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return model.NodeInfo{
		Name:              "node",
		Ip:                ip,
		Port:              port,
		PublicKey:         base64.StdEncoding.EncodeToString(pub),
		AvailabilityScore: sa,
		NetworkScore:      sn,
	}
}

func TestCalculateNodeScore(t *testing.T) {
	cases := []struct {
		sa, sn int
		want   float64
	}{
		{100, 100, 100.0},
		{0, 0, 0.0},
		{60, 40, 50.0},
	}
	for _, c := range cases {
		n := model.NodeInfo{AvailabilityScore: c.sa, NetworkScore: c.sn}
		got := calculateNodeScore(n)
		if got != c.want {
			t.Errorf("sa=%d sn=%d: got %f want %f", c.sa, c.sn, got, c.want)
		}
	}
}

func TestBuildSmartClusters_Shape(t *testing.T) {
	nodes := []model.NodeInfo{
		makeNodeInfo(t, "10.0.0.1", 9001, 80, 70),
		makeNodeInfo(t, "10.0.0.2", 9002, 60, 90),
		makeNodeInfo(t, "10.0.0.3", 9003, 50, 50),
		makeNodeInfo(t, "10.0.0.4", 9004, 40, 80),
		makeNodeInfo(t, "10.0.0.5", 9005, 30, 60),
	}

	numHops := 3
	clusters, score := BuildSmartClusters(nodes, numHops, nil)

	if len(clusters) != numHops {
		t.Fatalf("expected %d clusters, got %d", numHops, len(clusters))
	}
	if score <= 0 {
		t.Errorf("expected positive score, got %f", score)
	}
	for i, cl := range clusters {
		if len(cl.Addrs) == 0 {
			t.Errorf("cluster %d is empty", i)
		}
		if len(cl.Addrs) != len(cl.PubKeys) {
			t.Errorf("cluster %d: addrs/keys length mismatch", i)
		}
	}
}

func TestBuildSmartClusters_Blacklist(t *testing.T) {
	nodes := []model.NodeInfo{
		makeNodeInfo(t, "10.0.0.1", 9001, 80, 70),
		makeNodeInfo(t, "10.0.0.2", 9002, 60, 90),
		makeNodeInfo(t, "10.0.0.3", 9003, 50, 50),
		makeNodeInfo(t, "10.0.0.4", 9004, 40, 80),
		makeNodeInfo(t, "10.0.0.5", 9005, 30, 60),
	}
	blacklist := []string{"10.0.0.1:9001", "10.0.0.2:9002"}

	clusters, _ := BuildSmartClusters(nodes, 2, blacklist)
	for _, cl := range clusters {
		for _, addr := range cl.Addrs {
			for _, bl := range blacklist {
				if addr == bl {
					t.Errorf("blacklisted node %s appeared in cluster", addr)
				}
			}
		}
	}
}
