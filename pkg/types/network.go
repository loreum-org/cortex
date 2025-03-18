package types

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// NetworkConfig defines the configuration for the P2P network
type NetworkConfig struct {
	ListenAddresses []string       `json:"listen_addresses"`
	BootstrapPeers  []string       `json:"bootstrap_peers"`
	PrivateKey      crypto.PrivKey `json:"-"` // Private key not serialized to JSON
	Protocols       []protocol.ID  `json:"protocols"`
}

// NewNetworkConfig creates a new NetworkConfig with default values
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		ListenAddresses: []string{"/ip4/0.0.0.0/tcp/4001", "/ip4/0.0.0.0/udp/4001/quic-v1"},
		BootstrapPeers:  []string{},
		Protocols:       []protocol.ID{"/loreum/1.0.0"},
	}
}
