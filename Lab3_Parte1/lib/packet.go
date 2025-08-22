package lib

type Packet struct {
	ID      int
	To      string
	From    string
	TTL     int
	Payload any
}

// Packet used by an LSRNode to tell other nodes their name
type InfoPayload string
