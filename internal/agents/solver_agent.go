package agents

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
)

// SolverConfig holds configuration for the solver agent
type SolverConfig struct {
	DefaultModel    string
	PredictorConfig *PredictorConfig
}

// PredictorConfig holds configuration for the predictor
type PredictorConfig struct {
	Timeout      time.Duration
	MaxTokens    int
	Temperature  float64
	CacheResults bool
}

// SolverAgent implements the Agent interface for solving queries
type SolverAgent struct {
	modelManager   *ai.ModelManager
	ragSystem      *rag.RAGSystem
	config         *SolverConfig
	metrics        types.Metrics
	defaultOptions ai.GenerateOptions
	economicEngine interface {
		RecordQueryResult(nodeID, queryID string, success bool, responseTimeMs int64)
	}
	nodeID string
}

// NewSolverAgent creates a new solver agent
func NewSolverAgent(config *SolverConfig) *SolverAgent {
	return NewSolverAgentWithRAG(config, nil)
}

// NewSolverAgentWithRAG creates a new solver agent with RAG system
func NewSolverAgentWithRAG(config *SolverConfig, ragSystem *rag.RAGSystem) *SolverAgent {
	// Initialize config if nil
	if config == nil {
		config = &SolverConfig{}
	}

	// Create a new model manager
	modelManager := ai.NewModelManager()

	// Register a mock model as fallback
	mockModel := ai.NewMockModel("default", 384)
	modelManager.RegisterModel(mockModel)

	// Try to register Ollama models if available
	ollamaAvailable := false
	if err := tryRegisterOllamaModels(modelManager); err != nil {
		log.Printf("Warning: Failed to register Ollama models: %v", err)
	} else {
		ollamaAvailable = true
	}

	// Try to register OpenAI models if API key is available
	openaiAvailable := false
	if err := tryRegisterOpenAIModels(modelManager); err != nil {
		log.Printf("Warning: Failed to register OpenAI models: %v", err)
	} else {
		openaiAvailable = true
	}

	// Set default model preference: Ollama > OpenAI > Mock
	if config.DefaultModel == "" {
		if ollamaAvailable {
			config.DefaultModel = "ollama-cogito" // First Ollama model registered
		} else if openaiAvailable {
			config.DefaultModel = "openai-gpt-3.5-turbo" // First OpenAI model registered
		} else {
			config.DefaultModel = "default" // Fall back to mock
		}
		log.Printf("Auto-selected default model: %s", config.DefaultModel)
	}

	// Create default options from config
	defaultOptions := ai.DefaultGenerateOptions()
	if config.PredictorConfig != nil {
		defaultOptions.MaxTokens = config.PredictorConfig.MaxTokens
		defaultOptions.Temperature = float32(config.PredictorConfig.Temperature)
	}

	agent := &SolverAgent{
		modelManager: modelManager,
		ragSystem:    ragSystem,
		config:       config,
		metrics: types.Metrics{
			ResponseTime:   0.0,
			SuccessRate:    1.0,
			ErrorRate:      0.0,
			RequestsPerMin: 0,
		},
		defaultOptions: defaultOptions,
	}

	// Configure consciousness runtime with model manager if RAG system is available
	if ragSystem != nil && ragSystem.ContextManager != nil {
		ragSystem.ContextManager.SetModelManager(modelManager, config.DefaultModel)
	}

	return agent
}

// SetRAGSystem sets the RAG system for the solver agent
func (s *SolverAgent) SetRAGSystem(ragSystem *rag.RAGSystem) {
	s.ragSystem = ragSystem

	// Configure consciousness runtime with model manager if available
	if ragSystem != nil && ragSystem.ContextManager != nil && s.modelManager != nil {
		ragSystem.ContextManager.SetModelManager(s.modelManager, s.config.DefaultModel)
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
		return fmt.Errorf("Ollama service not available")
	}

	// Register default models - use cogito which is available
	models := []string{"cogito", "llama2", "mistral", "codellama"}
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
		return fmt.Errorf("OpenAI API key not set")
	}

	// Create OpenAI configuration
	config := ai.DefaultOpenAIConfig()
	config.APIKey = apiKey

	// Register default models
	models := []string{"gpt-3.5-turbo", "gpt-4"}
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

