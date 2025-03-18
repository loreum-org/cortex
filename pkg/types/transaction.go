package types

import (
	"time"
)

// Transaction represents a transaction in the DAG
type Transaction struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data"` // Query data
	Parents   []string  `json:"parents"`
	Signature []byte    `json:"signature"`
	Finalized bool      `json:"finalized"`
}

// NewTransaction creates a new transaction with the given data and parents
func NewTransaction(data string, parents []string) *Transaction {
	return &Transaction{
		Timestamp: time.Now(),
		Data:      data,
		Parents:   parents,
		Finalized: false,
	}
}
