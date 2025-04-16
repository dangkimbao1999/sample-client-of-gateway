package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"event-pool/config"
	"event-pool/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	configPath = flag.String("config", "config/config.yaml", "path to config file")
	chainID    = flag.String("chain", "polygon", "blockchain chain ID to connect to")
	useGateway = flag.Bool("gateway", true, "use gateway to get node address")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Get chain configuration
	chainCfg, err := cfg.GetChainConfig(*chainID)
	if err != nil {
		log.Fatalf("Failed to get chain config: %v", err)
	}

	var nodeAddress string

	// Try to use gateway if enabled
	if *useGateway {
		// Connect to the gateway
		gatewayAddr := cfg.GetGatewayAddr()
		log.Printf("Connecting to gateway at %s", gatewayAddr)

		gatewayConn, err := grpc.Dial(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Failed to connect to gateway: %v", err)
			log.Println("Falling back to direct connection")
		} else {
			defer gatewayConn.Close()

			// Create gateway client
			gatewayClient := proto.NewGatewayServiceClient(gatewayConn)

			// Request a node for the specified chain
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := gatewayClient.GetNodeForChain(ctx, &proto.GetNodeRequest{
				ChainId: "2484",
			})
			cancel()

			if err != nil {
				log.Printf("Failed to get node from gateway: %v", err)
				log.Println("Falling back to direct connection")
			} else if resp.ErrorMessage != "" {
				log.Printf("Gateway error: %s", resp.ErrorMessage)
				log.Println("Falling back to direct connection")
			} else {
				nodeAddress = resp.NodeAddress
				if strings.HasPrefix(nodeAddress, "http://") {
					nodeAddress = strings.TrimPrefix(nodeAddress, "http://")
					log.Printf("Stripped http:// prefix from node address")
				}
				log.Printf("Gateway returned node address: %s", nodeAddress)
			}
		}
	}

	// If gateway failed or is disabled, use direct connection
	if nodeAddress == "" {
		nodeAddress = "localhost:9090"
		log.Printf("Using direct connection to %s", nodeAddress)
	}

	// Connect to the node
	nodeConn, err := grpc.Dial(nodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to node: %v", err)
	}
	defer nodeConn.Close()

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Create event service client
	eventClient := proto.NewEventServiceClient(nodeConn)

	// Create the request
	req := &proto.StreamEventsRequest{
		ChainId:         int32(chainCfg.ChainID),
		ContractAddress: chainCfg.ContractAddress,
		EventSignature:  chainCfg.EventSignature,
	}

	// Subscribe to events
	stream, err := eventClient.StreamEvents(ctx, req)
	if err != nil {
		log.Fatalf("Failed to subscribe to events: %v", err)
	}

	log.Printf("Started streaming events for chain: %s", *chainID)

	// Receive events
	for {
		event, err := stream.Recv()
		if err != nil {
			log.Printf("Stream ended: %v", err)
			return
		}

		log.Printf("Received event: Block=%d, TxHash=%s, Data=%s",
			event.BlockNumber,
			event.TxHash,
			event.Data,
		)
	}
}
