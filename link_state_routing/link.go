package main

type Link struct {
	NodeA string
	NodeB string
	Cost  int
	Up    bool
}

func NewLink(nodeA, nodeB string, cost int) *Link {
	return &Link{
		NodeA: nodeA,
		NodeB: nodeB,
		Cost:  cost,
		Up:    true,
	}
}

func (l *Link) GetOtherEnd(node string) string {
	if l.NodeA == node {
		return l.NodeB
	}
	return l.NodeA
}

func (l *Link) SetStatus(up bool) {
	l.Up = up
}
