package rag

import (
	"context"
	"errors"

	"github.com/loreum-org/cortex/pkg/types"
)

// RAGSystem represents the Retrieval-Augmented Generation system
type RAGSystem struct {
	VectorDB       *VectorStorage
	QueryProcessor *QueryProcessor
	ModelManager   *ModelManager
	ContextBuilder *ContextBuilder
}

// QueryProcessor processes queries for the RAG system
type QueryProcessor struct {
	ModelID string
}

// ModelManager manages models for the RAG system
type ModelManager struct {
	Models map[string]Model
}

// Model represents an AI model for the RAG system
type Model interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateResponse(ctx context.Context, prompt string) (string, error)
}

// MockModel is a mock implementation of the Model interface
type MockModel struct {
	ID         string
	Dimensions int
}

// GenerateEmbedding generates an embedding for the given text
func (m *MockModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// This is a mock implementation that returns a random embedding
	// In a real implementation, this would call a model to generate the embedding
	embedding := make([]float32, m.Dimensions)
	for i := 0; i < m.Dimensions; i++ {
		embedding[i] = float32(i) / float32(m.Dimensions) // Simple pattern
	}
	return embedding, nil
}

// GenerateResponse generates a response for the given prompt
func (m *MockModel) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// This is a mock implementation that returns the prompt
	// In a real implementation, this would call a model to generate the response
	return "Response to: " + prompt, nil
}

// ContextBuilder builds context for the RAG system
type ContextBuilder struct {
	MaxContextLength int
}

// NewRAGSystem creates a new RAG system
func NewRAGSystem(dimensions int) *RAGSystem {
	modelManager := &ModelManager{
		Models: make(map[string]Model),
	}

	// Add a default model
	modelManager.Models["default"] = &MockModel{
		ID:         "default",
		Dimensions: dimensions,
	}

	return &RAGSystem{
		VectorDB: NewVectorStorage(dimensions),
		QueryProcessor: &QueryProcessor{
			ModelID: "default",
		},
		ModelManager: modelManager,
		ContextBuilder: &ContextBuilder{
			MaxContextLength: 4096,
		},
	}
}

// Query performs a RAG query
func (r *RAGSystem) Query(ctx context.Context, text string) (string, error) {
	// Get the model
	model, exists := r.ModelManager.Models[r.QueryProcessor.ModelID]
	if !exists {
		return "", errors.New("model not found")
	}

	// Generate an embedding for the query
	embedding, err := model.GenerateEmbedding(ctx, text)
	if err != nil {
		return "", err
	}

	// Search for similar documents
	vectorQuery := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    5,
		MinSimilarity: 0.7,
	}

	docs, err := r.VectorDB.SearchSimilar(vectorQuery)
	if err != nil {
		return "", err
	}

	// Build context from retrieved documents
	context := r.buildContext(docs)

	// Generate response with context
	prompt := r.buildPrompt(text, context)
	response, err := model.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

// AddDocument adds a document to the RAG system
func (r *RAGSystem) AddDocument(ctx context.Context, text string, metadata map[string]interface{}) error {
	// Get the model
	model, exists := r.ModelManager.Models[r.QueryProcessor.ModelID]
	if !exists {
		return errors.New("model not found")
	}

	// Generate an embedding for the document
	embedding, err := model.GenerateEmbedding(ctx, text)
	if err != nil {
		return err
	}

	// Create a new document
	doc := types.VectorDocument{
		ID:        generateID(text),
		Text:      text,
		Embedding: embedding,
		Metadata:  metadata,
	}

	// Add the document to the vector storage
	return r.VectorDB.AddDocument(doc)
}

// buildContext builds context from retrieved documents
func (r *RAGSystem) buildContext(docs []types.VectorDocument) string {
	context := ""
	for _, doc := range docs {
		context += doc.Text + "\n\n"
	}
	return context
}

// buildPrompt builds a prompt with context
func (r *RAGSystem) buildPrompt(query string, context string) string {
	return "Context:\n" + context + "\n\nQuestion: " + query + "\n\nAnswer:"
}

// generateID generates a simple ID for a document
func generateID(text string) string {
	// This is a very simple implementation
	// In a real system, you'd want something more robust
	return "doc_" + string(text[0:min(10, len(text))])
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
