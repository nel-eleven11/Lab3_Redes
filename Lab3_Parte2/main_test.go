package main

import "testing"

func Test_MsgParsing(t *testing.T) {
	// This channel only receives `MsgWrapper` types!
	senderChan := make(chan any, 10)
	defer close(senderChan)

	// This channel only receives payloads defined on `payloads.go`
	receiverChan := make(chan any, 10)
	defer close(receiverChan)

	// `main` function does this!
	receiverChan <- InitPayload{
		WhoAmI: formatRedisChannel("nodo1"),
		Neighbours: map[string]int{
			formatRedisChannel("nodo2"): 5,
			formatRedisChannel("nodo3"): 5,
		},
	}

	// `floodMain` function does this!
	anyPayload := <-receiverChan
	switch payload := anyPayload.(type) {
	case InitPayload:
		t.Log("Success!", payload.WhoAmI)
	default:
		t.Errorf("Invalid payload received! %#v", anyPayload)
	}

	// `floodMain` function does this!
	innerMsg := MsgWrapper[InitPayload]{
		Destination: formatRedisChannel("nodo3"),
		ProtocolMsg: ProtocolMsg[InitPayload]{
			Type: "init",
			Payload: InitPayload{
				WhoAmI: formatRedisChannel("nodo1"),
				Neighbours: map[string]int{
					formatRedisChannel("nodo2"): 5,
					formatRedisChannel("nodo3"): 5,
				},
			},
		},
	}
	senderChan <- innerMsg

	// `main` function does this!
	anyPayload = <-senderChan
	switch msg := anyPayload.(type) {
	case MsgWrapper[InitPayload]:
		t.Log("Success!", msg.Destination)
	default:
		t.Errorf("Invalid payload received! %#v", anyPayload)

	}

}
