package main

import "testing"

func Test_MsgParsing(t *testing.T) {
	// This channel only receives `MsgWrapper` types!
	senderChan := make(chan ProtocolMsg[any], 10)
	defer close(senderChan)

	// This channel only receives payloads defined on `payloads.go`
	receiverChan := make(chan ProtocolMsg[any], 10)
	defer close(receiverChan)

	// `main` function does this!
	receiverChan <- ProtocolMsg[string]{
		From:    formatRedisChannel("nodo3"),
		To:      formatRedisChannel("nodo1"),
		Proto:   "lsr",
		Ttl:     5,
		Headers: []string{},
		Payload: "asdlfkjalsdkfj",
	}.ToAny()

	// `floodMain` function does this!
	anyPayload := <-receiverChan
	switch payload := anyPayload.Payload.(type) {
	case string:
		t.Log("Success!", payload)
	default:
		t.Errorf("Invalid payload received! %#v", anyPayload)
	}

	// `floodMain` function does this!
	innerMsg := ProtocolMsg[map[string]int]{
		From:    formatRedisChannel("nodo3"),
		To:      formatRedisChannel("nodo1"),
		Proto:   "lsr",
		Ttl:     5,
		Headers: []string{},
		Payload: map[string]int{
			formatRedisChannel("nodo0"):  10,
			formatRedisChannel("nodo10"): 4,
		},
	}
	senderChan <- innerMsg.ToAny()

	// `main` function does this!
	anyPayload = <-senderChan
	switch msg := anyPayload.Payload.(type) {
	case map[string]int:
		t.Log("Success!", msg)
	default:
		t.Errorf("Invalid payload received! %#v", anyPayload)

	}

}
