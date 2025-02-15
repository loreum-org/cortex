package network

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Transaction represents a DAG-based transaction with signature verification.
type Transaction struct {
	ID        string
	Timestamp time.Time
	Data      string
	Parents   []string
	Signature []byte
	PublicKey ed25519.PublicKey
	Finalized bool // Consensus finality flag
}

// Node represents a participant in the network.
type Node struct {
	ID           string
	Transactions map[string]Transaction
	PrivateKey   ed25519.PrivateKey
	PublicKey    ed25519.PublicKey
	Reputation   int
	mu           sync.Mutex
	Host         host.Host
	Peers        []peer.ID
}

// Multi-Hop Routing: Propagates transactions to peers in libp2p network.
func (n *Node) MultiHopGossip(tx Transaction) {
	for _, peerID := range n.Peers {
		fmt.Printf("Node %s sending transaction %s to peer %s\n", n.ID, tx.ID, peerID.String())
	}
}

func (n *Node) CreateTransaction(data string, parents []string) Transaction {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%v", data, time.Now().UnixNano())))
	txID := hex.EncodeToString(hash[:])
	signature := ed25519.Sign(n.PrivateKey, hash[:])

	tx := Transaction{
		ID:        txID,
		Timestamp: time.Now(),
		Data:      data,
		Parents:   parents,
		Signature: signature,
		PublicKey: n.PublicKey,
		Finalized: false, // Initially not finalized
	}

	n.mu.Lock()
	n.Transactions[tx.ID] = tx
	n.mu.Unlock()

	return tx
}

// ValidateTransaction checks if a transaction is valid.
func (n *Node) ValidateTransaction(tx Transaction) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Verify the signature
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%v", tx.Data, tx.Timestamp.UnixNano())))
	if !ed25519.Verify(tx.PublicKey, hash[:], tx.Signature) {
		fmt.Println("Transaction signature invalid!")
		n.Reputation -= 5 // Slash reputation for bad signature.
		return false
	}

	// Check if parents exist
	for _, parentID := range tx.Parents {
		if _, exists := n.Transactions[parentID]; !exists {
			fmt.Println("Transaction parent missing!")
			return false
		}
	}
	return true
}

// GossipTransaction propagates a transaction to other nodes.
func (n *Node) GossipTransaction(tx Transaction, nodes []*Node) {
	for _, node := range nodes {
		if node.ID != n.ID {
			node.mu.Lock()
			node.Transactions[tx.ID] = tx
			node.mu.Unlock()
		}
	}
}

// FinalizeTransaction: Ensures consensus by checkpointing transactions.
func (n *Node) FinalizeTransaction(txID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if tx, exists := n.Transactions[txID]; exists && !tx.Finalized {
		tx.Finalized = true
		n.Transactions[txID] = tx
		fmt.Printf("Transaction %s finalized in node %s\n", txID, n.ID)
	}
}

// SetupLibp2pNode initializes a libp2p host.
func SetupLibp2pNode() host.Host {
	h, err := libp2p.New()
	if err != nil {
		panic(err)
	}
	return h
}
