package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"log"
	"maps"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

const NODE_ID = "nodo6"

var FULL_NODE_ID = formatRedisChannel(NODE_ID)
var NODE_TYPE string

var NODE = NewNode(formatRedisChannel(NODE_ID), map[string]int{
	// formatRedisChannel("nodo1"): 3,
	"sec20.topologia2.nodo10": 5,
	// "sec20.topologia2.nodo6.nodoc": 3,
})

var MESSAGE_PROTOS = struct {
	FLOOD string
	LSR   string
}{
	FLOOD: "flood",
	LSR:   "lsr",
}

func formatRedisChannel(nodeId string) string {
	// FIXME: uncomment line below
	// return "sec20.topologia2." + nodeId
	return "sec20.topologia2." + nodeId + ".prueba1"
}

func main() {
	flag.StringVar(&NODE_TYPE, "t", MESSAGE_PROTOS.FLOOD, "Node Type (flood or lsr)")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	redisAddr, found := os.LookupEnv("REDIS_ADDR")
	if !found {
		log.Fatal("Error `REDIS_ADDR` not found!")
	}

	redisPwd, found := os.LookupEnv("REDIS_PASSWORD")
	if !found {
		log.Fatal("Error `REDIS_PASSWORD` not found!")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPwd,
	})

	// This channel only receives `MsgWrapper` types!
	senderChan := make(chan MsgWrapper[any], 10)
	defer close(senderChan)

	// This channel only receives payloads defined on `payloads.go`
	receiverChan := make(chan ProtocolMsg[any], 10)
	defer close(receiverChan)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go parseMsgs(ctx, &wg, rdb, receiverChan)
	wg.Add(1)
	go sendMsgs(ctx, &wg, rdb, senderChan)
	wg.Add(1)
	go readStdin(ctx, &wg, senderChan, receiverChan)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	log.Println("Connected as:", FULL_NODE_ID, "with type:", NODE_TYPE)
	senderChan <- ProtocolMsg[string]{
		Proto:   NODE_TYPE,
		Type:    "hello",
		From:    FULL_NODE_ID,
		To:      "broadcast",
		Ttl:     5,
		Headers: []string{},
		Payload: "",
	}.ToAny().Wrapped("broadcast")

	for id := range NODE.neighbors {
		senderChan <- ProtocolMsg[map[string]int]{
			Proto:   MESSAGE_PROTOS.LSR,
			Type:    "info",
			From:    FULL_NODE_ID,
			To:      id,
			Ttl:     5,
			Headers: []string{},
			Payload: NODE.neighbors,
		}.ToAny().Wrapped(id)
	}
consumerLoop:
	for {
		select {
		case msg := <-receiverChan:
			switch msg.Proto {
			case MESSAGE_PROTOS.LSR:
				manageLSRMsg(ctx, msg, senderChan, NODE)
			case MESSAGE_PROTOS.FLOOD:
				manageFloodMsg(ctx, msg, senderChan, NODE)
			default:
				log.Println("ERROR: Invalid message proto received!", msg.Proto)
			}
		case <-c:
			log.Println("Interrupt signal received! Stopping node...")
			// senderChan <- ProtocolMsg[string]{
			// 	Proto:   MESSAGE_PROTOS.LSR,
			// 	Type:    "message",
			// 	From:    FULL_NODE_ID,
			// 	To:      "sec20.topologia2.nodo6.nodoa",
			// 	Ttl:     5,
			// 	Headers: []string{},
			// 	Payload: "Hola",
			// }.ToAny().Wrapped("sec20.topologia2.nodo6.nodob")
			// time.Sleep(2 * time.Second)
			break consumerLoop
		}
	}

	cancelCtx()
	wg.Wait()
}

