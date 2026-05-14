package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"

	"project/node_server/model"
)

func calculateNodeScore(n model.NodeInfo) float64 {
	return (float64(n.AvailabilityScore) * WeightAvailability) + (float64(n.NetworkScore) * WeightNetwork)
}

func sortNodesByScore(nodes []model.NodeInfo) []model.NodeInfo {
	sort.Slice(nodes, func(i, j int) bool {
		return calculateNodeScore(nodes[i]) > calculateNodeScore(nodes[j])
	})
	return nodes
}

func BuildSmartClusters(nodes []model.NodeInfo, numHops int, blacklist []string) ([]LayerGroup, float64) {
	var availableNodes []model.NodeInfo
	for _, n := range nodes {
		addr := fmt.Sprintf("%s:%d", n.Ip, n.Port)
		blacklisted := false
		for _, b := range blacklist {
			if b == addr {
				blacklisted = true
				break
			}
		}
		if !blacklisted {
			availableNodes = append(availableNodes, n)
		}
	}

	sortedNodes := sortNodesByScore(availableNodes)
	clusters := make([]LayerGroup, numHops)
	clusterScores := make([]float64, numHops)

	nodeIdx := 0
	for i := 0; i < numHops && nodeIdx < len(sortedNodes); i++ {
		remaining := len(sortedNodes) - nodeIdx
		pickOffset := 0
		if remaining >= 2 {
			r := rand.Intn(100)
			if r >= 70 && r < 85 {
				pickOffset = 1
			} else if r >= 85 && r < 95 && remaining >= 3 {
				pickOffset = 2
			} else if r >= 95 && remaining >= 4 {
				pickOffset = 3
			}
		}
		if pickOffset > 0 {
			sortedNodes[nodeIdx], sortedNodes[nodeIdx+pickOffset] = sortedNodes[nodeIdx+pickOffset], sortedNodes[nodeIdx]
		}
		n := sortedNodes[nodeIdx]
		pubKey := parsePublicKey(n.PublicKey)
		if pubKey == nil {
			nodeIdx++
			i--
			continue
		}
		clusters[i].Addrs = append(clusters[i].Addrs, n.Ip+":"+strconv.Itoa(n.Port))
		clusters[i].PubKeys = append(clusters[i].PubKeys, pubKey)
		clusterScores[i] += calculateNodeScore(n)
		nodeIdx++
	}

	for nodeIdx < len(sortedNodes) {
		targetIdx := 0
		minScore := 999999.0
		for i, s := range clusterScores {
			if s < minScore {
				minScore = s
				targetIdx = i
			}
		}
		n := sortedNodes[nodeIdx]
		pubKey := parsePublicKey(n.PublicKey)
		if pubKey != nil {
			clusters[targetIdx].Addrs = append(clusters[targetIdx].Addrs, n.Ip+":"+strconv.Itoa(n.Port))
			clusters[targetIdx].PubKeys = append(clusters[targetIdx].PubKeys, pubKey)
			clusterScores[targetIdx] += calculateNodeScore(n)
		}
		nodeIdx++
	}

	for i := range clusters {
		if len(clusters[i].Addrs) > 1 {
			rand.Shuffle(len(clusters[i].Addrs), func(j, k int) {
				clusters[i].Addrs[j], clusters[i].Addrs[k] = clusters[i].Addrs[k], clusters[i].Addrs[j]
				clusters[i].PubKeys[j], clusters[i].PubKeys[k] = clusters[i].PubKeys[k], clusters[i].PubKeys[j]
			})
		}
	}

	var totalScore float64
	for _, s := range clusterScores {
		totalScore += s
	}
	return clusters, totalScore / float64(numHops)
}
