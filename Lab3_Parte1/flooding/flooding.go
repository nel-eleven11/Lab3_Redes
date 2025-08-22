package main

import (
	"fmt"
	"github.com/nel-eleven11/Lab3_Redes/lib"
	"sync"
	"time"
)

type FloodingNode struct {
	ID          string
	Neighbors   map[string]*FloodingNode
	SeenPackets map[int]bool
	mutex       sync.RWMutex
}

func NewNode(id string) *FloodingNode {
	return &FloodingNode{
		ID:          id,
		Neighbors:   make(map[string]*FloodingNode),
		SeenPackets: make(map[int]bool),
	}
}

func (n *FloodingNode) AddNeighbor(neighbor *FloodingNode) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.Neighbors[neighbor.ID] = neighbor
}

func (n *FloodingNode) ReceivePacket(packet lib.Packet) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.SeenPackets[packet.ID] {
		fmt.Printf("Node %s: Packet %d already seen, dropping\n", n.ID, packet.ID)
		return
	}

	n.SeenPackets[packet.ID] = true

	if packet.TTL <= 0 {
		fmt.Printf("Node %s: Packet %d TTL expired, dropping\n", n.ID, packet.ID)
		return
	}

	if packet.To == n.ID {
		fmt.Printf("Node %s: Received packet %d with payload: %v\n", n.ID, packet.ID, packet.Payload)
		return
	}

	fmt.Printf("Node %s: Forwarding packet %d (TTL: %d)\n", n.ID, packet.ID, packet.TTL-1)

	forwardedPacket := lib.Packet{
		ID:      packet.ID,
		To:      packet.To,
		From:    n.ID,
		TTL:     packet.TTL - 1,
		Payload: packet.Payload,
	}

	for neighborID, neighbor := range n.Neighbors {
		if neighborID != packet.From {
			go neighbor.ReceivePacket(forwardedPacket)
		}
	}
}

func main() {
	// Create mock nodes
	nodeA := NewNode("A")
	nodeB := NewNode("B")
	nodeC := NewNode("C")

	// Create simple topology: A -- B -- C
	nodeA.AddNeighbor(nodeB)
	nodeB.AddNeighbor(nodeA)
	nodeB.AddNeighbor(nodeC)
	nodeC.AddNeighbor(nodeB)

	fmt.Println("Simple topology: A -- B -- C")
	fmt.Println()

	// Test packet: A sends to C
	testPacket := lib.Packet{
		ID:      1,
		To:      "C",
		From:    "A",
		TTL:     3,
		Payload: "Hello World!",
	}

	fmt.Println("Sending packet from A to C...")
	nodeA.ReceivePacket(testPacket)

	// Wait to see all the flooding activity
	time.Sleep(100 * time.Millisecond)
	fmt.Println("\nFlooding complete!")
}
