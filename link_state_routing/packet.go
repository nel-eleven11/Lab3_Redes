package main

import "time"

type PacketType int

const (
	LSA_PACKET PacketType = iota // Link State Advertisement
	DATA_PACKET
)

type Packet struct {
	Type        PacketType
	Source      string
	Destination string
	Data        any
	TTL         int
	SequenceNum int
}

type LSAData struct {
	OriginRouter string
	Neighbors    map[string]int // neighbor -> cost
	SequenceNum  int
	Timestamp    time.Time
}
