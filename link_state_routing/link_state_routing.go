package main

import (
	"sync"

	"github.com/nel-eleven11/Lab3_Redes/lib"
)

type LSRNode struct {
	ID          string
	Neighbors   map[string]*lib.Node
	SeenPackets map[int]struct{}
	mutex       sync.RWMutex
}

func NewLSRNode(id string) *LSRNode {
	return &LSRNode{
		ID:          id,
		Neighbors:   make(map[string]*lib.Node),
		SeenPackets: make(map[int]struct{}),
	}
}

func (l *LSRNode) Connect(neighbor *lib.Node, gps_bandwith float32) {

}

func main() {
	// Network topology:
	// A ---+
	// |  B |
	// | / \|
	// C ---D
	var a *lib.Node = NewLSRNode("A")
	b := NewLSRNode("B")
	c := NewLSRNode("C")
	d := NewLSRNode("D")

	a.Connect()

}
