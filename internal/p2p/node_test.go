package p2p

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/loreum-org/cortex/pkg/types"
	"github.com/multiformats/go-multiaddr"
)

// TestP2PNode_NewP2PNode tests creating a new P2P node
func TestP2PNode_NewP2PNode(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *types.NetworkConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			setup: func() *types.NetworkConfig {
				priv, _, err := crypto.GenerateEd25519Key(nil)
				if err != nil {
					t.Fatalf("Failed to generate key: %v", err)
				}
				return &types.NetworkConfig{
					PrivateKey:      priv,
					ListenAddresses: []string{"/ip4/127.0.0.1/tcp/0"},
					BootstrapPeers:  []string{},
				}
			},
			wantErr: false,
		},
		{
			name: "invalid listen address",
			setup: func() *types.NetworkConfig {
				priv, _, err := crypto.GenerateEd25519Key(nil)
				if err != nil {
					t.Fatalf("Failed to generate key: %v", err)
				}
				return &types.NetworkConfig{
					PrivateKey:      priv,
					ListenAddresses: []string{"invalid-address"},
					BootstrapPeers:  []string{},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setup()
			node, err := NewP2PNode(config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewP2PNode() expected error but got none")
					if node != nil {
						node.Stop()
					}
				}
				return
			}

			if err != nil {
				t.Errorf("NewP2PNode() unexpected error = %v", err)
				return
			}

			if node == nil {
				t.Errorf("NewP2PNode() returned nil node")
				return
			}

			// Verify node initialization
			if node.Host == nil {
				t.Errorf("NewP2PNode() returned node with nil Host")
			}

			if node.DHT == nil {
				t.Errorf("NewP2PNode() returned node with nil DHT")
			}

			if node.PubSub == nil {
				t.Errorf("NewP2PNode() returned node with nil PubSub")
			}

			if node.Config != config {
				t.Errorf("NewP2PNode() config mismatch")
			}

			// Clean up
			node.Stop()
		})
	}
}

// TestP2PNode_StartStop tests starting and stopping a P2P node
func TestP2PNode_StartStop(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test start
	err := node.Start(ctx)
	if err != nil {
		t.Errorf("Start() unexpected error = %v", err)
		return
	}

	// Verify node is running
	if len(node.Host.Addrs()) == 0 {
		t.Errorf("Start() node has no listening addresses")
	}

	// Test stop
	err = node.Stop()
	if err != nil {
		t.Errorf("Stop() unexpected error = %v", err)
	}
}

// TestP2PNode_JoinTopic tests joining and subscribing to topics
func TestP2PNode_JoinTopic(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	tests := []struct {
		name      string
		topicName string
		wantErr   bool
	}{
		{
			name:      "join valid topic",
			topicName: "test-topic",
			wantErr:   false,
		},
		{
			name:      "join another valid topic",
			topicName: "another-test-topic",
			wantErr:   false,
		},
		{
			name:      "rejoin existing topic",
			topicName: "test-topic-new", // Use different topic name
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := node.JoinTopic(tt.topicName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("JoinTopic() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("JoinTopic() unexpected error = %v", err)
				return
			}

			if sub == nil {
				t.Errorf("JoinTopic() returned nil subscription")
				return
			}

			// Verify topic and subscription are stored
			if _, exists := node.Topics[tt.topicName]; !exists {
				t.Errorf("JoinTopic() topic not stored in node.Topics")
			}

			if _, exists := node.Subscriptions[tt.topicName]; !exists {
				t.Errorf("JoinTopic() subscription not stored in node.Subscriptions")
			}
		})
	}
}

// TestP2PNode_Publish tests publishing messages to topics
func TestP2PNode_Publish(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Join a topic first
	topicName := "test-publish-topic"
	_, err = node.JoinTopic(topicName)
	if err != nil {
		t.Fatalf("Failed to join topic: %v", err)
	}

	tests := []struct {
		name      string
		topicName string
		data      []byte
		wantErr   bool
	}{
		{
			name:      "publish to existing topic",
			topicName: topicName,
			data:      []byte("test message"),
			wantErr:   false,
		},
		{
			name:      "publish to new topic",
			topicName: "new-topic",
			data:      []byte("message to new topic"),
			wantErr:   false,
		},
		{
			name:      "publish empty message",
			topicName: topicName,
			data:      []byte{},
			wantErr:   false,
		},
		{
			name:      "publish binary data",
			topicName: topicName,
			data:      []byte{0x00, 0x01, 0x02, 0xFF},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.Publish(tt.topicName, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Publish() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Publish() unexpected error = %v", err)
			}
		})
	}
}

