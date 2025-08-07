package p2p

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/loreum-org/cortex/pkg/types"
)

// Topic names for the Cortex network
const (
	TopicQueries      = "cortex/queries"
	TopicEvents       = "cortex/events"
	TopicResults      = "cortex/results"
	TopicStatus       = "cortex/status"
	TopicEconomic     = "cortex/economic"
	TopicPayments     = "cortex/payments"
	TopicRewards      = "cortex/rewards"
	TopicStakeUpdates = "cortex/stake"
)

// MessageType defines the type of message being sent
type MessageType string

const (
	MessageTypeQuery         MessageType = "query"
	MessageTypeResult        MessageType = "result"
	MessageTypeEvent         MessageType = "event"
	MessageTypeStatus        MessageType = "status"
	MessageTypePayment       MessageType = "payment"
	MessageTypeReward        MessageType = "reward"
	MessageTypeStakeUpdate   MessageType = "stake_update"
	MessageTypeBalanceUpdate MessageType = "balance_update"
	MessageTypeEconomicSync  MessageType = "economic_sync"
)

// NetworkMessage represents a message sent over the P2P network
type NetworkMessage struct {
	Type      MessageType       `json:"type"`
	SenderID  string            `json:"sender_id"`
	Timestamp int64             `json:"timestamp"`
	Payload   map[string]any    `json:"payload"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// QueryBroadcast represents a query broadcast message
type QueryBroadcast struct {
	QueryID   string            `json:"query_id"`
	Text      string            `json:"text"`
	Type      string            `json:"type"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

// PaymentBroadcast represents a payment transaction broadcast
type PaymentBroadcast struct {
	TransactionID string `json:"transaction_id"`
	FromAccount   string `json:"from_account"`
	ToAccount     string `json:"to_account"`
	Amount        string `json:"amount"`
	Fee           string `json:"fee"`
	ModelTier     string `json:"model_tier,omitempty"`
	QueryType     string `json:"query_type,omitempty"`
	Timestamp     int64  `json:"timestamp"`
}

// RewardBroadcast represents a reward distribution broadcast
type RewardBroadcast struct {
	TransactionID   string  `json:"transaction_id"`
	PaymentTxID     string  `json:"payment_tx_id"`
	NodeID          string  `json:"node_id"`
	Amount          string  `json:"amount"`
	ResponseTime    int64   `json:"response_time_ms"`
	QualityScore    float64 `json:"quality_score"`
	Success         bool    `json:"success"`
	ReputationDelta float64 `json:"reputation_delta"`
	Timestamp       int64   `json:"timestamp"`
}

// StakeUpdateBroadcast represents a stake change broadcast
type StakeUpdateBroadcast struct {
	NodeID        string `json:"node_id"`
	Amount        string `json:"amount"`
	Operation     string `json:"operation"` // "stake" or "unstake"
	TransactionID string `json:"transaction_id"`
	TotalStake    string `json:"total_stake"`
	Timestamp     int64  `json:"timestamp"`
}

// BalanceUpdateBroadcast represents a balance change broadcast
type BalanceUpdateBroadcast struct {
	AccountID     string `json:"account_id"`
	NewBalance    string `json:"new_balance"`
	ChangeAmount  string `json:"change_amount"`
	ChangeType    string `json:"change_type"` // "credit" or "debit"
	TransactionID string `json:"transaction_id"`
	Timestamp     int64  `json:"timestamp"`
}

// EconomicSyncMessage represents economic state synchronization
type EconomicSyncMessage struct {
	SenderID       string                 `json:"sender_id"`
	EconomicHeight int64                  `json:"economic_height"`
	StateHash      string                 `json:"state_hash"`
	NodeBalances   map[string]string      `json:"node_balances,omitempty"`
	NodeStakes     map[string]string      `json:"node_stakes,omitempty"`
	NodeMetrics    map[string]interface{} `json:"node_metrics,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
}

// NetworkStats tracks statistics about the P2P network
type NetworkStats struct {
	ConnectedPeers      int       `json:"connected_peers"`
	TotalBytesReceived  int64     `json:"total_bytes_received"`
	TotalBytesSent      int64     `json:"total_bytes_sent"`
	ConnectionsOpened   int64     `json:"connections_opened"`
	ConnectionsClosed   int64     `json:"connections_closed"`
	LastMessageReceived time.Time `json:"last_message_received"`
	LastMessageSent     time.Time `json:"last_message_sent"`
	StartTime           time.Time `json:"start_time"`
}

// P2PNode represents a P2P network node
type P2PNode struct {
	Host          host.Host
	DHT           *dht.IpfsDHT
	PubSub        *pubsub.PubSub
	Topics        map[string]*pubsub.Topic
	Subscriptions map[string]*pubsub.Subscription
	Config        *types.NetworkConfig

	// Message handling
	MessageHandlers map[MessageType][]MessageHandler

	// Network statistics
	stats      NetworkStats
	statsMutex sync.RWMutex

	// Context for the node
	ctx    context.Context
	cancel context.CancelFunc
}

// MessageHandler is a function that handles a specific type of message
type MessageHandler func(msg *NetworkMessage) error

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Printf("Discovered new peer %s\n", pi.ID.String())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		log.Printf("Error connecting to peer %s: %s\n", pi.ID.String(), err)
	}
}

// NewP2PNode creates a new P2P node with the given configuration
func NewP2PNode(cfg *types.NetworkConfig) (*P2PNode, error) {
	// Create a new context
	ctx, cancel := context.WithCancel(context.Background())

	// Create a new libp2p Host
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(cfg.ListenAddresses...),
		libp2p.Identity(cfg.PrivateKey),
		libp2p.NATPortMap(),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	// Create a new DHT for peer discovery
	kdht, err := dht.New(ctx, h)
	if err != nil {
		cancel()
		return nil, err
	}

	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		return nil, err
	}

	// Create a new PubSub service using the GossipSub router
	node := &P2PNode{
		Host:            h,
		DHT:             kdht,
		PubSub:          ps,
		Topics:          make(map[string]*pubsub.Topic),
		Subscriptions:   make(map[string]*pubsub.Subscription),
		Config:          cfg,
		MessageHandlers: make(map[MessageType][]MessageHandler),
		stats: NetworkStats{
			StartTime: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Set up connection event handlers
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			node.onPeerConnected(c.RemotePeer())
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			node.onPeerDisconnected(c.RemotePeer())
		},
	})

	log.Printf("Host created. We are: %s\n", h.ID().String())
	log.Printf("Listening on: %s\n", h.Addrs())

	return node, nil
}

// Start starts the P2P node
func (n *P2PNode) Start(ctx context.Context) error {
	// Bootstrap the DHT
	if err := n.DHT.Bootstrap(ctx); err != nil {
		return err
	}

	// Connect to bootstrap peers
	for _, addrStr := range n.Config.BootstrapPeers {
		addr, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Printf("Error parsing bootstrap peer address: %s\n", err)
			continue
		}
		err = n.Host.Connect(ctx, *addr)
		if err != nil {
			log.Printf("Error connecting to bootstrap peer: %s\n", err)
		}
	}

	// Setup local mDNS discovery
	if err := setupDiscovery(n.Host); err != nil {
		return err
	}

	// Join standard topics
	standardTopics := []string{
		TopicQueries, TopicEvents, TopicResults, TopicStatus,
		TopicEconomic, TopicPayments, TopicRewards, TopicStakeUpdates,
	}
	for _, topicName := range standardTopics {
		_, err := n.JoinTopic(topicName)
		if err != nil {
			log.Printf("Error joining topic %s: %s\n", topicName, err)
			continue
		}

		// Start handling messages from this topic
		go n.handleTopicMessages(topicName)
	}

	return nil
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host
func setupDiscovery(h host.Host) error {
	// Setup local mDNS discovery
	s := mdns.NewMdnsService(h, "loreum-cortex", &discoveryNotifee{h: h})
	return s.Start()
}

// JoinTopic joins a topic and returns a subscription
func (n *P2PNode) JoinTopic(topicName string) (*pubsub.Subscription, error) {
	topic, err := n.PubSub.Join(topicName)
	if err != nil {
		return nil, err
	}
	n.Topics[topicName] = topic

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}
	n.Subscriptions[topicName] = sub

	return sub, nil
}

// Publish publishes a message to a topic
func (n *P2PNode) Publish(topicName string, data []byte) error {
	topic, exists := n.Topics[topicName]
	if !exists {
		var err error
		topic, err = n.PubSub.Join(topicName)
		if err != nil {
			return err
		}
		n.Topics[topicName] = topic
	}

	err := topic.Publish(n.ctx, data)
	if err == nil {
		n.statsMutex.Lock()
		n.stats.TotalBytesSent += int64(len(data))
		n.stats.LastMessageSent = time.Now()
		n.statsMutex.Unlock()
	}
	return err
}

// BroadcastQuery broadcasts a query to the network
func (n *P2PNode) BroadcastQuery(query *types.Query) error {
	// Create a query broadcast message
	queryBroadcast := QueryBroadcast{
		QueryID:   query.ID,
		Text:      query.Text,
		Type:      query.Type,
		Metadata:  query.Metadata,
		Timestamp: query.Timestamp,
	}

	// Create a network message
	msg := NetworkMessage{
		Type:      MessageTypeQuery,
		SenderID:  n.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"query": queryBroadcast},
	}

	// Marshal the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the message to the queries topic
	return n.Publish(TopicQueries, data)
}

// BroadcastEvent broadcasts an event to the network
func (n *P2PNode) BroadcastEvent(eventType string, data map[string]any) error {
	// Create a network message
	msg := NetworkMessage{
		Type:      MessageTypeEvent,
		SenderID:  n.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"event_type": eventType, "data": data},
	}

	// Marshal the message to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the message to the events topic
	return n.Publish(TopicEvents, jsonData)
}

// BroadcastPayment broadcasts a payment transaction to the network
func (n *P2PNode) BroadcastPayment(payment *PaymentBroadcast) error {
	// Create a network message
	msg := NetworkMessage{
		Type:      MessageTypePayment,
		SenderID:  n.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"payment": payment},
	}

	// Marshal the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the message to the payments topic
	return n.Publish(TopicPayments, data)
}

// BroadcastReward broadcasts a reward distribution to the network
func (n *P2PNode) BroadcastReward(reward *RewardBroadcast) error {
	// Create a network message
	msg := NetworkMessage{
		Type:      MessageTypeReward,
		SenderID:  n.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"reward": reward},
	}

	// Marshal the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the message to the rewards topic
	return n.Publish(TopicRewards, data)
}

// BroadcastStakeUpdate broadcasts a stake update to the network
func (n *P2PNode) BroadcastStakeUpdate(stakeUpdate *StakeUpdateBroadcast) error {
	// Create a network message
	msg := NetworkMessage{
		Type:      MessageTypeStakeUpdate,
		SenderID:  n.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"stake_update": stakeUpdate},
	}

	// Marshal the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the message to the stake updates topic
	return n.Publish(TopicStakeUpdates, data)
}

// BroadcastBalanceUpdate broadcasts a balance update to the network
func (n *P2PNode) BroadcastBalanceUpdate(balanceUpdate *BalanceUpdateBroadcast) error {
	// Create a network message
	msg := NetworkMessage{
		Type:      MessageTypeBalanceUpdate,
		SenderID:  n.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"balance_update": balanceUpdate},
	}

	// Marshal the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the message to the economic topic
	return n.Publish(TopicEconomic, data)
}

// BroadcastEconomicSync broadcasts economic state synchronization to the network
func (n *P2PNode) BroadcastEconomicSync(syncMsg *EconomicSyncMessage) error {
	// Create a network message
	msg := NetworkMessage{
		Type:      MessageTypeEconomicSync,
		SenderID:  n.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Payload:   map[string]any{"economic_sync": syncMsg},
	}

	// Marshal the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the message to the economic topic
	return n.Publish(TopicEconomic, data)
}

// RegisterMessageHandler registers a handler for a specific message type
func (n *P2PNode) RegisterMessageHandler(msgType MessageType, handler MessageHandler) {
	n.MessageHandlers[msgType] = append(n.MessageHandlers[msgType], handler)
}

// handleTopicMessages handles messages from a specific topic
func (n *P2PNode) handleTopicMessages(topicName string) {
	sub, exists := n.Subscriptions[topicName]
	if !exists {
		log.Printf("No subscription for topic %s\n", topicName)
		return
	}

	for {
		msg, err := sub.Next(n.ctx)
		if err != nil {
			// Context canceled or subscription closed
			if err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			log.Printf("Error receiving message from topic %s: %s\n", topicName, err)
			continue
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == n.Host.ID() {
			continue
		}

		// Update statistics
		n.statsMutex.Lock()
		n.stats.TotalBytesReceived += int64(len(msg.Data))
		n.stats.LastMessageReceived = time.Now()
		n.statsMutex.Unlock()

		// Process the message
		go n.HandleNetworkMessage(msg.Data)
	}
}

// HandleNetworkMessage processes an incoming network message
func (n *P2PNode) HandleNetworkMessage(data []byte) error {
	// Unmarshal the message
	var msg NetworkMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	// Call the appropriate handlers
	handlers, exists := n.MessageHandlers[msg.Type]
	if !exists {
		// No handlers for this message type
		return nil
	}

	// Call all registered handlers
	for _, handler := range handlers {
		if err := handler(&msg); err != nil {
			log.Printf("Error handling message of type %s: %s\n", msg.Type, err)
		}
	}

	return nil
}

// onPeerConnected is called when a peer connects
func (n *P2PNode) onPeerConnected(p peer.ID) {
	n.statsMutex.Lock()
	n.stats.ConnectionsOpened++
	n.stats.ConnectedPeers = len(n.Host.Network().Peers())
	n.statsMutex.Unlock()

	log.Printf("Peer connected: %s (total: %d)\n", p.String(), n.stats.ConnectedPeers)

	// Broadcast a status update
	go n.BroadcastEvent("peer_connected", map[string]any{
		"peer_id":     p.String(),
		"peers_count": n.stats.ConnectedPeers,
	})
}

// onPeerDisconnected is called when a peer disconnects
func (n *P2PNode) onPeerDisconnected(p peer.ID) {
	n.statsMutex.Lock()
	n.stats.ConnectionsClosed++
	n.stats.ConnectedPeers = len(n.Host.Network().Peers())
	n.statsMutex.Unlock()

	log.Printf("Peer disconnected: %s (total: %d)\n", p.String(), n.stats.ConnectedPeers)

	// Broadcast a status update
	go n.BroadcastEvent("peer_disconnected", map[string]any{
		"peer_id":     p.String(),
		"peers_count": n.stats.ConnectedPeers,
	})
}

// GetNetworkStats returns current network statistics
func (n *P2PNode) GetNetworkStats() NetworkStats {
	n.statsMutex.RLock()
	defer n.statsMutex.RUnlock()

	// Create a copy of the stats
	stats := n.stats

	// Update the connected peers count (this might have changed)
	stats.ConnectedPeers = len(n.Host.Network().Peers())

	return stats
}

// Stop stops the P2P node
func (n *P2PNode) Stop() error {
	// Cancel the context
	n.cancel()

	// Close all subscriptions
	for _, sub := range n.Subscriptions {
		sub.Cancel()
	}

	// Close the DHT
	if err := n.DHT.Close(); err != nil {
		return err
	}

	// Close the Host
	return n.Host.Close()
}

// WaitForPeers waits until we have at least minPeers connected
func (n *P2PNode) WaitForPeers(ctx context.Context, minPeers int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if len(n.Host.Network().Peers()) >= minPeers {
				return nil
			}
			log.Printf("Waiting for peers... currently connected to %d\n", len(n.Host.Network().Peers()))
			time.Sleep(time.Second)
		}
	}
}

// Broadcast sends data to a specific topic (generic interface)
func (n *P2PNode) Broadcast(topic string, data []byte) error {
	return n.Publish(topic, data)
}

// Subscribe subscribes to a topic with a handler (generic interface)
func (n *P2PNode) Subscribe(topic string, handler func([]byte)) error {
	subscription, err := n.JoinTopic(topic)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := subscription.Next(context.Background())
			if err != nil {
				log.Printf("Error reading from subscription: %v", err)
				break
			}
			handler(msg.Data)
		}
	}()

	return nil
}

// GetConnectedPeers returns list of connected peer IDs
func (n *P2PNode) GetConnectedPeers() []string {
	peers := n.Host.Network().Peers()
	peerStrs := make([]string, len(peers))
	for i, peer := range peers {
		peerStrs[i] = peer.String()
	}
	return peerStrs
}
