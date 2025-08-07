package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// EmbeddedModelManager extends ModelManager with embedded Ollama capabilities
type EmbeddedModelManager struct {
	*ModelManager

	// Embedded Ollama
	ollama  *EmbeddedOllama
	enabled bool
	mu      sync.RWMutex

	// Auto-management
	autoStart   bool
	autoModels  []string // Models to auto-pull on startup
	healthCheck bool

	// Callbacks
	OnOllamaStarted     func()
	OnOllamaStopped     func()
	OnModelRegistered   func(modelID string)
	OnDefaultModelReady func(modelID string) // New callback for default model ready
}

// EmbeddedManagerConfig configures the embedded model manager
type EmbeddedManagerConfig struct {
	// Ollama configuration
	OllamaConfig EmbeddedOllamaConfig `json:"ollama"`

	// Management options
	AutoStart   bool     `json:"auto_start"`
	AutoModels  []string `json:"auto_models"`
	HealthCheck bool     `json:"health_check"`

	// Fallback options
	ExternalOllama string `json:"external_ollama"` // URL to external Ollama if embedded fails
}

// DefaultEmbeddedManagerConfig returns default configuration
func DefaultEmbeddedManagerConfig() EmbeddedManagerConfig {
	return EmbeddedManagerConfig{
		OllamaConfig: DefaultEmbeddedOllamaConfig(),
		AutoStart:    true,
		AutoModels:   []string{"cogito", "nomic-embed-text"},
		HealthCheck:  true,
	}
}

// NewEmbeddedModelManager creates a new embedded model manager
func NewEmbeddedModelManager(config EmbeddedManagerConfig) *EmbeddedModelManager {
	manager := &EmbeddedModelManager{
		ModelManager: NewModelManager(),
		autoStart:    config.AutoStart,
		autoModels:   config.AutoModels,
		healthCheck:  config.HealthCheck,
	}

	// Create embedded Ollama instance
	manager.ollama = NewEmbeddedOllama(config.OllamaConfig)

	// Set up callbacks
	manager.ollama.OnStarted = func() {
		log.Printf("Embedded Ollama started successfully")
		if manager.OnOllamaStarted != nil {
			manager.OnOllamaStarted()
		}
		go func() {
			manager.autoRegisterModels()
			// Update default model to use embedded Ollama
			manager.updateDefaultModel()
		}()
	}

	manager.ollama.OnStopped = func() {
		log.Printf("Embedded Ollama stopped")
		if manager.OnOllamaStopped != nil {
			manager.OnOllamaStopped()
		}
	}

	manager.ollama.OnUnhealthy = func(err error) {
		log.Printf("Embedded Ollama unhealthy: %v", err)
		// Could implement auto-restart logic here
	}

	return manager
}

// Start starts the embedded model manager
func (emm *EmbeddedModelManager) Start(ctx context.Context) error {
	emm.mu.Lock()
	defer emm.mu.Unlock()

	if !emm.autoStart {
		log.Printf("Embedded Ollama auto-start disabled, skipping...")
		return nil
	}

	log.Printf("Starting embedded model manager...")

	// Start embedded Ollama
	if err := emm.ollama.Start(ctx); err != nil {
		log.Printf("Failed to start embedded Ollama: %v", err)
		// Try to register external Ollama models as fallback
		emm.registerFallbackModels()
		return err
	}

	emm.enabled = true
	log.Printf("Embedded model manager started successfully")

	return nil
}

// Stop stops the embedded model manager
func (emm *EmbeddedModelManager) Stop() error {
	emm.mu.Lock()
	defer emm.mu.Unlock()

	if !emm.enabled {
		return nil
	}

	log.Printf("Stopping embedded model manager...")

	err := emm.ollama.Stop()
	emm.enabled = false

	log.Printf("Embedded model manager stopped")
	return err
}

// IsEmbeddedRunning returns whether the embedded Ollama is running
func (emm *EmbeddedModelManager) IsEmbeddedRunning() bool {
	emm.mu.RLock()
	defer emm.mu.RUnlock()
	return emm.enabled && emm.ollama.IsRunning()
}

// GetOllamaAddress returns the address of the embedded Ollama server
func (emm *EmbeddedModelManager) GetOllamaAddress() string {
	if emm.ollama != nil {
		return emm.ollama.GetAddress()
	}
	return "http://localhost:11434" // Default fallback
}