// TestP2PNode_BroadcastQuery tests broadcasting queries to the network
func TestP2PNode_BroadcastQuery(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	query := &types.Query{
		ID:        "test-query-1",
		Text:      "What is the meaning of life?",
		Type:      "question",
		Metadata:  map[string]string{"priority": "high"},
		Timestamp: time.Now().Unix(),
	}

	err = node.BroadcastQuery(query)
	if err != nil {
		t.Errorf("BroadcastQuery() unexpected error = %v", err)
	}

	// Verify query topic exists
	if _, exists := node.Topics[TopicQueries]; !exists {
		t.Errorf("BroadcastQuery() did not create queries topic")
	}
}

// TestP2PNode_BroadcastEvent tests broadcasting events to the network
func TestP2PNode_BroadcastEvent(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	eventData := map[string]any{
		"timestamp": time.Now().Unix(),
		"message":   "test event",
		"level":     "info",
	}

	err = node.BroadcastEvent("test-event", eventData)
	if err != nil {
		t.Errorf("BroadcastEvent() unexpected error = %v", err)
	}

	// Verify events topic exists
	if _, exists := node.Topics[TopicEvents]; !exists {
		t.Errorf("BroadcastEvent() did not create events topic")
	}
}

// TestP2PNode_MessageHandlers tests message handler registration and invocation
func TestP2PNode_MessageHandlers(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	// Test handler registration
	var handledMessages []NetworkMessage
	var mu sync.Mutex

	handler := func(msg *NetworkMessage) error {
		mu.Lock()
		defer mu.Unlock()
		handledMessages = append(handledMessages, *msg)
		return nil
	}

	node.RegisterMessageHandler(MessageTypeQuery, handler)

	// Verify handler is registered
	handlers, exists := node.MessageHandlers[MessageTypeQuery]
	if !exists {
		t.Errorf("RegisterMessageHandler() handler not registered")
		return
	}

	if len(handlers) != 1 {
		t.Errorf("RegisterMessageHandler() expected 1 handler, got %d", len(handlers))
	}

	// Test message handling
	testMessage := NetworkMessage{
		Type:      MessageTypeQuery,
		SenderID:  "test-sender",
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"test": "data"},
	}

	data, err := json.Marshal(testMessage)
	if err != nil {
		t.Fatalf("Failed to marshal test message: %v", err)
	}

	err = node.HandleNetworkMessage(data)
	if err != nil {
		t.Errorf("HandleNetworkMessage() unexpected error = %v", err)
	}

	// Verify handler was called
	mu.Lock()
	if len(handledMessages) != 1 {
		t.Errorf("Handler not called: expected 1 message, got %d", len(handledMessages))
	} else if handledMessages[0].Type != MessageTypeQuery {
		t.Errorf("Handler received wrong message type: %s", handledMessages[0].Type)
	}
	mu.Unlock()
}

// TestP2PNode_NetworkStats tests network statistics tracking
func TestP2PNode_NetworkStats(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Get initial stats
	stats := node.GetNetworkStats()

	if stats.StartTime.IsZero() {
		t.Errorf("GetNetworkStats() StartTime not set")
	}

	if stats.ConnectedPeers < 0 {
		t.Errorf("GetNetworkStats() ConnectedPeers should be non-negative")
	}

	// Publish a message to update stats
	testData := []byte("test message for stats")
	err = node.Publish("test-stats-topic", testData)
	if err != nil {
		t.Errorf("Failed to publish test message: %v", err)
	}

	// Get updated stats
	newStats := node.GetNetworkStats()

	// Verify bytes sent increased
	if newStats.TotalBytesSent <= stats.TotalBytesSent {
		t.Errorf("TotalBytesSent should have increased after publishing")
	}

	// Verify LastMessageSent was updated
	if newStats.LastMessageSent.Before(stats.LastMessageSent) {
		t.Errorf("LastMessageSent should have been updated")
	}
}

