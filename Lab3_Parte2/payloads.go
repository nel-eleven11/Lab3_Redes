package main

type MsgWrapper[T any] struct {
	Destination string
	// This should be a ProtocolMsg with a specific payload type
	ProtocolMsg ProtocolMsg[T]
}

type ProtocolMsg[T any] struct {
	Type    string `json:"type"`
	Payload T      `json:"payload"`
}

// Mensaje cuando el <nodo 1> se presenta ante sus vecinos
// {
//   "type": "init",
//   "payload": {
//     "whoAmI": "<nodo 1>",
//     "neighbours": {
//       "<nodo 1>": 5,
//       "<nodo 2>": 10
//     }
//   }
// }

type InitPayload struct {
	WhoAmI     string         `json:"whoAmI"`
	Neighbours map[string]int `json:"neighbours"`
}

// Mensaje enviado cuando el <nodo 1> necesita enviar un mensaje al <nodo 2>
// {
//   "type": "message",
//   "payload": {
//     "origin": "<nodo 1>",
//     "destination": "<nodo 2>",
//     "ttl": 5,
//     "content": "asdflkjalsdkfjalksjdf"
//   }
// }

type MessagePayload struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Ttl         uint   `json:"ttl"`
	Content     string `json:"content"`
}

// Mensaje enviado cuando ya terminó de calcular sus tablitas
// {
//   "type": "done",
//   "payload": {
//     "whoAmI": "<nodo 1>"
//   }
// }

type DonePayload struct {
	WhoAmI string `json:"whoAmI"`
}
