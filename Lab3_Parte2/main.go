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
	senderChan := make(chan any, 10)
	defer close(senderChan)

	// This channel only receives payloads defined on `payloads.go`
	receiverChan := make(chan any, 10)
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
func floodMain(ctx context.Context, receiverChan chan any, senderChan chan any) {
}

// Implement LSR logic here!
// Read a `{type}Payload` from the `receiverChan` if you want to receive a message!
// Write a `MsgWrapper` to the `senderChan` if you want to send a message!
func lsrMain(ctx context.Context, receiverChan chan any, senderChan chan any) {
}

func parseMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, receiverChan chan any) {
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
			var initMsg ProtocolMsg[InitPayload]
			var messageMsg ProtocolMsg[MessagePayload]
			var doneMsg ProtocolMsg[DonePayload]

			if err := json.Unmarshal([]byte(msg), &initMsg); err != nil {
				// Handle init msg
				receiverChan <- initMsg.Payload
				continue
			} else if err = json.Unmarshal([]byte(msg), &messageMsg); err != nil {
				// Handle message msg
				receiverChan <- messageMsg.Payload
			} else if err = json.Unmarshal([]byte(msg), &doneMsg); err != nil {
				// Handle done msg
				receiverChan <- doneMsg.Payload
			} else {
				log.Panicf("Message is not on the correct format!\n%s", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func sendMsgs(ctx context.Context, wg *sync.WaitGroup, rdb *redis.Client, senderChan chan any) {
	defer wg.Done()
	log.Println("Sending messages to redis...")
	defer log.Println("Done sending messages!")

	for {
		select {
		case receivedMsg := <-senderChan:
			switch msg := receivedMsg.(type) {
			case MsgWrapper[InitPayload]:
				publishMsg(ctx, rdb, msg)
			case MsgWrapper[MessagePayload]:
				publishMsg(ctx, rdb, msg)
			case MsgWrapper[DonePayload]:
				publishMsg(ctx, rdb, msg)
			default:
				log.Panicf("Received invalid MsgWrapper!\n%#v", receivedMsg)
			}
		case <-ctx.Done():
			return
		}
	}
}

func publishMsg[T any](ctx context.Context, rdb *redis.Client, msg MsgWrapper[T]) {
	jsonString, err := json.Marshal(msg.ProtocolMsg)
	if err != nil {
		log.Panicf("Failed to marshal json: %#v", msg)
	}
	rdb.Publish(ctx, msg.Destination, jsonString)

}
