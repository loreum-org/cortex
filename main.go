package main

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/loreum-org/cortex/crypto"
	"github.com/loreum-org/cortex/network"
)

// DAG-based aBFT Simulation with Multi-Hop Gossiping and Finality
func main() {
	// Generate keys for each node
	pubA, privA := crypto.GenerateKeys()
	pubB, privB := crypto.GenerateKeys()
	pubC, privC := crypto.GenerateKeys()

	// Create libp2p nodes
	hostA := network.SetupLibp2pNode()
	hostB := network.SetupLibp2pNode()
	hostC := network.SetupLibp2pNode()

	// Create DAG nodes
	nodeA := &network.Node{ID: "A", Transactions: make(map[string]network.Transaction), PrivateKey: privA, PublicKey: pubA, Reputation: 100, Host: hostA}
	nodeB := &network.Node{ID: "B", Transactions: make(map[string]network.Transaction), PrivateKey: privB, PublicKey: pubB, Reputation: 100, Host: hostB}
	nodeC := &network.Node{ID: "C", Transactions: make(map[string]network.Transaction), PrivateKey: privC, PublicKey: pubC, Reputation: 100, Host: hostC}
	nodes := []*network.Node{nodeA, nodeB, nodeC}

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
