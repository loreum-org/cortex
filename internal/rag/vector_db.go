package rag

import (
	"errors"
	"math"
	"sort"
	"sync"

	"github.com/loreum-org/cortex/pkg/types"
)

// VectorStorage represents a simple in-memory vector database
type VectorStorage struct {
	documents map[string]types.VectorDocument
	index     *types.VectorIndex
	mu        sync.RWMutex
}

// NewVectorStorage creates a new vector storage instance
func NewVectorStorage(dimensions int) *VectorStorage {
	return &VectorStorage{
		documents: make(map[string]types.VectorDocument),
		index: &types.VectorIndex{
			Name:       "default",
			Dimensions: dimensions,
			Size:       0,
		},
		mu: sync.RWMutex{},
	}
}

// AddDocument adds a document to the vector storage
func (v *VectorStorage) AddDocument(doc types.VectorDocument) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if the document has the correct dimensions
	if len(doc.Embedding) != v.index.Dimensions {
		return errors.New("document embedding dimensions do not match index dimensions")
	}

	// Check if this is a new document or an update
	_, exists := v.documents[doc.ID]

	// Add the document to the storage
	v.documents[doc.ID] = doc

	// Only increment size if it's a new document
	if !exists {
		v.index.Size++
	}

	return nil
}

// GetDocument retrieves a document from the vector storage
func (v *VectorStorage) GetDocument(id string) (types.VectorDocument, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	doc, exists := v.documents[id]
	if !exists {
		return types.VectorDocument{}, errors.New("document not found")
	}

	return doc, nil
}

// DeleteDocument deletes a document from the vector storage
func (v *VectorStorage) DeleteDocument(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.documents[id]; !exists {
		return errors.New("document not found")
	}

	delete(v.documents, id)
	v.index.Size--

	return nil
}

// SearchSimilar searches for similar documents
func (v *VectorStorage) SearchSimilar(query types.VectorQuery) ([]types.VectorDocument, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if the query has the correct dimensions
	if len(query.Embedding) != v.index.Dimensions {
		return nil, errors.New("query embedding dimensions do not match index dimensions")
	}

	// Calculate similarity for all documents
	type docWithScore struct {
		doc   types.VectorDocument
		score float32
	}

	results := make([]docWithScore, 0, len(v.documents))
	for _, doc := range v.documents {
		score := cosineSimilarity(query.Embedding, doc.Embedding)
		if score >= query.MinSimilarity {
			results = append(results, docWithScore{doc, score})
		}
	}

	// Sort by similarity score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Limit results to max_results
	if query.MaxResults > 0 && len(results) > query.MaxResults {
		results = results[:query.MaxResults]
	}

	// Extract just the documents
	docs := make([]types.VectorDocument, len(results))
	for i, result := range results {
		docs[i] = result.doc
	}

	return docs, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// GetStats returns statistics about the vector storage
func (v *VectorStorage) GetStats() types.VectorIndex {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return *v.index
}
