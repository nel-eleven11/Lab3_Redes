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

var NODE_TYPES = struct {
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
	flag.StringVar(&nodeType, "t", NODE_TYPES.FLOOD, "Node Type (flood or lsr)")
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

	if nodeType == NODE_TYPES.FLOOD {
		log.Println("Starting node as flood type...")
		floodMain(ctx, receiverChan, senderChan)
	} else {
		log.Println("Starting node as LSR type...")
		lsrMain(ctx, receiverChan, senderChan)
	}

	cancelCtx()
	wg.Wait()
}

// Implement Flood logic here!
// Read a `{type}Payload` from the `receiverChan` if you want to receive a message!
// Write a `MsgWrapper` to the `senderChan` if you want to send a message!
func floodMain(ctx context.Context, receiverChan chan ProtocolMsg[any], senderChan chan ProtocolMsg[any]) {
}

// Implement LSR logic here!
// Read a `{type}Payload` from the `receiverChan` if you want to receive a message!
// Write a `MsgWrapper` to the `senderChan` if you want to send a message!
func lsrMain(ctx context.Context, receiverChan chan ProtocolMsg[any], senderChan chan ProtocolMsg[any]) {
}

func parseMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, receiverChan chan ProtocolMsg[any]) {
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

func sendMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, senderChan chan ProtocolMsg[any]) {
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
