package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"event-pool/proto"
	"event-pool/sdk"
)

func main() {
	// Parse command line flags
	gatewayURL := flag.String("gateway", "localhost:50051", "Gateway URL")
	chainID := flag.String("chain", "2484", "Chain ID")
	contractAddress := flag.String("contract", "0xeddd02437aa5db6def90ff32c329decd2bcb86db", "Contract address")
	eventSignature := flag.String("event", "TransferSingle", "Event signature")
	flag.Parse()

	// Connect to the gateway
	conn, err := sdk.Connect(*gatewayURL, *chainID)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Subscribe to events
	err = conn.SubscribeEvent(2484, *contractAddress, *eventSignature, func(event *proto.Event) {
		log.Printf("Received event: Block=%d, TxHash=%s, Data=%s",
			event.BlockNumber,
			event.TxHash,
			event.Data,
		)
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to events: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")
}
