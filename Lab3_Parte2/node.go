package main

import "time"

type Node struct {
	ID        string
	neighbors map[string]int // neighbor -> cost
	// routingTable *RoutingTable
	// topologyDB   *TopologyDB
	sequenceNum int
	lsaInterval time.Duration
}

func NewNode(ID string, neighbours map[string]int) *Node {
	return &Node{
		ID:        ID,
		neighbors: neighbours,
	}
}
