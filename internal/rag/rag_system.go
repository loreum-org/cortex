package rag

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/pkg/types"
)

// RAGSystem represents the Retrieval-Augmented Generation system
type RAGSystem struct {
	VectorDB       *VectorStorage
	QueryProcessor *QueryProcessor
	ModelManager   *ai.ModelManager
	ContextBuilder *ContextBuilder
	DefaultOptions ai.GenerateOptions
}

// QueryProcessor processes queries for the RAG system
type QueryProcessor struct {
	ModelID string
}

// ContextBuilder builds context for the RAG system
type ContextBuilder struct {
	MaxContextLength int
}

// NewRAGSystem creates a new RAG system
func NewRAGSystem(dimensions int) *RAGSystem {
	// Create a new model manager
	modelManager := ai.NewModelManager()

	// Register a mock model by default
	mockModel := ai.NewMockModel("default", dimensions)
	modelManager.RegisterModel(mockModel)

	// Try to register Ollama models if available
	if err := tryRegisterOllamaModels(modelManager); err != nil {
		log.Printf("Warning: Failed to register Ollama models: %v", err)
	}

	// Try to register OpenAI models if API key is available
	if err := tryRegisterOpenAIModels(modelManager); err != nil {
		log.Printf("Warning: Failed to register OpenAI models: %v", err)
	}

	// Create the RAG system
	return &RAGSystem{
		VectorDB: NewVectorStorage(dimensions),
		QueryProcessor: &QueryProcessor{
			ModelID: "default",
		},
		ModelManager: modelManager,
		ContextBuilder: &ContextBuilder{
			MaxContextLength: 4096,
		},
		DefaultOptions: ai.DefaultGenerateOptions(),
	}
}

// tryRegisterOllamaModels attempts to register Ollama models if available
func tryRegisterOllamaModels(manager *ai.ModelManager) error {
	// Check if Ollama is available
	config := ai.DefaultOllamaConfig()

	// Create a test model to check connectivity
	testModel := ai.NewOllamaModel("llama2", config)

	// Check if Ollama is healthy
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	if !testModel.IsHealthy(ctx) {
		return errors.New("Ollama service not available")
	}

	// Register default models
	models := []string{"llama2", "mistral", "codellama"}
	for _, modelName := range models {
		model := ai.NewOllamaModel(modelName, config)
		manager.RegisterModel(model)
		log.Printf("Registered Ollama model: %s", modelName)
	}

	return nil
}

// tryRegisterOpenAIModels attempts to register OpenAI models if API key is available
func tryRegisterOpenAIModels(manager *ai.ModelManager) error {
	// Check if OpenAI API key is available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return errors.New("OpenAI API key not set")
	}

	// Create OpenAI configuration
	config := ai.DefaultOpenAIConfig()
	config.APIKey = apiKey

	// Register default models
	models := []string{"gpt-3.5-turbo", "gpt-4", "text-embedding-ada-002"}
	for _, modelName := range models {
		model, err := ai.NewOpenAIModel(modelName, config)
		if err != nil {
			log.Printf("Error creating OpenAI model %s: %v", modelName, err)
			continue
		}

		manager.RegisterModel(model)
		log.Printf("Registered OpenAI model: %s", modelName)
	}

	return nil
}

// SetDefaultModel sets the default model for the RAG system
func (r *RAGSystem) SetDefaultModel(modelID string) error {
	// Check if the model exists
	_, err := r.ModelManager.GetModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to set default model: %w", err)
	}

	r.QueryProcessor.ModelID = modelID
	return nil
}

// ListAvailableModels returns a list of available models
func (r *RAGSystem) ListAvailableModels() []ai.ModelInfo {
	return r.ModelManager.ListModels()
}

// Query performs a RAG query
func (r *RAGSystem) Query(ctx context.Context, text string) (string, error) {
	// Get the model
	model, err := r.ModelManager.GetModel(r.QueryProcessor.ModelID)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	// Generate an embedding for the query
	embedding, err := model.GenerateEmbedding(ctx, text)
	if err != nil {
		return "", fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search for similar documents
	vectorQuery := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    5,
		MinSimilarity: 0.7,
	}

	docs, err := r.VectorDB.SearchSimilar(vectorQuery)
	if err != nil {
		return "", fmt.Errorf("failed to search similar documents: %w", err)
	}

	// Build context from retrieved documents
	context := r.buildContext(docs)

	// Generate response with context
	prompt := r.buildPrompt(text, context)

	// Use the default options for generation
	options := r.DefaultOptions

	response, err := model.GenerateResponse(ctx, prompt, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return response, nil
}

// AddDocument adds a document to the RAG system
func (r *RAGSystem) AddDocument(ctx context.Context, text string, metadata map[string]interface{}) error {
	// Get the model
	model, err := r.ModelManager.GetModel(r.QueryProcessor.ModelID)
	if err != nil {
		return fmt.Errorf("add document failed: %w", err)
	}

	// Generate an embedding for the document
	embedding, err := model.GenerateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create a new document
	doc := types.VectorDocument{
		ID:        generateID(text),
		Text:      text,
		Embedding: embedding,
		Metadata:  metadata,
	}

	// Add the document to the vector storage
	if err := r.VectorDB.AddDocument(doc); err != nil {
		return fmt.Errorf("failed to add document to vector storage: %w", err)
	}

	log.Printf("Added document with ID: %s", doc.ID)
	return nil
}

// buildContext builds context from retrieved documents
func (r *RAGSystem) buildContext(docs []types.VectorDocument) string {
	var builder strings.Builder

	for _, doc := range docs {
		builder.WriteString(doc.Text)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

// buildPrompt builds a prompt with context
func (r *RAGSystem) buildPrompt(query string, context string) string {
	return fmt.Sprintf("Context:\n%s\n\nQuestion: %s\n\nAnswer:", context, query)
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