func manageLSRMsg(ctx context.Context, msg ProtocolMsg[any], senderChan chan<- MsgWrapper[any], node *Node) {
	// Ensure our own LSA exists in the DB
	ensureLocalLSA(node)

	// TTL check
	if msg.Ttl <= 0 {
		log.Println("LSR: TTL expired, dropping")
		return
	}

	switch payload := msg.Payload.(type) {

	case string:
		// HELLO or MESSAGE (string payload)
		switch msg.Type {
		case "hello":
			// Do not retransmit. Optional: announce my state (INFO) to neighbors to bootstrap.
			info := make(map[string]int, len(node.neighbors))
			maps.Copy(info, node.neighbors)
			forward := ProtocolMsg[any]{
				Proto:   MESSAGE_PROTOS.LSR,
				Type:    "info",
				From:    FULL_NODE_ID,
				To:      "broadcast",
				Ttl:     5,
				Headers: rotateHeaders(nil, FULL_NODE_ID),
				Payload: info,
			}.Wrapped("broadcast")
			select {
			case senderChan <- forward:
			case <-ctx.Done():
				return
			}

		case "message":
			// If I'm the destination, consume it
			if msg.To == FULL_NODE_ID {
				log.Printf("LSR: Message for me from %s: %s\n", msg.From, payload)
				return
			}
			// Otherwise, route using Dijkstra
			graph := buildGraphFromLSDB()
			nextHop := computeNextHop(graph, FULL_NODE_ID, msg.To)
			if nextHop == "" {
				log.Printf("LSR: No route to %s, dropping. Graph nodes: %d\n", msg.To, len(graph))
				return
			}
			forward := ProtocolMsg[any]{
				Proto:   msg.Proto,
				Type:    msg.Type,
				From:    FULL_NODE_ID,
				To:      msg.To,
				Ttl:     msg.Ttl - 1,
				Headers: rotateHeaders(msg.Headers, FULL_NODE_ID),
				Payload: payload,
			}.Wrapped(nextHop)
			log.Printf("LSR: Forwarding message to %s via %s (ttl %d)\n", msg.To, nextHop, forward.InnerMsg.Ttl)
			select {
			case senderChan <- forward:
			case <-ctx.Done():
				return
			}

		default:
			log.Printf("ERROR: Invalid LSR string-type message `%s`\n", msg.Type)
		}

	case map[string]any:
		// INFO: neighbors->cost announced by router msg.From
		if msg.Type != "info" {
			log.Printf("ERROR: Unexpected map payload for type `%s`\n", msg.Type)
			return
		}

		// Loop avoidance: if I'm already present in headers, drop
		if containsHeader(msg.Headers, FULL_NODE_ID) {
			log.Printf("LSR: Drop INFO from %s (loop detected in headers: %v)\n", msg.From, msg.Headers)
			return
		}

		// Update LSDB
		payload_int := make(map[string]int)
		for key, v := range payload {
			payload_int[key] = int(v.(float64))
		}
		changed := updateLSA(msg.From, payload_int)
		if changed {
			log.Printf("LSR: Updated LSA from %s", msg.From)
		}

		// Retransmit to all my neighbors (except the one who sent it), decreasing TTL and rotating headers
		if msg.Ttl-1 > 0 {
			newHeaders := rotateHeaders(msg.Headers, FULL_NODE_ID)
			for neighbor := range node.neighbors {
				if neighbor == msg.From {
					continue
				}
				forward := ProtocolMsg[any]{
					Proto:   msg.Proto,
					Type:    msg.Type,
					From:    FULL_NODE_ID,
					To:      msg.To,
					Ttl:     msg.Ttl - 1,
					Headers: newHeaders,
					Payload: payload,
				}.Wrapped(neighbor)
				select {
				case senderChan <- forward:
				case <-ctx.Done():
					return
				}
			}
		}

	default:
		log.Panicf("Invalid LSR message payload: %#v", msg)
	}
}

func manageFloodMsg(ctx context.Context, msg ProtocolMsg[any], senderChan chan<- MsgWrapper[any], nodes *Node) {
	log.Printf("Received flood message from %s with TTL %d", msg.From, msg.Ttl)

	// Check if TTL is 0, if so, discard the message
	if msg.Ttl <= 0 {
		log.Println("TTL expired, discarding message")
		return
	}

	// Reduce TTL by 1
	msg.Ttl--

	// Forward the message to all neighbors except the sender
	for neighbor := range nodes.neighbors {
		if neighbor != msg.From {
			// Create a copy of the message for each neighbor
			forwardMsg := ProtocolMsg[any]{
				Proto:   msg.Proto,
				Type:    msg.Type,
				From:    FULL_NODE_ID, // Update sender to current node
				To:      neighbor,
				Ttl:     msg.Ttl,
				Headers: msg.Headers,
				Payload: msg.Payload,
			}

			log.Printf("Forwarding message to %s with TTL %d", neighbor, forwardMsg.Ttl)

			// Send the message using the pattern from manageLSRMsg
			select {
			case senderChan <- MsgWrapper[any]{
				InnerMsg:      forwardMsg,
				TargetChannel: neighbor,
			}:
			case <-ctx.Done():
				return
			}
		}
	}
}

func parseMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, receiverChan chan<- ProtocolMsg[any]) {
	defer wg.Done()
	log.Println("Parsing messages from redis...")
	defer log.Println("Done parsing messages!")

	// Receive messages from redis
	pubsub := rdb.Subscribe(ctx, FULL_NODE_ID, "broadcast")
	defer pubsub.Close()
	redisReceiveChannel := pubsub.Channel()

	for {
		select {
		case redisMsg := <-redisReceiveChannel:
			msg := redisMsg.Payload

			var messageMsg ProtocolMsg[any]
			err := json.Unmarshal([]byte(msg), &messageMsg)
			if err == nil {
				log.Printf("Received message:%s\n`%s` -> `%s`(ttl: %d):\n%#v", msg, messageMsg.From, messageMsg.To, messageMsg.Ttl, messageMsg.Payload)
				receiverChan <- messageMsg
			} else {
				log.Printf("Received invalid message! %s\n%s", err, msg)
			}

		case <-ctx.Done():
			return
		}
	}
}

func sendMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, senderChan <-chan MsgWrapper[any]) {
	defer wg.Done()
	log.Println("Sending messages to redis...")
	defer log.Println("Done sending messages!")

	for {
		select {
		case receivedMsg := <-senderChan:
			jsonString, err := json.Marshal(receivedMsg.InnerMsg)
			if err != nil {
				log.Panicf("Failed to marshal json: %#v", receivedMsg)
			}
			log.Printf("Sending msg:\n%#v\n%s", receivedMsg, jsonString)
			rdb.Publish(ctx, receivedMsg.TargetChannel, jsonString)
			log.Printf("Done!")
		case <-ctx.Done():
			return
		}
	}
}

func readStdin(ctx context.Context, wg *sync.WaitGroup, sendChan chan<- MsgWrapper[any], receiverChan chan<- ProtocolMsg[any]) {
	defer wg.Done()
	log.Println("Reading messages from stdin...")
	defer log.Println("Done reading from stdin!")

	scanner := bufio.NewScanner(os.Stdin)
	msg := MsgWrapper[any]{}
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		line := scanner.Text()
		lineParts := strings.Split(line, " ")

		switch lineParts[0] {
		case "hello":
			msg.InnerMsg = ProtocolMsg[any]{
				Proto:   NODE_TYPE,
				Type:    "hello",
				From:    FULL_NODE_ID,
				To:      "broadcast",
				Ttl:     5,
				Headers: rotateHeaders(nil, FULL_NODE_ID),
				Payload: "",
			}
			msg.TargetChannel = "broadcast"
		case "info":
			msg.TargetChannel = "broadcast"
			msg.InnerMsg = ProtocolMsg[any]{
				Proto:   NODE_TYPE,
				Type:    "info",
				From:    FULL_NODE_ID,
				To:      "broadcast",
				Ttl:     5,
				Headers: rotateHeaders(nil, FULL_NODE_ID),
				Payload: NODE.neighbors,
			}
		case "send":
			target := lineParts[len(lineParts)-1]
			receiverChan <- ProtocolMsg[any]{
				Proto:   NODE_TYPE,
				Type:    "message",
				From:    FULL_NODE_ID,
				To:      target,
				Ttl:     6,
				Headers: rotateHeaders(nil, FULL_NODE_ID),
				Payload: strings.Join(lineParts[1:len(lineParts)-1], " "),
			}
			continue
		case "sendd":
			msg.TargetChannel = lineParts[len(lineParts)-1]
			msg.InnerMsg = ProtocolMsg[any]{
				Proto:   NODE_TYPE,
				Type:    "message",
				From:    FULL_NODE_ID,
				To:      msg.TargetChannel,
				Ttl:     5,
				Headers: rotateHeaders(nil, FULL_NODE_ID),
				Payload: strings.Join(lineParts[1:len(lineParts)-1], " "),
			}
		default:
			log.Println("Invalid command!")
		}

		select {
		case sendChan <- msg:
		case <-ctx.Done():
			return
		}
	}

}
