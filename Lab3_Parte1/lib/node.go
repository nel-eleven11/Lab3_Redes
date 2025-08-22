package lib

type Node interface {
	GetID() string
	Connect(neighbor Node, gps_bandwith float32)
	Receive(packet Packet)
}
