package types

// DAGNode represents a node in the DAG
type DAGNode struct {
	Transaction *Transaction `json:"transaction"`
	Children    []*DAGNode   `json:"children"`
	Score       float64      `json:"score"` // Reputation-weighted score
}

// DAG represents the Directed Acyclic Graph
type DAG struct {
	Nodes map[string]*DAGNode `json:"nodes"` // Map of transaction ID to DAGNode
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		Nodes: make(map[string]*DAGNode),
	}
}

// AddNode adds a node to the DAG
func (d *DAG) AddNode(txn *Transaction) *DAGNode {
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

// GetNode returns a node from the DAG
func (d *DAG) GetNode(id string) (*DAGNode, bool) {
	node, exists := d.Nodes[id]
	return node, exists
}

// IsFinalized returns whether a transaction is finalized
func (d *DAG) IsFinalized(id string) bool {
	node, exists := d.Nodes[id]
	return exists && node.Transaction.Finalized
}

// Lock locks the DAG for writing
func (d *DAG) Lock() {
	// No-op for now, as DAG struct itself doesn't have a dedicated lock.
	// Locking is typically handled by the service (e.g., ConsensusService) that manages the DAG instance.
	// In a real implementation where DAG might be accessed by multiple goroutines directly
	// without an external synchronizing service, this would acquire a lock (e.g., d.mu.Lock()).
}

// Unlock unlocks the DAG
func (d *DAG) Unlock() {
	// No-op for now, corresponding to the Lock method.
	// In a real implementation, this would release the lock (e.g., d.mu.Unlock()).
}
