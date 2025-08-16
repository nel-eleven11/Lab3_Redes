package main

import (
	"fmt"
	"sort"
	"time"
)

type Simulation struct {
	nodes map[string]*Node
	links map[string]*Link
}

func NewSimulation() *Simulation {
	return &Simulation{
		nodes: make(map[string]*Node),
		links: make(map[string]*Link),
	}
}

func (s *Simulation) AddNode(id string) *Node {
	node := NewNode(id, s)
	s.nodes[id] = node
	return node
}

func (s *Simulation) AddLink(nodeA, nodeB string, cost int) {
	linkID := s.getLinkID(nodeA, nodeB)
	link := NewLink(nodeA, nodeB, cost)
	s.links[linkID] = link

	// Update neighbor tables
	if nodeObjA, exists := s.nodes[nodeA]; exists {
		nodeObjA.AddNeighbor(nodeB, cost)
	}
	if nodeObjB, exists := s.nodes[nodeB]; exists {
		nodeObjB.AddNeighbor(nodeA, cost)
	}
}

func (s *Simulation) RemoveLink(nodeA, nodeB string) {
	linkID := s.getLinkID(nodeA, nodeB)
	delete(s.links, linkID)

	// Update neighbor tables
	if nodeObjA, exists := s.nodes[nodeA]; exists {
		nodeObjA.RemoveNeighbor(nodeB)
	}
	if nodeObjB, exists := s.nodes[nodeB]; exists {
		nodeObjB.RemoveNeighbor(nodeA)
	}
}

func (s *Simulation) getLinkID(nodeA, nodeB string) string {
	if nodeA < nodeB {
		return nodeA + "-" + nodeB
	}
	return nodeB + "-" + nodeA
}

func (s *Simulation) DeliverPacket(nodeID string, packet *Packet) {
	if node, exists := s.nodes[nodeID]; exists {
		node.SendPacket(packet)
	}
}

func (s *Simulation) StartAllNodes() {
	for _, node := range s.nodes {
		node.Start()
	}
}

func (s *Simulation) PrintTopology() {
	fmt.Println("\n=== Network Topology ===")
	for linkID, link := range s.links {
		status := "UP"
		if !link.Up {
			status = "DOWN"
		}
		fmt.Printf("Link %s: %s <-> %s (Cost: %d, Status: %s)\n",
			linkID, link.NodeA, link.NodeB, link.Cost, status)
	}
	fmt.Println()
}

func (s *Simulation) PrintAllRoutingTables() {
	var nodeIDs []string
	for id := range s.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for _, id := range nodeIDs {
		s.nodes[id].PrintRoutingTable()
	}
}

func (s *Simulation) SendDataBetweenNodes(source, dest string, data any) {
	if node, exists := s.nodes[source]; exists {
		node.SendDataPacket(dest, data)
	} else {
		fmt.Printf("Source node %s not found\n", source)
	}
}

func main() {
	sim := NewSimulation()

	nodes := []string{"A", "B", "C", "D", "E"}
	for _, nodeID := range nodes {
		sim.AddNode(nodeID)
	}

	// Add links to create a network topology
	sim.AddLink("A", "B", 1)
	sim.AddLink("A", "C", 4)
	sim.AddLink("B", "C", 2)
	sim.AddLink("B", "D", 5)
	sim.AddLink("C", "D", 1)
	sim.AddLink("C", "E", 3)
	sim.AddLink("D", "E", 2)

	sim.PrintTopology()
	sim.StartAllNodes()

	fmt.Println("Starting simulation...")
	fmt.Println("Waiting for LSA propagation...")
	time.Sleep(2 * time.Second)

	fmt.Println("\n=== Initial Routing Tables ===")
	sim.PrintAllRoutingTables()

	fmt.Println("=== Sending Data Packets ===")
	sim.SendDataBetweenNodes("A", "E", "Hello from A to E!")
	sim.SendDataBetweenNodes("B", "D", "Message from B to D")
	sim.SendDataBetweenNodes("E", "A", "Reply from E to A")

	time.Sleep(1 * time.Second)

	fmt.Println("\n=== Simulating Link Failure: C-D ===")
	sim.RemoveLink("C", "D")

	// Trigger LSA updates
	for _, node := range sim.nodes {
		lsa := node.GenerateLSA()
		node.topologyDB.UpdateLSA(lsa)
		node.FloodLSA(lsa)
	}

	// Wait for convergence
	time.Sleep(2 * time.Second)

	fmt.Println("=== Routing Tables After Link Failure ===")
	sim.PrintAllRoutingTables()

	fmt.Println("=== Testing Connectivity After Failure ===")
	sim.SendDataBetweenNodes("A", "E", "Testing path after link failure")
	sim.SendDataBetweenNodes("C", "D", "Can C still reach D?")

	time.Sleep(1 * time.Second)

	fmt.Println("\n=== Restoring Link: C-D ===")
	sim.AddLink("C", "D", 1)

	for _, node := range sim.nodes {
		lsa := node.GenerateLSA()
		node.topologyDB.UpdateLSA(lsa)
		node.FloodLSA(lsa)
	}
	time.Sleep(2 * time.Second)

	fmt.Println("=== Final Routing Tables After Link Restoration ===")
	sim.PrintAllRoutingTables()

	fmt.Println("=== Simulation Complete ===")
}