// GetOllamaUptime returns how long the embedded Ollama has been running
func (emm *EmbeddedModelManager) GetOllamaUptime() time.Duration {
	if emm.ollama != nil {
		return emm.ollama.GetUptime()
	}
	return 0
}

// RegisterOllamaModel registers an Ollama model with the embedded instance
func (emm *EmbeddedModelManager) RegisterOllamaModel(modelName string, pullIfMissing bool) error {
	emm.mu.Lock()
	defer emm.mu.Unlock()

	// Determine base URL
	baseURL := "http://localhost:11434" // Default
	if emm.enabled && emm.ollama.IsRunning() {
		baseURL = emm.ollama.GetAddress()
	}

	// Check if model exists
	if pullIfMissing && emm.enabled {
		models, err := emm.ollama.ListModels(context.Background())
		if err == nil {
			modelExists := false
			for _, model := range models {
				if strings.Contains(model, modelName) {
					modelExists = true
					break
				}
			}

			if !modelExists {
				log.Printf("Pulling model %s...", modelName)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()

				if err := emm.ollama.PullModel(ctx, modelName); err != nil {
					log.Printf("Failed to pull model %s: %v", modelName, err)
					// Continue with registration anyway - model might exist but not be listed
				}
			}
		}
	}

	// Create and register the model
	config := DefaultOllamaConfig()
	config.BaseURL = baseURL

	model := NewOllamaModel(modelName, config)
	emm.RegisterModel(model)

	log.Printf("Registered Ollama model: %s (URL: %s)", modelName, baseURL)

	if emm.OnModelRegistered != nil {
		emm.OnModelRegistered(model.GetModelInfo().ID)
	}

	return nil
}

// autoRegisterModels automatically registers configured models
func (emm *EmbeddedModelManager) autoRegisterModels() {
	// Use dynamic model registration based on what's actually available
	emm.autoRegisterAvailableModels()

	log.Printf("Auto-registration completed")
}

// registerFallbackModels registers models against external Ollama as fallback
func (emm *EmbeddedModelManager) registerFallbackModels() {
	log.Printf("Registering fallback Ollama models...")

	// Try to register models against default Ollama installation
	for _, modelName := range emm.autoModels {
		config := DefaultOllamaConfig()
		config.BaseURL = "http://localhost:11434"

		model := NewOllamaModel(modelName, config)
		emm.RegisterModel(model)

		log.Printf("Registered fallback Ollama model: %s", modelName)
	}
}

// RestartOllama restarts the embedded Ollama server
func (emm *EmbeddedModelManager) RestartOllama(ctx context.Context) error {
	emm.mu.Lock()
	defer emm.mu.Unlock()

	if !emm.enabled {
		return fmt.Errorf("embedded Ollama is not enabled")
	}

	log.Printf("Restarting embedded Ollama...")

	if err := emm.ollama.Restart(ctx); err != nil {
		return fmt.Errorf("failed to restart embedded Ollama: %w", err)
	}

	// Re-register models after restart
	go emm.autoRegisterModels()

	return nil
}

// GetEmbeddedStatus returns detailed status of the embedded Ollama
func (emm *EmbeddedModelManager) GetEmbeddedStatus() map[string]interface{} {
	emm.mu.RLock()
	defer emm.mu.RUnlock()

	status := map[string]interface{}{
		"enabled":     emm.enabled,
		"running":     emm.ollama.IsRunning(),
		"address":     emm.GetOllamaAddress(),
		"uptime":      emm.GetOllamaUptime().String(),
		"auto_start":  emm.autoStart,
		"auto_models": emm.autoModels,
	}

	if emm.enabled {
		// Get model list if possible
		if models, err := emm.ollama.ListModels(context.Background()); err == nil {
			status["available_models"] = models
		}
	}

	return status
}

// PullModel pulls a model using the embedded Ollama
func (emm *EmbeddedModelManager) PullModel(ctx context.Context, modelName string, autoRegister bool) error {
	emm.mu.Lock()
	defer emm.mu.Unlock()

	if !emm.enabled || !emm.ollama.IsRunning() {
		return fmt.Errorf("embedded Ollama is not running")
	}

	if err := emm.ollama.PullModel(ctx, modelName); err != nil {
		return fmt.Errorf("failed to pull model %s: %w", modelName, err)
	}

	if autoRegister {
		return emm.RegisterOllamaModel(modelName, false)
	}

	return nil
}

