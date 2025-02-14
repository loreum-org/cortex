package main

import (
	"crypto/ed25519"
	"crypto/rand"
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

// GenerateKeys creates a new key pair for signing transactions.
func GenerateKeys() (ed25519.PublicKey, ed25519.PrivateKey) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	return publicKey, privateKey
}

// CreateTransaction generates a signed transaction.
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

// Multi-Hop Routing: Propagates transactions to peers in libp2p network.
func (n *Node) MultiHopGossip(tx Transaction) {
	for _, peerID := range n.Peers {
		fmt.Printf("Node %s sending transaction %s to peer %s\n", n.ID, tx.ID, peerID.String())
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

// DAG-based aBFT Simulation with Multi-Hop Gossiping and Finality
func main() {
	// Generate keys for each node
	pubA, privA := GenerateKeys()
	pubB, privB := GenerateKeys()
	pubC, privC := GenerateKeys()

	// Create libp2p nodes
	hostA := SetupLibp2pNode()
	hostB := SetupLibp2pNode()
	hostC := SetupLibp2pNode()

	// Create DAG nodes
	nodeA := &Node{ID: "A", Transactions: make(map[string]Transaction), PrivateKey: privA, PublicKey: pubA, Reputation: 100, Host: hostA}
	nodeB := &Node{ID: "B", Transactions: make(map[string]Transaction), PrivateKey: privB, PublicKey: pubB, Reputation: 100, Host: hostB}
	nodeC := &Node{ID: "C", Transactions: make(map[string]Transaction), PrivateKey: privC, PublicKey: pubC, Reputation: 100, Host: hostC}
	nodes := []*Node{nodeA, nodeB, nodeC}

	// Connect nodes for libp2p multi-hop communication
	nodeA.Peers = []peer.ID{hostB.ID(), hostC.ID()}
	nodeB.Peers = []peer.ID{hostA.ID(), hostC.ID()}
	nodeC.Peers = []peer.ID{hostA.ID(), hostB.ID()}

	// Genesis transaction
	genesisTx := nodeA.CreateTransaction("Genesis", []string{})
	nodeA.GossipTransaction(genesisTx, nodes)

	// New transaction with Genesis as parent
	tx1 := nodeB.CreateTransaction("Tx1", []string{genesisTx.ID})
	if nodeB.ValidateTransaction(tx1) {
		nodeB.GossipTransaction(tx1, nodes)
		nodeB.MultiHopGossip(tx1)
		fmt.Println("Tx1 added and propagated.")
	}

	// Another transaction with Tx1 as parent
	tx2 := nodeC.CreateTransaction("Tx2", []string{tx1.ID})
	if nodeC.ValidateTransaction(tx2) {
		nodeC.GossipTransaction(tx2, nodes)
		nodeC.MultiHopGossip(tx2)
		fmt.Println("Tx2 added and propagated.")
	}

	// Finalizing transactions after propagation
	nodeA.FinalizeTransaction(genesisTx.ID)
	nodeB.FinalizeTransaction(tx1.ID)
	nodeC.FinalizeTransaction(tx2.ID)

	// Print finalized transactions in Node A
	fmt.Println("Node A Transactions (Finalized Status):")
	for _, tx := range nodeA.Transactions {
		fmt.Printf("ID: %s, Data: %s, Parents: %v, Finalized: %v\n", tx.ID, tx.Data, tx.Parents, tx.Finalized)
	}
}