// Process processes a query with context tracking
func (s *SolverAgent) Process(ctx context.Context, query *types.Query) (*types.Response, error) {
	startTime := time.Now()
	success := false
	var eventID string
	var ragSources []string

	// Track the query if RAG system is available
	if s.ragSystem != nil && s.ragSystem.ContextManager != nil {
		eventID = s.ragSystem.ContextManager.TrackQuery(ctx, query.Text, query.Type, map[string]interface{}{
			"query_id": query.ID,
			"model":    s.config.DefaultModel,
		})

		// Try consciousness-based processing first
		if consciousnessRuntime := s.ragSystem.ContextManager.GetConsciousnessRuntime(); consciousnessRuntime != nil {
			log.Printf("Processing query with consciousness runtime: %s", query.Text)
			response, err := consciousnessRuntime.ProcessQuery(ctx, query)
			if err == nil {
				success = true
				return response, nil
			}
			log.Printf("Consciousness processing failed, falling back to standard processing: %v", err)
		}
	}

	// Record query result at the end
	defer func() {
		responseTime := time.Since(startTime).Milliseconds()
		s.updateMetrics(time.Since(startTime), success)

		// Track the response if RAG system is available
		if s.ragSystem != nil && s.ragSystem.ContextManager != nil && eventID != "" {
			var responseText string
			var errorMsg string
			if success {
				responseText = "Query processed successfully"
			} else {
				errorMsg = "Query processing failed"
			}
			s.ragSystem.ContextManager.TrackResponse(ctx, eventID, responseText, time.Since(startTime), success, errorMsg)
		}

		// Record result for staking system if economic engine is available
		if s.economicEngine != nil && s.nodeID != "" {
			s.economicEngine.RecordQueryResult(s.nodeID, query.ID, success, responseTime)
		}
	}()

	// Check if this should use RAG processing
	if s.ragSystem != nil && (query.Type == "rag" || strings.Contains(strings.ToLower(query.Text), "document") || strings.Contains(strings.ToLower(query.Text), "search")) {
		// Use RAG system for processing
		log.Printf("Processing query with RAG system")
		output, err := s.ragSystem.QueryWithContext(ctx, query.Text, true, 5)
		if err != nil {
			return nil, fmt.Errorf("RAG query failed: %w", err)
		}

		// Format the response with RAG metadata
		response, err := s.formatResponseWithRAG(query, output, "rag-system", ragSources)
		if err != nil {
			return nil, fmt.Errorf("failed to format RAG response: %w", err)
		}

		success = true
		return response, nil
	}

	// Standard processing with contextual enhancement
	// Select the appropriate model
	model, err := s.selectModel(query)
	if err != nil {
		return nil, fmt.Errorf("failed to select model: %w", err)
	}

	// Prepare the input for the model with context if available
	input, err := s.prepareContextualInput(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare input: %w", err)
	}

	// Create generation options
	options := s.defaultOptions

	// Adjust options based on query type if needed
	if query.Type == "code" || query.Type == "programming" {
		options.Temperature = 0.3 // Lower temperature for code generation
	} else if query.Type == "creative" {
		options.Temperature = 0.9 // Higher temperature for creative tasks
	}

	// Run inference
	log.Printf("Processing query with model: %s", model.GetModelInfo().ID)
	output, err := model.GenerateResponse(ctx, input, options)
	if err != nil {
		return nil, fmt.Errorf("model inference failed: %w", err)
	}

	// Format the response
	response, err := s.formatResponseWithRAG(query, output, model.GetModelInfo().ID, ragSources)
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	success = true
	return response, nil
}

// GetCapabilities returns the capabilities of the agent
func (s *SolverAgent) GetCapabilities() []types.Capability {
	return []types.Capability{"solve", "infer", "generate"}
}

// GetPerformanceMetrics returns the performance metrics of the agent
func (s *SolverAgent) GetPerformanceMetrics() types.Metrics {
	return s.metrics
}

// selectModel selects the appropriate model for the query
func (s *SolverAgent) selectModel(query *types.Query) (ai.AIModel, error) {
	// First try to use a model specified in the query metadata
	if modelID, ok := query.Metadata["model_id"]; ok {
		model, err := s.modelManager.GetModel(modelID)
		if err == nil {
			return model, nil
		}
		// Log the error but continue to try other models
		log.Printf("Requested model %s not found, falling back to selection logic", modelID)
	}

	// Select model based on query type
	var preferredModelID string

	// Check query type for specialized models
	if query.Type == "code" || query.Type == "programming" {
		// Try to find a code-specialized model
		models := s.modelManager.ListModels()
		for _, info := range models {
			for _, cap := range info.Capabilities {
				if cap == ai.CapabilityCodeGen {
					preferredModelID = info.ID
					break
				}
			}
			if preferredModelID != "" {
				break
			}
		}
	}

	// If we found a preferred model, try to use it
	if preferredModelID != "" {
		model, err := s.modelManager.GetModel(preferredModelID)
		if err == nil {
			return model, nil
		}
		// Log the error but continue to default model
		log.Printf("Preferred model %s not available, falling back to default", preferredModelID)
	}

	// Use the default model
	defaultID := s.config.DefaultModel
	if defaultID == "" {
		defaultID = "default" // Fallback to mock model
	}

	model, err := s.modelManager.GetModel(defaultID)
	if err != nil {
		// If default model is not found, try to use any available model
		models := s.modelManager.ListModels()
		if len(models) > 0 {
			model, err = s.modelManager.GetModel(models[0].ID)
			if err == nil {
				return model, nil
			}
		}
		return nil, fmt.Errorf("no suitable model found: %w", err)
	}

	return model, nil
}

