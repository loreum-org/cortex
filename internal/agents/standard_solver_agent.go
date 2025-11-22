package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
)

// StandardSolverAgent implements the StandardAgent interface for solving queries
type StandardSolverAgent struct {
	*BaseAgent
	modelManager   *ai.ModelManager
	ragSystem      *rag.RAGSystem
	solverConfig   *SolverConfig
	defaultOptions ai.GenerateOptions
	economicEngine interface {
		RecordQueryResult(nodeID, queryID string, success bool, responseTimeMs int64)
	}
	nodeID string
}

// NewStandardSolverAgent creates a new standard solver agent
func NewStandardSolverAgent(nodeID string, config *SolverConfig, ragSystem *rag.RAGSystem) *StandardSolverAgent {
	// Initialize config if nil
	if config == nil {
		config = &SolverConfig{
			DefaultModel: "ollama-cogito:latest",
		}
	}

	// Create agent info
	agentInfo := AgentInfo{
		ID:          fmt.Sprintf("solver-%s", nodeID[:8]),
		Name:        "Primary Solver Agent",
		Description: "Main query processing and problem-solving agent with RAG capabilities",
		Type:        AgentTypeSolver,
		Version:     "1.0.0",
		Status:      AgentStatusInactive,
		Capabilities: []AgentCapability{
			{
				Name:        "natural_language_processing",
				Description: "Process and understand natural language queries",
				Version:     "1.0.0",
				Parameters: map[string]interface{}{
					"max_tokens":  4096,
					"temperature": 0.7,
					"top_p":       0.9,
				},
				InputTypes:  []string{"user_query", "natural_language", "text", "question"},
				OutputTypes: []string{"text", "structured_response"},
			},
			{
				Name:        "rag_retrieval",
				Description: "Retrieve relevant information from vector database",
				Version:     "1.2.0",
				Parameters: map[string]interface{}{
					"top_k":                10,
					"similarity_threshold": 0.7,
					"max_context_length":   8192,
				},
				InputTypes:  []string{"user_query", "search_query", "question"},
				OutputTypes: []string{"retrieved_context", "augmented_response"},
			},
			{
				Name:        "context_awareness",
				Description: "Maintain conversation context and memory",
				Version:     "1.1.0",
				Parameters: map[string]interface{}{
					"context_window":     8192,
					"memory_retention":   "persistent",
					"conversation_depth": 10,
				},
				InputTypes:  []string{"conversation", "context_query"},
				OutputTypes: []string{"contextual_response"},
			},
		},
		ModelID:   config.DefaultModel,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create base agent
	baseAgent := NewBaseAgent(agentInfo)

	// Create solver agent
	solver := &StandardSolverAgent{
		BaseAgent:      baseAgent,
		solverConfig:   config,
		ragSystem:      ragSystem,
		nodeID:         nodeID,
		defaultOptions: ai.DefaultGenerateOptions(),
	}

	// Set process function
	baseAgent.SetProcessFunction(solver.processQuery)

	return solver
}

// Initialize initializes the solver agent
func (s *StandardSolverAgent) Initialize(ctx context.Context, config map[string]interface{}) error {
	if err := s.BaseAgent.Initialize(ctx, config); err != nil {
		return err
	}

	// Initialize model manager
	s.modelManager = ai.NewModelManager()

	// Register a mock model as fallback
	mockModel := ai.NewMockModel("default", 384)
	s.modelManager.RegisterModel(mockModel)

	// Try to register Ollama models if available
	ollamaAvailable := false
	if err := tryRegisterOllamaModels(s.modelManager); err != nil {
		log.Printf("Warning: Failed to register Ollama models: %v", err)
	} else {
		ollamaAvailable = true
	}

	// Try to register OpenAI models if API key is available
	openaiAvailable := false
	if err := tryRegisterOpenAIModels(s.modelManager); err != nil {
		log.Printf("Warning: Failed to register OpenAI models: %v", err)
	} else {
		openaiAvailable = true
	}

	// Set default model preference: Ollama > OpenAI > Mock
	if s.solverConfig.DefaultModel == "" {
		if ollamaAvailable {
			s.solverConfig.DefaultModel = "ollama-cogito:latest"
		} else if openaiAvailable {
			s.solverConfig.DefaultModel = "openai-gpt-3.5-turbo"
		} else {
			s.solverConfig.DefaultModel = "default"
		}
		log.Printf("Auto-selected default model: %s", s.solverConfig.DefaultModel)
	}

	// Create default options from config
	s.defaultOptions = ai.DefaultGenerateOptions()
	if s.solverConfig.PredictorConfig != nil {
		s.defaultOptions.MaxTokens = s.solverConfig.PredictorConfig.MaxTokens
		s.defaultOptions.Temperature = float32(s.solverConfig.PredictorConfig.Temperature)
	}

	// Configure consciousness runtime with model manager if RAG system is available
	if s.ragSystem != nil && s.ragSystem.ContextManager != nil {
		s.ragSystem.ContextManager.SetModelManager(s.modelManager, s.solverConfig.DefaultModel)
	}

	return nil
}

// Start starts the solver agent
func (s *StandardSolverAgent) Start(ctx context.Context) error {
	if err := s.BaseAgent.Start(ctx); err != nil {
		return err
	}

	log.Printf("Standard Solver Agent %s started successfully", s.GetInfo().ID)
	return nil
}

// processQuery processes a query using the solver agent's capabilities
func (s *StandardSolverAgent) processQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	startTime := time.Now()

	// Record query for economic tracking
	defer func() {
		if s.economicEngine != nil {
			responseTime := time.Since(startTime)
			success := true // This would be set based on actual result
			s.economicEngine.RecordQueryResult(s.nodeID, query.ID, success, responseTime.Milliseconds())
		}
	}()

	// Use AGI consciousness processing if available
	if s.ragSystem != nil {
		agiSystem := s.ragSystem.GetAGISystem()
		if agiSystem != nil {
			resp, err := agiSystem.ProcessQueryWithConsciousness(ctx, query)
			if err == nil && resp != nil {
				return &types.Response{
					QueryID:   query.ID,
					Text:      resp.Text,
					Data:      resp.Data,
					Metadata:  resp.Metadata,
					Status:    resp.Status,
					Timestamp: time.Now().Unix(),
				}, nil
			}
			log.Printf("Consciousness runtime processing failed: %v", err)
		}
	}

	// Fallback to direct model processing
	if s.modelManager == nil {
		return nil, fmt.Errorf("model manager not initialized")
	}

	model, err := s.modelManager.GetModel(s.solverConfig.DefaultModel)
	if err != nil {
		return nil, fmt.Errorf("failed to get model %s: %w", s.solverConfig.DefaultModel, err)
	}

	// Generate response
	responseText, err := model.GenerateResponse(ctx, query.Text, s.defaultOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Create response
	response := &types.Response{
		QueryID:   query.ID,
		Text:      responseText,
		Metadata:  make(map[string]string),
		Status:    "success",
		Timestamp: time.Now().Unix(),
	}

	// Add metadata
	response.Metadata["model"] = s.solverConfig.DefaultModel
	response.Metadata["agent_id"] = s.GetInfo().ID
	response.Metadata["agent_type"] = string(AgentTypeSolver)
	response.Metadata["processing_time"] = time.Since(startTime).String()

	return response, nil
}

// SetRAGSystem sets the RAG system for the solver agent
func (s *StandardSolverAgent) SetRAGSystem(ragSystem *rag.RAGSystem) {
	s.ragSystem = ragSystem

	// Configure consciousness runtime with model manager if available
	if ragSystem != nil && ragSystem.ContextManager != nil && s.modelManager != nil {
		ragSystem.ContextManager.SetModelManager(s.modelManager, s.solverConfig.DefaultModel)
	}
}

// SetModelManager sets the model manager and updates the default model
func (s *StandardSolverAgent) SetModelManager(modelManager *ai.ModelManager, defaultModel string) {
	s.modelManager = modelManager
	s.solverConfig.DefaultModel = defaultModel

	// Update agent info
	info := s.GetInfo()
	info.ModelID = defaultModel
	s.UpdateInfo(info)

	// Update consciousness runtime if available
	if s.ragSystem != nil && s.ragSystem.ContextManager != nil {
		s.ragSystem.ContextManager.SetModelManager(modelManager, defaultModel)
	}

	log.Printf("[StandardSolver] SetModelManager called with defaultModel: %s", defaultModel)
}

// SetEconomicEngine sets the economic engine for query result tracking
func (s *StandardSolverAgent) SetEconomicEngine(engine interface {
	RecordQueryResult(nodeID, queryID string, success bool, responseTimeMs int64)
}) {
	s.economicEngine = engine
}

// HealthCheck performs a health check on the solver agent
func (s *StandardSolverAgent) HealthCheck(ctx context.Context) error {
	if err := s.BaseAgent.HealthCheck(ctx); err != nil {
		return err
	}

	// Check if model manager is available
	if s.modelManager == nil {
		return fmt.Errorf("model manager not initialized")
	}

	// Check if default model is available
	_, err := s.modelManager.GetModel(s.solverConfig.DefaultModel)
	if err != nil {
		return fmt.Errorf("default model %s not available: %w", s.solverConfig.DefaultModel, err)
	}

	// Check RAG system health if available
	if s.ragSystem != nil {
		// Could add RAG system health check here
	}

	return nil
}

// UpdateConfig updates the solver agent configuration
func (s *StandardSolverAgent) UpdateConfig(config map[string]interface{}) error {
	if err := s.BaseAgent.UpdateConfig(config); err != nil {
		return err
	}

	// Handle solver-specific config updates
	if defaultModel, ok := config["default_model"].(string); ok {
		s.solverConfig.DefaultModel = defaultModel

		// Update agent info
		info := s.GetInfo()
		info.ModelID = defaultModel
		s.UpdateInfo(info)
	}

	if tempInterface, ok := config["temperature"]; ok {
		if temp, ok := tempInterface.(float64); ok {
			s.defaultOptions.Temperature = float32(temp)
		}
	}

	if maxTokensInterface, ok := config["max_tokens"]; ok {
		if maxTokens, ok := maxTokensInterface.(int); ok {
			s.defaultOptions.MaxTokens = maxTokens
		}
	}

	return nil
}

// GetModelManager returns the model manager
func (s *StandardSolverAgent) GetModelManager() *ai.ModelManager {
	return s.modelManager
}

// GetRAGSystem returns the RAG system
func (s *StandardSolverAgent) GetRAGSystem() *rag.RAGSystem {
	return s.ragSystem
}
