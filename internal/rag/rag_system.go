package rag

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/pkg/types"
)

// RAGSystem represents the Retrieval-Augmented Generation system
type RAGSystem struct {
	VectorDB         *VectorStorage
	QueryProcessor   *QueryProcessor
	ModelManager     *ai.ModelManager
	EmbeddedManager  *ai.EmbeddedModelManager  // Reference to embedded Ollama manager
	ContextBuilder   *ContextBuilder
	ContextManager   *ContextManager
	DefaultOptions   ai.GenerateOptions
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
	return NewRAGSystemWithNodeID(dimensions, "default-node")
}

// NewRAGSystemWithNodeID creates a new RAG system with a specific node ID
func NewRAGSystemWithNodeID(dimensions int, nodeID string) *RAGSystem {
	// Create an embedded model manager with Ollama lifecycle management
	embeddedConfig := ai.DefaultEmbeddedManagerConfig()
	embeddedConfig.AutoModels = []string{"cogito", "llama2", "nomic-embed-text"}
	
	embeddedManager := ai.NewEmbeddedModelManager(embeddedConfig)
	
	// Will be set up after RAG system creation
	var ragSystemRef *RAGSystem
	
	// Set up callback to update consciousness default model when ready
	embeddedManager.OnDefaultModelReady = func(modelID string) {
		if ragSystemRef != nil && ragSystemRef.ContextManager != nil {
			log.Printf("Updating consciousness runtime to use embedded model: %s", modelID)
			ragSystemRef.ContextManager.SetModelManager(embeddedManager.ModelManager, modelID)
		}
	}
	
	// Start embedded Ollama (non-blocking)
	go func() {
		ctx := context.Background()
		if err := embeddedManager.Start(ctx); err != nil {
			log.Printf("Failed to start embedded Ollama: %v", err)
			// Continue with fallback models
		}
	}()

	// Register a mock model by default (fallback)
	mockModel := ai.NewMockModel("default", dimensions)
	embeddedManager.RegisterModel(mockModel)

	// Try to register external Ollama models as additional fallback
	if err := tryRegisterOllamaModels(embeddedManager.ModelManager); err != nil {
		log.Printf("Warning: Failed to register external Ollama models: %v", err)
	}

	// Try to register OpenAI models if API key is available
	if err := tryRegisterOpenAIModels(embeddedManager.ModelManager); err != nil {
		log.Printf("Warning: Failed to register OpenAI models: %v", err)
	}

	// Create the RAG system
	ragSystem := &RAGSystem{
		VectorDB: NewVectorStorage(dimensions),
		QueryProcessor: &QueryProcessor{
			ModelID: "default",
		},
		ModelManager:    embeddedManager.ModelManager,
		EmbeddedManager: embeddedManager,
		ContextBuilder: &ContextBuilder{
			MaxContextLength: 4096,
		},
		DefaultOptions: ai.DefaultGenerateOptions(),
	}
	
	// Create and assign the context manager
	ragSystem.ContextManager = NewContextManager(ragSystem, nodeID)
	
	// Set up the reference for callback
	ragSystemRef = ragSystem
	
	return ragSystem
}

// Shutdown gracefully shuts down the RAG system
func (r *RAGSystem) Shutdown() error {
	log.Printf("Shutting down RAG system...")
	
	var shutdownErr error
	
	// Stop embedded Ollama if running
	if r.EmbeddedManager != nil {
		if err := r.EmbeddedManager.Stop(); err != nil {
			log.Printf("Error stopping embedded Ollama: %v", err)
			shutdownErr = err
		}
	}
	
	log.Printf("RAG system shutdown complete")
	return shutdownErr
}

// GetEmbeddedStatus returns the status of embedded Ollama
func (r *RAGSystem) GetEmbeddedStatus() map[string]interface{} {
	if r.EmbeddedManager != nil {
		return r.EmbeddedManager.GetEmbeddedStatus()
	}
	return map[string]interface{}{
		"enabled": false,
		"message": "Embedded Ollama not available",
	}
}

// RestartEmbeddedOllama restarts the embedded Ollama server
func (r *RAGSystem) RestartEmbeddedOllama(ctx context.Context) error {
	if r.EmbeddedManager != nil {
		return r.EmbeddedManager.RestartOllama(ctx)
	}
	return fmt.Errorf("embedded Ollama not available")
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
	return r.QueryWithContext(ctx, text, true, 5)
}

// QueryWithContext performs a RAG query with optional conversation context
func (r *RAGSystem) QueryWithContext(ctx context.Context, text string, useConversationContext bool, contextHistoryCount int) (string, error) {
	startTime := time.Now()
	
	// Track the query
	eventID := r.ContextManager.TrackQuery(ctx, text, "rag", map[string]interface{}{
		"use_conversation_context": useConversationContext,
		"context_history_count":    contextHistoryCount,
	})
	
	var response string
	var err error
	
	defer func() {
		// Track the response
		duration := time.Since(startTime)
		success := err == nil
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}
		r.ContextManager.TrackResponse(ctx, eventID, response, duration, success, errorMsg)
	}()
	
	// Build contextual prompt if requested
	queryText := text
	if useConversationContext && contextHistoryCount > 0 {
		contextualPrompt, contextErr := r.ContextManager.GetContextualPrompt(ctx, text, contextHistoryCount)
		if contextErr != nil {
			log.Printf("Error getting contextual prompt: %v", contextErr)
		} else {
			queryText = contextualPrompt
		}
	}
	
	// Get the model
	model, err := r.ModelManager.GetModel(r.QueryProcessor.ModelID)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	// Generate an embedding for the query (use original text for document search)
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
	documentContext := r.buildContext(docs)

	// Generate response with context (use contextual prompt if available)
	prompt := r.buildPrompt(queryText, documentContext)

	// Use the default options for generation
	options := r.DefaultOptions

	response, err = model.GenerateResponse(ctx, prompt, options)
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