// TestP2PNode_TwoNodesIntegration tests communication between two nodes
func TestP2PNode_TwoNodesIntegration(t *testing.T) {
	// Create two nodes
	node1 := createTestNode(t)
	defer node1.Stop()

	node2 := createTestNode(t)
	defer node2.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start both nodes
	err := node1.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node1: %v", err)
	}

	err = node2.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node2: %v", err)
	}

	// Connect node2 to node1
	node1Addr := node1.Host.Addrs()[0]
	node1PeerInfo := node1.Host.ID()

	err = node2.Host.Connect(ctx, peer.AddrInfo{
		ID:    node1PeerInfo,
		Addrs: []multiaddr.Multiaddr{node1Addr},
	})
	if err != nil {
		t.Fatalf("Failed to connect nodes: %v", err)
	}

	// Wait for connection to establish
	time.Sleep(2 * time.Second)

	// Set up message handler on node2
	var receivedMessages []NetworkMessage
	var mu sync.Mutex

	handler := func(msg *NetworkMessage) error {
		mu.Lock()
		defer mu.Unlock()
		receivedMessages = append(receivedMessages, *msg)
		return nil
	}

	node2.RegisterMessageHandler(MessageTypeQuery, handler)

	// Join same topic on both nodes
	topicName := "integration-test-topic"
	_, err = node1.JoinTopic(topicName)
	if err != nil {
		t.Fatalf("Node1 failed to join topic: %v", err)
	}

	_, err = node2.JoinTopic(topicName)
	if err != nil {
		t.Fatalf("Node2 failed to join topic: %v", err)
	}

	// Start message handling on node2
	go node2.handleTopicMessages(topicName)

	// Wait for subscriptions to settle
	time.Sleep(3 * time.Second)

	// Broadcast a query from node1
	query := &types.Query{
		ID:        "integration-test-query",
		Text:      "Integration test query",
		Type:      "test",
		Metadata:  map[string]string{"test": "integration"},
		Timestamp: time.Now().Unix(),
	}

	err = node1.BroadcastQuery(query)
	if err != nil {
		t.Fatalf("Failed to broadcast query: %v", err)
	}

	// Wait for message to be received
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Errorf("Timeout waiting for message to be received")
			return
		case <-ticker.C:
			mu.Lock()
			if len(receivedMessages) > 0 {
				// Verify the received message
				msg := receivedMessages[0]
				if msg.Type != MessageTypeQuery {
					t.Errorf("Received wrong message type: %s", msg.Type)
				}
				if msg.SenderID != node1.Host.ID().String() {
					t.Errorf("Received wrong sender ID: %s", msg.SenderID)
				}
				mu.Unlock()
				return
			}
			mu.Unlock()
		}
	}
}

// TestP2PNode_WaitForPeers tests waiting for peer connections
func TestP2PNode_WaitForPeers(t *testing.T) {
	node := createTestNode(t)
	defer node.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Test waiting for 0 peers (should return immediately)
	waitCtx, waitCancel := context.WithTimeout(ctx, 1*time.Second)
	defer waitCancel()

	err = node.WaitForPeers(waitCtx, 0)
	if err != nil {
		t.Errorf("WaitForPeers(0) unexpected error = %v", err)
	}

	// Test timeout when waiting for peers that won't connect
	waitCtx2, waitCancel2 := context.WithTimeout(ctx, 100*time.Millisecond)
	defer waitCancel2()

	err = node.WaitForPeers(waitCtx2, 5)
	if err == nil {
		t.Errorf("WaitForPeers() expected timeout error but got none")
	}
}

// Helper functions

func createTestNode(t *testing.T) *P2PNode {
	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	config := &types.NetworkConfig{
		PrivateKey:      priv,
		ListenAddresses: []string{"/ip4/127.0.0.1/tcp/0"},
		BootstrapPeers:  []string{},
	}

	node, err := NewP2PNode(config)
	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	return node
}
