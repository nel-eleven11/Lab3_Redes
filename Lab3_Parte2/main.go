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

func formatRedisChannel(nodeId string) string {
	return "sec20.topologia2." + nodeId
}

func main() {
	var nodeType string
	flag.StringVar(&nodeType, "t", "flood", "Node Type (flood or lsr)")
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

	// This channel only receives `ProtoclMsg` types!
	senderChan := make(chan any, 10)
	defer close(senderChan)
	// This channel only receives payloads defined on `payloads.go`
	receiverChan := make(chan any, 10)
	defer close(receiverChan)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

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
	}()

	wg.Add(1)
	go func() {
		wg.Done()
		neighboursId := []string{}

		for {
			select {
			case receivedMsg := <-senderChan:
				if msg, is := receivedMsg.(ProtocolMsg[InitPayload]); is {
					jsonString, err := json.Marshal(msg)
					if err != nil {
						log.Panicf("Failed to marshal json: %#v", msg)
					}

					for _, nodeId := range neighboursId {
						rdb.Publish(ctx, formatRedisChannel(nodeId), jsonString)
					}
				} else if msg, is := receivedMsg.(ProtocolMsg[MessagePayload]); is {
					msg.Payload.Origin = NODE_ID
					jsonString, err := json.Marshal(msg)
					if err != nil {
						log.Panicf("Failed to marshal json: %#v", msg)
					}

					rdb.Publish(ctx, formatRedisChannel(msg.Payload.Destination), jsonString)
				} else if msg, is := receivedMsg.(ProtocolMsg[DonePayload]); is {
					jsonString, err := json.Marshal(msg)
					if err != nil {
						log.Panicf("Failed to marshal json: %#v", msg)
					}

					for _, nodeId := range neighboursId {
						rdb.Publish(ctx, formatRedisChannel(nodeId), jsonString)
					}
				} else {
					log.Panicf("Received invalid ProtocolMsg!\n%#v", receivedMsg)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