// GetAvailableModels returns models available in the embedded Ollama
func (emm *EmbeddedModelManager) GetAvailableModels(ctx context.Context) ([]string, error) {
	emm.mu.RLock()
	defer emm.mu.RUnlock()

	if !emm.enabled || !emm.ollama.IsRunning() {
		return nil, fmt.Errorf("embedded Ollama is not running")
	}

	return emm.ollama.ListModels(ctx)
}

// EnableHealthMonitoring enables periodic health checks for registered models
func (emm *EmbeddedModelManager) EnableHealthMonitoring(ctx context.Context, interval time.Duration) {
	if !emm.healthCheck {
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				emm.checkModelHealth(ctx)
			}
		}
	}()
}

// checkModelHealth checks the health of all registered models
func (emm *EmbeddedModelManager) checkModelHealth(ctx context.Context) {
	models := emm.ListModels()
	unhealthyCount := 0

	for _, modelInfo := range models {
		if model, err := emm.GetModel(modelInfo.ID); err == nil {
			if !model.IsHealthy(ctx) {
				unhealthyCount++
				log.Printf("Model %s is unhealthy", modelInfo.ID)
			}
		}
	}

	if unhealthyCount > 0 {
		log.Printf("%d/%d models are unhealthy", unhealthyCount, len(models))
	}
}

// updateDefaultModel updates the default model to use embedded Ollama models
func (emm *EmbeddedModelManager) updateDefaultModel() {
	// Find a good default model from the registered models
	models := emm.ListModels()

	var defaultModelID string

	// Prefer cogito if available
	for _, model := range models {
		if strings.Contains(model.ID, "cogito") {
			defaultModelID = model.ID
			break
		}
	}

	// Fallback to llama3.1 if cogito not found
	if defaultModelID == "" {
		for _, model := range models {
			if strings.Contains(model.ID, "llama3.1") {
				defaultModelID = model.ID
				break
			}
		}
	}

	// Further fallback to any llama3 variant
	if defaultModelID == "" {
		for _, model := range models {
			if strings.Contains(model.ID, "llama3") {
				defaultModelID = model.ID
				break
			}
		}
	}

	// Use any Ollama model if no specific preference found
	if defaultModelID == "" {
		for _, model := range models {
			if strings.Contains(model.ID, "ollama-") {
				defaultModelID = model.ID
				break
			}
		}
	}

	if defaultModelID != "" {
		log.Printf("Updated default model to: %s", defaultModelID)
		if emm.OnDefaultModelReady != nil {
			emm.OnDefaultModelReady(defaultModelID)
		}
	}
}

// getAvailableOllamaModels returns a list of models available in Ollama
func (emm *EmbeddedModelManager) getAvailableOllamaModels() []string {
	if emm.ollama == nil {
		return []string{}
	}

	// Get list of installed models from Ollama
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	availableModels, err := emm.ollama.ListModels(ctx)
	if err != nil {
		log.Printf("Failed to get available Ollama models: %v", err)
		return []string{}
	}

	log.Printf("Found %d available Ollama models: %v", len(availableModels), availableModels)
	return availableModels
}

// autoRegisterAvailableModels registers only models that are actually available
func (emm *EmbeddedModelManager) autoRegisterAvailableModels() {
	availableModels := emm.getAvailableOllamaModels()
	if len(availableModels) == 0 {
		log.Printf("No Ollama models available for registration")
		return
	}

	// Preferred models in order of preference
	preferredModels := []string{"cogito", "llama3.2:3b", "llama3.2:1b", "nomic-embed-text"}

	registeredCount := 0
	for _, preferred := range preferredModels {
		for _, available := range availableModels {
			if available == preferred || strings.HasPrefix(available, preferred) {
				if err := emm.RegisterOllamaModel(available, true); err != nil {
					log.Printf("Failed to register available model %s: %v", available, err)
				} else {
					registeredCount++
				}
				break // Move to next preferred model
			}
		}
	}

	// If no preferred models were found, register any available models
	if registeredCount == 0 {
		log.Printf("No preferred models available, registering first available model")
		for _, available := range availableModels {
			if err := emm.RegisterOllamaModel(available, true); err != nil {
				log.Printf("Failed to register fallback model %s: %v", available, err)
			} else {
				log.Printf("Registered fallback model: %s", available)
				break // Just register one fallback model
			}
		}
	}

	log.Printf("Auto-registration complete. Registered %d models from %d available", registeredCount, len(availableModels))
}