// prepareInput prepares the input for the model
func (s *SolverAgent) prepareInput(query *types.Query) (string, error) {
	// For simple queries, just return the text
	if query.Type == "" || query.Type == "question" {
		return query.Text, nil
	}

	// For code queries, add a prefix
	if query.Type == "code" || query.Type == "programming" {
		return fmt.Sprintf("Write code to solve the following problem: %s", query.Text), nil
	}

	// For creative queries, add a prefix
	if query.Type == "creative" {
		return fmt.Sprintf("Be creative and generate: %s", query.Text), nil
	}

	// For other types, include the type in the prompt
	return fmt.Sprintf("[%s] %s", strings.ToUpper(query.Type), query.Text), nil
}

// prepareContextualInput prepares the input for the model with conversation context
func (s *SolverAgent) prepareContextualInput(ctx context.Context, query *types.Query) (string, error) {
	// Start with the basic input preparation
	baseInput, err := s.prepareInput(query)
	if err != nil {
		return "", err
	}

	// If RAG system is available, try to get contextual prompt
	if s.ragSystem != nil && s.ragSystem.ContextManager != nil {
		contextualPrompt, contextErr := s.ragSystem.ContextManager.GetContextualPrompt(ctx, baseInput, 3)
		if contextErr != nil {
			log.Printf("Error getting contextual prompt: %v", contextErr)
			return baseInput, nil // Fallback to base input
		}
		return contextualPrompt, nil
	}

	return baseInput, nil
}

// formatResponse formats the output as a response
func (s *SolverAgent) formatResponse(query *types.Query, output string, modelID string) (*types.Response, error) {
	return &types.Response{
		QueryID:   query.ID,
		Text:      output,
		Data:      map[string]string{"model": modelID},
		Metadata:  query.Metadata,
		Status:    "success",
		Timestamp: time.Now().Unix(),
	}, nil
}

// formatResponseWithRAG formats the output as a response with RAG metadata
func (s *SolverAgent) formatResponseWithRAG(query *types.Query, output string, modelID string, ragSources []string) (*types.Response, error) {
	data := map[string]string{"model": modelID}

	if len(ragSources) > 0 {
		data["rag_sources"] = strings.Join(ragSources, ", ")
		data["context_used"] = "true"
	}

	// Add conversation tracking info if available
	if s.ragSystem != nil && s.ragSystem.ContextManager != nil {
		data["conversation_id"] = s.ragSystem.ContextManager.GetConversationID()
		data["context_tracking"] = "enabled"
	}

	return &types.Response{
		QueryID:   query.ID,
		Text:      output,
		Data:      data,
		Metadata:  query.Metadata,
		Status:    "success",
		Timestamp: time.Now().Unix(),
	}, nil
}

// updateMetrics updates the performance metrics
func (s *SolverAgent) updateMetrics(duration time.Duration, success bool) {
	// Update response time (exponential moving average) - use microseconds for precision
	durationMs := float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds with decimal precision
	s.metrics.ResponseTime = 0.9*s.metrics.ResponseTime + 0.1*durationMs

	// Update success/error rate
	if success {
		s.metrics.SuccessRate = 0.99*s.metrics.SuccessRate + 0.01
		s.metrics.ErrorRate = 0.99 * s.metrics.ErrorRate
	} else {
		s.metrics.SuccessRate = 0.99 * s.metrics.SuccessRate
		s.metrics.ErrorRate = 0.99*s.metrics.ErrorRate + 0.01
	}

	// Simple increment for requests per minute
	// In a real implementation, this would involve a sliding window
	s.metrics.RequestsPerMin++
}

// RegisterModel registers a model with the agent
func (s *SolverAgent) RegisterModel(model ai.AIModel) {
	s.modelManager.RegisterModel(model)
}

// GetModel returns a model by ID
func (s *SolverAgent) GetModel(id string) (ai.AIModel, error) {
	return s.modelManager.GetModel(id)
}

// ListModels returns a list of all registered models
func (s *SolverAgent) ListModels() []ai.ModelInfo {
	return s.modelManager.ListModels()
}

// SetDefaultModel sets the default model for the agent
func (s *SolverAgent) SetDefaultModel(modelID string) error {
	// Check if the model exists
	_, err := s.modelManager.GetModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to set default model: %w", err)
	}

	s.config.DefaultModel = modelID
	return nil
}

// SetEconomicEngine sets the economic engine for query result tracking
func (s *SolverAgent) SetEconomicEngine(engine interface {
	RecordQueryResult(nodeID, queryID string, success bool, responseTimeMs int64)
}) {
	s.economicEngine = engine
}

// SetNodeID sets the node ID for this solver agent
func (s *SolverAgent) SetNodeID(nodeID string) {
	s.nodeID = nodeID
}
