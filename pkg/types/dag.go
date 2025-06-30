package types

import "sync"

// DAGNode represents a node in the DAG
type DAGNode struct {
	Transaction *Transaction `json:"transaction"`
	Children    []*DAGNode   `json:"children"`
	Score       float64      `json:"score"` // Reputation-weighted score
}

// DAG represents the Directed Acyclic Graph
type DAG struct {
	Nodes map[string]*DAGNode `json:"nodes"` // Map of transaction ID to DAGNode
	mu    sync.RWMutex        `json:"-"`     // Mutex for thread-safe operations
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		Nodes: make(map[string]*DAGNode),
	}
}

// AddNode adds a node to the DAG (thread-safe)
func (d *DAG) AddNode(txn *Transaction) *DAGNode {
	d.mu.Lock()
	defer d.mu.Unlock()

	node := &DAGNode{
		Transaction: txn,
		Children:    make([]*DAGNode, 0),
		Score:       0.0,
	}

	d.Nodes[txn.ID] = node

	// Connect the node to its parents
	for _, parentID := range txn.ParentIDs {
		if parent, exists := d.Nodes[parentID]; exists {
			parent.Children = append(parent.Children, node)
		}
	}

	return node
}

// GetNode returns a node from the DAG (thread-safe)
func (d *DAG) GetNode(id string) (*DAGNode, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, exists := d.Nodes[id]
	return node, exists
}

// IsFinalized returns whether a transaction is finalized (thread-safe)
func (d *DAG) IsFinalized(id string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	node, exists := d.Nodes[id]
	return exists && node.Transaction.Finalized
}

// Lock locks the DAG for writing
func (d *DAG) Lock() {
	d.mu.Lock()
}

// Unlock unlocks the DAG
func (d *DAG) Unlock() {
	d.mu.Unlock()
}

// RLock locks the DAG for reading
func (d *DAG) RLock() {
	d.mu.RLock()
}

// RUnlock unlocks the DAG for reading
func (d *DAG) RUnlock() {
	d.mu.RUnlock()
}

// GetAllNodes returns all nodes in the DAG (thread-safe)
func (d *DAG) GetAllNodes() map[string]*DAGNode {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Create a copy to avoid race conditions
	nodes := make(map[string]*DAGNode, len(d.Nodes))
	for id, node := range d.Nodes {
		nodes[id] = node
	}
	return nodes
}
