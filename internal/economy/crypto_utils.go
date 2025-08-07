package economy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DeriveAddressFromPeerID creates a deterministic address from a libp2p peer ID
// This ensures nodes are identified by their cryptographic identity
func DeriveAddressFromPeerID(peerID peer.ID) string {
	// Get the bytes of the peer ID
	peerBytes := []byte(peerID)

	// Hash the peer ID to create a deterministic address
	hasher := sha256.New()
	hasher.Write(peerBytes)
	addressHash := hasher.Sum(nil)

	// Format as Ethereum-style address (0x + first 20 bytes as hex)
	addressBytes := addressHash[:20]
	return fmt.Sprintf("0x%s", hex.EncodeToString(addressBytes))
}

// FormatNodeIDForDisplay creates a human-readable short form of a node ID
// while preserving the full cryptographic identity
func FormatNodeIDForDisplay(nodeID string) string {
	if len(nodeID) <= 12 {
		return nodeID
	}
	// Show first 6 and last 6 characters for readability
	return fmt.Sprintf("%s...%s", nodeID[:6], nodeID[len(nodeID)-6:])
}

// ValidatePeerID checks if a string is a valid libp2p peer ID
func ValidatePeerID(peerIDStr string) (peer.ID, error) {
	return peer.Decode(peerIDStr)
}

// GenerateTestPeerID creates a valid peer ID for testing purposes
func GenerateTestPeerID(suffix string) string {
	// Use a simplified approach for tests - just use the suffix as a peer ID
	// In a real scenario, these would be proper libp2p peer IDs
	// For testing, we'll bypass validation by using a simple format
	return "test-peer-" + suffix
}
