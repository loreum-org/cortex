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
	for _, parentID := range txn.Parents {
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
