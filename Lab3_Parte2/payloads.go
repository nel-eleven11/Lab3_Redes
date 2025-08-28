package main

type MsgWrapper[T any] struct {
	InnerMsg      ProtocolMsg[T]
	TargetChannel string
}

type ProtocolMsg[T any] struct {
	Proto   string   `json:"proto"`
	Type    string   `json:"type"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Ttl     int      `json:"ttl"`
	Headers []string `json:"headers"`
	Payload T        `json:"payload"`
}

type INFOMsg = ProtocolMsg[map[string]int]
type MESSAGEMsg = ProtocolMsg[string]

func (p ProtocolMsg[T]) ToAny() ProtocolMsg[any] {
	return ProtocolMsg[any]{
		Proto:   p.Proto,
		Type:    p.Type,
		From:    p.From,
		To:      p.To,
		Ttl:     p.Ttl,
		Headers: p.Headers,
		Payload: p.Payload,
	}
}

func (p ProtocolMsg[T]) Wrapped(targetChannel string) MsgWrapper[T] {
	return MsgWrapper[T]{
		InnerMsg:      p,
		TargetChannel: targetChannel,
	}
}
