package p2p

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/loreum-org/cortex/pkg/types"
)

// P2PNode represents a P2P network node
type P2PNode struct {
	Host          host.Host
	DHT           *dht.IpfsDHT
	PubSub        *pubsub.PubSub
	Topics        map[string]*pubsub.Topic
	Subscriptions map[string]*pubsub.Subscription
	Config        *types.NetworkConfig
}

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
	// Create a new libp2p Host
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(cfg.ListenAddresses...),
		libp2p.Identity(cfg.PrivateKey),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, err
	}

	// Create a new DHT for peer discovery
	kdht, err := dht.New(context.Background(), h)
	if err != nil {
		return nil, err
	}

	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(context.Background(), h)
	if err != nil {
		return nil, err
	}

	// Create a new PubSub service using the GossipSub router
	node := &P2PNode{
		Host:          h,
		DHT:           kdht,
		PubSub:        ps,
		Topics:        make(map[string]*pubsub.Topic),
		Subscriptions: make(map[string]*pubsub.Subscription),
		Config:        cfg,
	}

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

	return topic.Publish(context.Background(), data)
}

// Stop stops the P2P node
func (n *P2PNode) Stop() error {
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
