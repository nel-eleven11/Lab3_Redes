package main

import (
	"fmt"
	"maps"
	"math"
	"sort"
	"time"
)

type Node struct {
	ID           string
	neighbors    map[string]int // neighbor -> cost
	routingTable *RoutingTable
	topologyDB   *TopologyDB
	sequenceNum  int
	packetQueue  chan *Packet
	lsaInterval  time.Duration
	simulation   *Simulation
}

func NewNode(id string, simulation *Simulation) *Node {
	return &Node{
		ID:           id,
		neighbors:    make(map[string]int),
		routingTable: NewRoutingTable(),
		topologyDB:   NewTopologyDB(),
		sequenceNum:  0,
		packetQueue:  make(chan *Packet, 100),
		lsaInterval:  time.Second * 10,
		simulation:   simulation,
	}
}

func (n *Node) AddNeighbor(neighbor string, cost int) {
	n.neighbors[neighbor] = cost
}

func (n *Node) RemoveNeighbor(neighbor string) {
	delete(n.neighbors, neighbor)
}

func (n *Node) GetNeighbors() map[string]int {
	return n.neighbors
}

func (n *Node) SendPacket(packet *Packet) {
	select {
	case n.packetQueue <- packet:
	default:
		fmt.Printf("Node %s: Packet queue full, dropping packet\n", n.ID)
	}
}

func (n *Node) GenerateLSA() *LSAData {
	n.sequenceNum++
	return &LSAData{
		OriginRouter: n.ID,
		Neighbors:    n.copyNeighbors(),
		SequenceNum:  n.sequenceNum,
		Timestamp:    time.Now(),
	}
}

func (n *Node) copyNeighbors() map[string]int {
	neighboursCopy := make(map[string]int)
	maps.Copy(neighboursCopy, n.neighbors)
	return neighboursCopy
}

func (n *Node) FloodLSA(lsa *LSAData) {
	packet := &Packet{
		Type:        LSA_PACKET,
		Source:      n.ID,
		Destination: "broadcast",
		Data:        lsa,
		TTL:         64,
		SequenceNum: lsa.SequenceNum,
	}

	// Send to all neighbors
	for neighbor := range n.neighbors {
		if neighbor != lsa.OriginRouter { // Don't send back to originator
			n.simulation.DeliverPacket(neighbor, packet)
		}
	}
}

func (n *Node) ProcessPacket(packet *Packet) {
	switch packet.Type {
	case LSA_PACKET:
		n.processLSA(packet)
	case DATA_PACKET:
		n.processDataPacket(packet)
	}
}

func (n *Node) processLSA(packet *Packet) {
	lsa := packet.Data.(*LSAData)

	// Update topology database
	if n.topologyDB.UpdateLSA(lsa) {
		fmt.Printf("Node %s: Received new LSA from %s (seq: %d)\n",
			n.ID, lsa.OriginRouter, lsa.SequenceNum)

		// Flood to neighbors (except source)
		for neighbor := range n.neighbors {
			if neighbor != packet.Source {
				n.simulation.DeliverPacket(neighbor, packet)
			}
		}

		// Recalculate routing table
		n.calculateRoutingTable()
	}
}

func (n *Node) processDataPacket(packet *Packet) {
	if packet.Destination == n.ID {
		fmt.Printf("Node %s: Received data packet from %s: %v\n",
			n.ID, packet.Source, packet.Data)
		return
	}

	// Forward packet
	if route, exists := n.routingTable.GetRoute(packet.Destination); exists {
		packet.TTL--
		if packet.TTL > 0 {
			n.simulation.DeliverPacket(route.NextHop, packet)
		}
	} else {
		fmt.Printf("Node %s: No route to %s, dropping packet\n",
			n.ID, packet.Destination)
	}
}

func (n *Node) calculateRoutingTable() {
	result := RunDijkstra(n.topologyDB, n.ID)
	n.routingTable.Clear()

	for dest, cost := range result.Distances {
		if dest != n.ID && cost != math.MaxInt32 {
			nextHop := n.findNextHop(dest, result.Previous)
			if nextHop != "" {
				n.routingTable.AddRoute(dest, nextHop, cost)
			}
		}
	}
}

func (n *Node) findNextHop(dest string, previous map[string]string) string {
	current := dest
	for {
		prev, exists := previous[current]
		if !exists {
			return ""
		}
		if prev == n.ID {
			return current
		}
		current = prev
	}
}

func (n *Node) Start() {
	// Send initial LSA
	lsa := n.GenerateLSA()
	n.topologyDB.UpdateLSA(lsa)
	n.FloodLSA(lsa)

	// Start periodic LSA generation
	ticker := time.NewTicker(n.lsaInterval)

	go func() {
		for {
			select {
			case packet := <-n.packetQueue:
				n.ProcessPacket(packet)
			case <-ticker.C:
				lsa := n.GenerateLSA()
				n.topologyDB.UpdateLSA(lsa)
				n.FloodLSA(lsa)
			}
		}
	}()
}

func (n *Node) SendDataPacket(dest string, data any) {
	packet := &Packet{
		Type:        DATA_PACKET,
		Source:      n.ID,
		Destination: dest,
		Data:        data,
		TTL:         64,
		SequenceNum: n.sequenceNum,
	}

	if route, exists := n.routingTable.GetRoute(dest); exists {
		n.simulation.DeliverPacket(route.NextHop, packet)
	} else {
		fmt.Printf("Node %s: No route to %s\n", n.ID, dest)
	}
}

func (n *Node) PrintRoutingTable() {
	fmt.Printf("\n=== Routing Table for Node %s ===\n", n.ID)
	fmt.Printf("%-12s %-12s %-8s\n", "Destination", "Next Hop", "Cost")
	fmt.Printf("%-12s %-12s %-8s\n", "-----------", "--------", "----")

	routes := n.routingTable.GetAllRoutes()

	// Sort destinations for consistent output
	var destinations []string
	for dest := range routes {
		destinations = append(destinations, dest)
	}
	sort.Strings(destinations)

	for _, dest := range destinations {
		route := routes[dest]
		if route.Valid {
			fmt.Printf("%-12s %-12s %-8d\n", dest, route.NextHop, route.Cost)
		}
	}
	fmt.Println()
}
