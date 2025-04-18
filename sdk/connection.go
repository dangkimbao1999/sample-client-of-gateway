package sdk

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"event-pool/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Connection represents a connection to a node server
type Connection struct {
	nodeConn    *grpc.ClientConn
	eventClient proto.EventServiceClient
	ctx         context.Context
	cancel      context.CancelFunc
}

// Connect creates a new connection to a node server via the gateway
func Connect(gatewayURL string, chainID string) (*Connection, error) {
	// Strip "http://" prefix if present
	if strings.HasPrefix(gatewayURL, "http://") {
		gatewayURL = strings.TrimPrefix(gatewayURL, "http://")
	}

	// Connect to the gateway
	log.Printf("Connecting to gateway at %s", gatewayURL)
	gatewayConn, err := grpc.Dial(gatewayURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gateway: %v", err)
	}
	defer gatewayConn.Close()

	// Create gateway client
	gatewayClient := proto.NewGatewayServiceClient(gatewayConn)

	// Request a node for the specified chain
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := gatewayClient.GetNodeForChain(ctx, &proto.GetNodeRequest{
		ChainId: chainID,
	})
	cancel()

	if err != nil {
		return nil, fmt.Errorf("failed to get node from gateway: %v", err)
	}

	if resp.ErrorMessage != "" {
		return nil, fmt.Errorf("gateway error: %s", resp.ErrorMessage)
	}

	// Strip "http://" prefix if present
	nodeAddress := resp.NodeAddress
	if strings.HasPrefix(nodeAddress, "http://") {
		nodeAddress = strings.TrimPrefix(nodeAddress, "http://")
	}
	log.Printf("Gateway returned node address: %s", nodeAddress)

	// Connect to the node
	nodeConn, err := grpc.Dial(nodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %v", err)
	}

	// Create event service client
	eventClient := proto.NewEventServiceClient(nodeConn)

	// Create a context with cancellation
	ctx, cancel = context.WithCancel(context.Background())

	return &Connection{
		nodeConn:    nodeConn,
		eventClient: eventClient,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Close closes the connection
func (c *Connection) Close() {
	c.cancel()
	c.nodeConn.Close()
}

// SubscribeEvent subscribes to events for the specified chain, contract, and event signature
func (c *Connection) SubscribeEvent(chainID int32, contractAddress string, eventSignature string, eventHandler func(*proto.Event)) error {
	// Create the request
	req := &proto.StreamEventsRequest{
		ChainId:         chainID,
		ContractAddress: contractAddress,
		EventSignature:  eventSignature,
	}

	// Subscribe to events
	stream, err := c.eventClient.StreamEvents(c.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %v", err)
	}

	log.Printf("Started streaming events for chain: %d, contract: %s, event: %s",
		chainID, contractAddress, eventSignature)

	// Receive events in a goroutine
	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Printf("Stream ended: %v", err)
				return
			}

			// Call the event handler
			eventHandler(event)
		}
	}()

	return nil
}

func (c *Connection) GetEvents(chainID int32, contractAddress string, txHash string, skip int32, take int32) ([]*proto.EventData, error) {
	req := &proto.GetEventsRequest{
		ChainId:         chainID,
		ContractAddress: contractAddress,
		TxHash:          txHash,
		Take:            take,
		Skip:            skip,
	}

	resp, err := c.eventClient.GetEvents(c.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %v", err)
	}

	return resp.Data, nil
}
