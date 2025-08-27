package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

const NODE_ID = "nodo6"

var FULL_NODE_ID = formatRedisChannel(NODE_ID)

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
	var nodeType string
	flag.StringVar(&nodeType, "t", MESSAGE_PROTOS.FLOOD, "Node Type (flood or lsr)")
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
	senderChan := make(chan ProtocolMsg[any], 10)
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

	node := NewNode(formatRedisChannel("nodo6"), map[string]int{
		formatRedisChannel("nodo1"): 3,
		formatRedisChannel("nodo2"): 7,
		formatRedisChannel("nodo4"): 10,
		// "sec20.topologia2.nodo4": 10,
	})
	for msg := range receiverChan {
		switch msg.Proto {
		case MESSAGE_PROTOS.LSR:
			manageLSRMsg(ctx, msg, senderChan, node)
		case MESSAGE_PROTOS.FLOOD:
			manageFloodMsg(ctx, msg, senderChan)
		default:
			log.Println("ERROR: Invalid message proto received!", msg.Proto)
		}
	}

	cancelCtx()
	wg.Wait()
}

func manageLSRMsg(ctx context.Context, msg ProtocolMsg[any], senderChan chan<- ProtocolMsg[any], node *Node) {
	// Nelson implement LSR here!
	switch payload := msg.Payload.(type) {
	case string:
		log.Println("Received string payload:", payload)
		// HELLO
		switch msg.Type {
		case "hello":
			// Add new node to DB
			example := ProtocolMsg[string]{
				Proto:   MESSAGE_PROTOS.LSR,
				Type:    "hello",
				From:    FULL_NODE_ID,
				To:      "broadcast",
				Ttl:     5,
				Headers: []string{},
				Payload: "",
			}

			// Por favor siempre escribir al channel de esta forma:
			select {
			case senderChan <- example.ToAny():
			case <-ctx.Done():
				return
			}
		case "message":
			// If we're not the destination,
			// Use DIJKSTRA table to send msg
		default:
			log.Printf("ERROR: Invalid message type received: %s\n%#v", msg.Type, msg)
		}
	case map[string]int:
		log.Println("Received info payload:", payload)
		// INFO
		// Regenerate DIJKSTRA table
	default:
		log.Panicf("Invalid message received: %#v", msg)
	}
}

func manageFloodMsg(ctx context.Context, msg ProtocolMsg[any], senderChan chan<- ProtocolMsg[any]) {
	// Joaquin implement Flood here!
}

func parseMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, receiverChan chan<- ProtocolMsg[any]) {
	defer wg.Done()
	log.Println("Parsing messages from redis...")
	defer log.Println("Done parsing messages!")

	// Receive messages from redis
	pubsub := rdb.Subscribe(ctx, FULL_NODE_ID)
	defer pubsub.Close()
	redisReceiveChannel := pubsub.Channel()

	for {
		select {
		case redisMsg := <-redisReceiveChannel:
			msg := redisMsg.Payload

			var messageMsg ProtocolMsg[any]
			err := json.Unmarshal([]byte(msg), &messageMsg)
			if err != nil {
				log.Printf("Received message: `%s` -> `%s`(ttl: %d):\n%#v", messageMsg.From, messageMsg.To, messageMsg.Ttl, messageMsg.Payload)
				receiverChan <- messageMsg
			} else {
				log.Printf("Received invalid message! %s\n%s", err, msg)
			}

		case <-ctx.Done():
			return
		}
	}
}

func sendMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, senderChan <-chan ProtocolMsg[any]) {
	defer wg.Done()
	log.Println("Sending messages to redis...")
	defer log.Println("Done sending messages!")

	for {
		select {
		case receivedMsg := <-senderChan:
			jsonString, err := json.Marshal(receivedMsg)
			if err != nil {
				log.Panicf("Failed to marshal json: %#v", receivedMsg)
			}
			rdb.Publish(ctx, receivedMsg.To, jsonString)
		case <-ctx.Done():
			return
		}
	}
}
