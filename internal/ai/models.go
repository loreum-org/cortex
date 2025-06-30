package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Common errors
var (
	ErrModelNotFound      = errors.New("model not found")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrUnsupportedModel   = errors.New("unsupported model")
	ErrAPIKeyNotSet       = errors.New("API key not set")
	ErrContextExceeded    = errors.New("context length exceeded")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout            = errors.New("request timed out")
	ErrRateLimited        = errors.New("rate limited")
)

// ModelType represents the type of AI model
type ModelType string

const (
	ModelTypeOllama      ModelType = "ollama"
	ModelTypeOpenAI      ModelType = "openai"
	ModelTypeHuggingFace ModelType = "huggingface"
	ModelTypeMock        ModelType = "mock"
)

// ModelCapability represents a capability of an AI model
type ModelCapability string

const (
	CapabilityEmbedding  ModelCapability = "embedding"
	CapabilityCompletion ModelCapability = "completion"
	CapabilityChat       ModelCapability = "chat"
	CapabilityCodeGen    ModelCapability = "codegen"
	CapabilityVision     ModelCapability = "vision"
)

// ModelInfo contains information about an AI model
type ModelInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         ModelType         `json:"type"`
	Capabilities []ModelCapability `json:"capabilities"`
	MaxTokens    int               `json:"max_tokens"`
	Dimensions   int               `json:"dimensions"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	IsLocal      bool              `json:"is_local"`
}

// GenerateOptions contains options for generating responses
type GenerateOptions struct {
	Temperature      float32           `json:"temperature"`
	MaxTokens        int               `json:"max_tokens"`
	TopP             float32           `json:"top_p"`
	FrequencyPenalty float32           `json:"frequency_penalty"`
	PresencePenalty  float32           `json:"presence_penalty"`
	Stop             []string          `json:"stop"`
	Metadata         map[string]string `json:"metadata"`
}

// DefaultGenerateOptions returns default generation options
func DefaultGenerateOptions() GenerateOptions {
	return GenerateOptions{
		Temperature:      0.7,
		MaxTokens:        1024,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Stop:             []string{},
		Metadata:         map[string]string{},
	}
}

// Message represents a message in a chat conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamCallback is called for each token/chunk during streaming
type StreamCallback func(chunk string) error

// AIModel is the interface that all AI models must implement
type AIModel interface {
	// GenerateEmbedding generates an embedding for the given text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateResponse generates a response for the given prompt
	GenerateResponse(ctx context.Context, prompt string, options GenerateOptions) (string, error)

	// GenerateChatResponse generates a response for a chat conversation
	GenerateChatResponse(ctx context.Context, messages []Message, options GenerateOptions) (string, error)

	// GetModelInfo returns information about the model
	GetModelInfo() ModelInfo

	// IsHealthy checks if the model is healthy and available
	IsHealthy(ctx context.Context) bool
}

// StreamingAIModel represents an AI model that supports streaming responses
type StreamingAIModel interface {
	AIModel
	// GenerateResponseStream generates a streaming response for the given prompt
	GenerateResponseStream(ctx context.Context, prompt string, options GenerateOptions, callback StreamCallback) error
	// GenerateChatResponseStream generates a streaming response for a chat conversation
	GenerateChatResponseStream(ctx context.Context, messages []Message, options GenerateOptions, callback StreamCallback) error
}

// ModelManager manages multiple AI models
type ModelManager struct {
	Models map[string]AIModel
}

// NewModelManager creates a new model manager
func NewModelManager() *ModelManager {
	return &ModelManager{
		Models: make(map[string]AIModel),
	}
}

// RegisterModel registers a model with the manager
func (m *ModelManager) RegisterModel(model AIModel) {
	info := model.GetModelInfo()
	m.Models[info.ID] = model
}

// GetModel returns a model by ID
func (m *ModelManager) GetModel(id string) (AIModel, error) {
	model, exists := m.Models[id]
	if !exists {
		return nil, ErrModelNotFound
	}
	return model, nil
}

// ListModels returns a list of all registered models
func (m *ModelManager) ListModels() []ModelInfo {
	models := make([]ModelInfo, 0, len(m.Models))
	for _, model := range m.Models {
		models = append(models, model.GetModelInfo())
	}
	return models
}

// HTTPClient is an interface for HTTP clients to allow for mocking in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RetryConfig contains configuration for retry logic
type RetryConfig struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:  3,
		InitialWait: 500 * time.Millisecond,
		MaxWait:     5 * time.Second,
	}
}

// doWithRetry performs an HTTP request with retry logic
func doWithRetry(client HTTPClient, req *http.Request, retryConfig RetryConfig) (*http.Response, error) {
	var resp *http.Response
	var err error
	wait := retryConfig.InitialWait

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Clone the request to ensure it can be sent again
			// Ensure GetBody is set on the original request if the body needs to be resent.
			clonedReq := req.Clone(req.Context())
			if req.GetBody != nil {
				clonedReq.Body, err = req.GetBody()
				if err != nil {
					// If GetBody fails, we can't retry with body.
					// This might happen if the original request body was already consumed and GetBody was not set.
					return nil, fmt.Errorf("failed to get request body for retry: %w", err)
				}
			} else if req.Body != nil {
				// If GetBody is not set, but there was a body, we cannot reliably retry POST/PUT.
				// For GET/HEAD/DELETE, req.Body is nil, so this is fine.
				// For POST/PUT without GetBody, retrying is problematic.
				// The http.Request.Clone method handles this by setting Body to nil if GetBody is not set.
				// This is a simplification; robust body handling for retries needs GetBody.
			}

			req = clonedReq // Use the cloned request for the retry attempt
			time.Sleep(wait)
			// Exponential backoff with jitter
			wait = time.Duration(float64(wait) * 1.5) // Basic exponential backoff
			if wait > retryConfig.MaxWait {
				wait = retryConfig.MaxWait
			}
		}

		resp, err = client.Do(req)

		// Don't retry on context cancellation or deadline exceeded
		if err != nil {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err() // Context was cancelled or timed out
			default:
				// For other errors (e.g., network issues), continue to retry
				if attempt == retryConfig.MaxRetries {
					return nil, err // Return last error if max retries reached
				}
				continue
			}
		}

		// Don't retry on success or certain non-retryable status codes
		// Retry on 5xx server errors and 429 Too Many Requests.
		if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil // Success or non-retryable client error
		}

		// If it's a retryable status code and we haven't exhausted retries,
		// ensure the response body is closed before the next attempt.
		if attempt < retryConfig.MaxRetries {
			if resp.Body != nil {
				// Drain and close the body to allow connection reuse
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}

	return resp, err // Return the last response or error after all retries
}

// DoWithRetryTesting exposes doWithRetry for testing purposes
func DoWithRetryTesting(client HTTPClient, req *http.Request, retryConfig RetryConfig) (*http.Response, error) {
	return doWithRetry(client, req, retryConfig)
}

// OllamaConfig contains configuration for Ollama models
type OllamaConfig struct {
	BaseURL     string
	Timeout     time.Duration
	RetryConfig RetryConfig
}

// DefaultOllamaConfig returns a default Ollama configuration
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		BaseURL:     "http://localhost:11434",
		Timeout:     30 * time.Second,
		RetryConfig: DefaultRetryConfig(),
	}
}

// OllamaModel implements the AIModel interface for Ollama
type OllamaModel struct {
	Config    OllamaConfig
	ModelName string
	Info      ModelInfo
	Client    HTTPClient
}

// NewOllamaModel creates a new Ollama model
func NewOllamaModel(modelName string, config OllamaConfig) *OllamaModel {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Default model info
	info := ModelInfo{
		ID:           fmt.Sprintf("ollama-%s", modelName),
		Name:         modelName,
		Type:         ModelTypeOllama,
		Capabilities: []ModelCapability{CapabilityCompletion, CapabilityChat},
		MaxTokens:    4096,
		Dimensions:   384, // For nomic-embed-text
		Version:      "latest",
		Description:  fmt.Sprintf("Ollama model: %s", modelName),
		IsLocal:      true,
	}

	// Add code generation capability for codellama models
	if contains(modelName, "codellama") || contains(modelName, "code-llama") {
		info.Capabilities = append(info.Capabilities, CapabilityCodeGen)
	}
	if modelName == "nomic-embed-text" { // nomic-embed-text is primarily for embeddings
		info.Capabilities = []ModelCapability{CapabilityEmbedding}
		// Dimensions for nomic-embed-text can vary, but a common default is 768.
		// However, the example used 384. Let's stick to the existing value unless specified.
	}

	return &OllamaModel{
		Config:    config,
		ModelName: modelName,
		Info:      info,
		Client:    client,
	}
}

// ollamaEmbedRequest represents a request to the Ollama embedding API
type ollamaEmbedRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// ollamaEmbedResponse represents a response from the Ollama embedding API
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// ollamaGenerateRequest represents a request to the Ollama generation API
type ollamaGenerateRequest struct {
	Model    string                 `json:"model"`
	Prompt   string                 `json:"prompt,omitempty"` // Prompt is optional if Messages is used
	System   string                 `json:"system,omitempty"`
	Template string                 `json:"template,omitempty"`
	Context  []int                  `json:"context,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Stream   bool                   `json:"stream"`
	Raw      bool                   `json:"raw,omitempty"`
	Format   string                 `json:"format,omitempty"`
	Messages []Message              `json:"messages,omitempty"`
}

// ollamaGenerateResponse represents a response from the Ollama generation API
type ollamaGenerateResponse struct {
	Model              string  `json:"model"`
	CreatedAt          string  `json:"created_at"`
	Response           string  `json:"response"` // For non-chat
	Message            Message `json:"message"`  // For /api/chat
	Context            []int   `json:"context,omitempty"`
	TotalDuration      int64   `json:"total_duration"`
	LoadDuration       int64   `json:"load_duration"`
	PromptEvalDuration int64   `json:"prompt_eval_duration"`
	EvalDuration       int64   `json:"eval_duration"`
	EvalCount          int     `json:"eval_count"`
	Done               bool    `json:"done"`
}

// GenerateEmbedding generates an embedding for the given text using Ollama
func (m *OllamaModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Use nomic-embed-text model for embeddings if not otherwise specified by ModelName
	embedModelName := m.ModelName
	isEmbeddingModel := false
	for _, cap := range m.Info.Capabilities {
		if cap == CapabilityEmbedding {
			isEmbeddingModel = true
			break
		}
	}
	if !isEmbeddingModel || !strings.Contains(embedModelName, "embed") { // Fallback or default
		embedModelName = "nomic-embed-text"
	}

	reqBody := ollamaEmbedRequest{
		Model:  embedModelName,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/embeddings", m.Config.BaseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Embedding, nil
}

// GenerateResponse generates a response for the given prompt using Ollama
func (m *OllamaModel) GenerateResponse(ctx context.Context, prompt string, options GenerateOptions) (string, error) {
	// Convert options to Ollama format
	ollamaOptions := map[string]interface{}{
		"temperature": options.Temperature,
		"top_p":       options.TopP,
		"num_predict": options.MaxTokens,
		// Ollama uses num_ctx for context window size, not directly set here
		// FrequencyPenalty and PresencePenalty are not standard Ollama options, map if needed
		"stop": options.Stop,
	}
	if options.FrequencyPenalty != 0 {
		ollamaOptions["frequency_penalty"] = options.FrequencyPenalty
	}
	if options.PresencePenalty != 0 {
		ollamaOptions["presence_penalty"] = options.PresencePenalty
	}

	reqBody := ollamaGenerateRequest{
		Model:   m.ModelName,
		Prompt:  prompt,
		Options: ollamaOptions,
		Stream:  false, // Not streaming for this method
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/generate", m.Config.BaseURL), // Use /api/generate for prompt-based
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var genResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return genResp.Response, nil
}

// GenerateChatResponse generates a response for a chat conversation using Ollama
func (m *OllamaModel) GenerateChatResponse(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	// Convert options to Ollama format
	ollamaOptions := map[string]interface{}{
		"temperature": options.Temperature,
		"top_p":       options.TopP,
		"num_predict": options.MaxTokens,
		"stop":        options.Stop,
	}
	if options.FrequencyPenalty != 0 {
		ollamaOptions["frequency_penalty"] = options.FrequencyPenalty
	}
	if options.PresencePenalty != 0 {
		ollamaOptions["presence_penalty"] = options.PresencePenalty
	}

	reqBody := ollamaGenerateRequest{
		Model:    m.ModelName,
		Messages: messages,
		Options:  ollamaOptions,
		Stream:   false, // Not streaming for this method
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/chat", m.Config.BaseURL), // Use /api/chat for message-based
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var genResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	// For /api/chat, the response is in genResp.Message.Content
	return genResp.Message.Content, nil
}

// GetModelInfo returns information about the model
func (m *OllamaModel) GetModelInfo() ModelInfo {
	return m.Info
}

// IsHealthy checks if the model is healthy and available
func (m *OllamaModel) IsHealthy(ctx context.Context) bool {
	// A more specific check might be to try /api/show for the specific model
	// but /api/tags is a general health check for the Ollama service.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/tags", m.Config.BaseURL), // Checks if Ollama service is up
		nil,
	)
	if err != nil {
		return false
	}

	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // Short timeout for health check
	defer cancel()
	req = req.WithContext(healthCtx)

	resp, err := m.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GenerateResponseStream generates a streaming response for the given prompt
func (m *OllamaModel) GenerateResponseStream(ctx context.Context, prompt string, options GenerateOptions, callback StreamCallback) error {
	// Convert options to Ollama format
	ollamaOptions := map[string]interface{}{
		"temperature": options.Temperature,
		"top_p":       options.TopP,
		"num_predict": options.MaxTokens,
		"stop":        options.Stop,
	}

	reqBody := ollamaGenerateRequest{
		Model:   m.ModelName,
		Prompt:  prompt,
		Options: ollamaOptions,
		Stream:  true, // Enable streaming
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/generate", m.Config.BaseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Read streaming response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var genResp ollamaGenerateResponse
		if err := json.Unmarshal([]byte(line), &genResp); err != nil {
			continue // Skip malformed lines
		}

		// Call callback with the response chunk
		if genResp.Response != "" {
			if err := callback(genResp.Response); err != nil {
				return err
			}
		}

		// If done, break the loop
		if genResp.Done {
			break
		}
	}

	return scanner.Err()
}

// GenerateChatResponseStream generates a streaming response for a chat conversation
func (m *OllamaModel) GenerateChatResponseStream(ctx context.Context, messages []Message, options GenerateOptions, callback StreamCallback) error {
	// Convert options to Ollama format
	ollamaOptions := map[string]interface{}{
		"temperature": options.Temperature,
		"top_p":       options.TopP,
		"num_predict": options.MaxTokens,
		"stop":        options.Stop,
	}

	reqBody := ollamaGenerateRequest{
		Model:    m.ModelName,
		Messages: messages,
		Options:  ollamaOptions,
		Stream:   true, // Enable streaming
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/chat", m.Config.BaseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Read streaming response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var genResp ollamaGenerateResponse
		if err := json.Unmarshal([]byte(line), &genResp); err != nil {
			continue // Skip malformed lines
		}

		// Call callback with the response chunk
		if genResp.Message.Content != "" {
			if err := callback(genResp.Message.Content); err != nil {
				return err
			}
		}

		// If done, break the loop
		if genResp.Done {
			break
		}
	}

	return scanner.Err()
}

// OpenAIConfig contains configuration for OpenAI models
type OpenAIConfig struct {
	APIKey      string
	BaseURL     string
	Timeout     time.Duration
	RetryConfig RetryConfig
}

// DefaultOpenAIConfig returns a default OpenAI configuration
func DefaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		APIKey:      "",
		BaseURL:     "https://api.openai.com/v1",
		Timeout:     60 * time.Second,
		RetryConfig: DefaultRetryConfig(),
	}
}

// OpenAIModel implements the AIModel interface for OpenAI
type OpenAIModel struct {
	Config    OpenAIConfig
	ModelName string
	Info      ModelInfo
	Client    HTTPClient
}

// NewOpenAIModel creates a new OpenAI model
func NewOpenAIModel(modelName string, config OpenAIConfig) (*OpenAIModel, error) {
	if config.APIKey == "" {
		return nil, ErrAPIKeyNotSet
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Default model info
	info := ModelInfo{
		ID:           fmt.Sprintf("openai-%s", modelName),
		Name:         modelName,
		Type:         ModelTypeOpenAI,
		Capabilities: []ModelCapability{CapabilityCompletion, CapabilityChat}, // Most chat models also do completion
		MaxTokens:    4096,                                                    // Common default, adjust per model
		Dimensions:   1536,                                                    // For text-embedding-ada-002
		Version:      "latest",                                                // Or specific version if known
		Description:  fmt.Sprintf("OpenAI model: %s", modelName),
		IsLocal:      false,
	}

	// Add capabilities based on model name
	// This is illustrative; a more robust way might be to query the /v1/models endpoint
	if strings.HasPrefix(modelName, "gpt-4") {
		info.MaxTokens = 8192 // Default for gpt-4, some versions have more (e.g. gpt-4-32k)
		if strings.Contains(modelName, "vision") {
			info.Capabilities = append(info.Capabilities, CapabilityVision)
			info.MaxTokens = 128000 // gpt-4-vision-preview
		}
		if strings.Contains(modelName, "turbo") { // gpt-4-turbo
			info.MaxTokens = 128000
		}
	} else if strings.HasPrefix(modelName, "gpt-3.5-turbo") {
		info.MaxTokens = 4096 // Or 16385 for gpt-3.5-turbo-16k
		if strings.Contains(modelName, "16k") {
			info.MaxTokens = 16385
		}
	}

	// Add code generation capability for codex models (if they were still primary)
	// GPT models are generally good at code too.
	if strings.Contains(modelName, "code") || strings.Contains(modelName, "codex") { // "codex" is legacy
		if !containsCapability(info.Capabilities, CapabilityCodeGen) {
			info.Capabilities = append(info.Capabilities, CapabilityCodeGen)
		}
	}

	// Adjust for specific embedding models
	if modelName == "text-embedding-ada-002" || modelName == "text-embedding-3-small" || modelName == "text-embedding-3-large" {
		info.Capabilities = []ModelCapability{CapabilityEmbedding}
		if modelName == "text-embedding-ada-002" {
			info.Dimensions = 1536
		} else if modelName == "text-embedding-3-small" {
			info.Dimensions = 1536 // Can be 512 too
		} else if modelName == "text-embedding-3-large" {
			info.Dimensions = 3072 // Can be 256 or 1024 too
		}
	}

	return &OpenAIModel{
		Config:    config,
		ModelName: modelName,
		Info:      info,
		Client:    client,
	}, nil
}

// openAIEmbedRequest represents a request to the OpenAI embedding API
type openAIEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	// Optional: encoding_format, dimensions (for newer models)
	EncodingFormat string `json:"encoding_format,omitempty"`
	Dimensions     int    `json:"dimensions,omitempty"`
}

// openAIEmbedResponse represents a response from the OpenAI embedding API
type openAIEmbedResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// openAIChatRequest represents a request to the OpenAI chat API
type openAIChatRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	Temperature      float32   `json:"temperature,omitempty"` // Omitempty if it's the default 1.0
	MaxTokens        int       `json:"max_tokens,omitempty"`
	TopP             float32   `json:"top_p,omitempty"`
	FrequencyPenalty float32   `json:"frequency_penalty,omitempty"`
	PresencePenalty  float32   `json:"presence_penalty,omitempty"`
	Stop             []string  `json:"stop,omitempty"`
	// Add other params like response_format, seed, etc. if needed
}

// openAIChatResponse represents a response from the OpenAI chat API
type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
		// Logprobs *Logprobs `json:"logprobs,omitempty"` // If logprobs are requested
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	// SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// GenerateEmbedding generates an embedding for the given text using OpenAI
func (m *OpenAIModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedModelName := m.ModelName
	// Ensure we are using an actual embedding model ID
	if !strings.Contains(embedModelName, "embed") {
		embedModelName = "text-embedding-ada-002" // Fallback to a known embedding model
	}

	reqBody := openAIEmbedRequest{
		Model: embedModelName,
		Input: []string{text},
	}
	// For newer models like text-embedding-3-small/large, dimensions can be specified
	// if m.Info.Dimensions > 0 && (embedModelName == "text-embedding-3-small" || embedModelName == "text-embedding-3-large") {
	//    reqBody.Dimensions = m.Info.Dimensions
	// }

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/embeddings", m.Config.BaseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.Config.APIKey))

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for specific OpenAI error structure if possible
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var embedResp openAIEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, errors.New("no embedding data in response")
	}

	return embedResp.Data[0].Embedding, nil
}

// GenerateResponse generates a response for the given prompt using OpenAI
// For OpenAI, this typically means using the chat completions endpoint with a user message.
func (m *OpenAIModel) GenerateResponse(ctx context.Context, prompt string, options GenerateOptions) (string, error) {
	messages := []Message{
		{Role: "user", Content: prompt},
	}
	// If a system prompt is desired for "completion" style, it could be added here.
	// e.g., {Role: "system", Content: "You are a helpful assistant."}

	return m.GenerateChatResponse(ctx, messages, options)
}

// GenerateChatResponse generates a response for a chat conversation using OpenAI
func (m *OpenAIModel) GenerateChatResponse(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	reqBody := openAIChatRequest{
		Model:            m.ModelName,
		Messages:         messages,
		Temperature:      options.Temperature,
		MaxTokens:        options.MaxTokens,
		TopP:             options.TopP,
		FrequencyPenalty: options.FrequencyPenalty,
		PresencePenalty:  options.PresencePenalty,
		Stop:             options.Stop,
	}
	if options.MaxTokens == 0 { // If MaxTokens is 0, OpenAI might use model's default or error. Be explicit.
		reqBody.MaxTokens = m.Info.MaxTokens // Use model's configured max tokens if not specified
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/chat/completions", m.Config.BaseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.Config.APIKey))

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", errors.New("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GetModelInfo returns information about the model
func (m *OpenAIModel) GetModelInfo() ModelInfo {
	return m.Info
}

// IsHealthy checks if the model is healthy and available
func (m *OpenAIModel) IsHealthy(ctx context.Context) bool {
	// OpenAI doesn't have a simple health check endpoint.
	// A common way is to try to retrieve the specific model's info.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/models/%s", m.Config.BaseURL, m.ModelName),
		nil,
	)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.Config.APIKey))

	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Short timeout for health check
	defer cancel()
	req = req.WithContext(healthCtx)

	resp, err := m.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// HuggingFaceConfig contains configuration for HuggingFace models
type HuggingFaceConfig struct {
	APIKey      string
	BaseURL     string // Base URL for HuggingFace Inference API
	Timeout     time.Duration
	RetryConfig RetryConfig
}

// DefaultHuggingFaceConfig returns a default HuggingFace configuration
func DefaultHuggingFaceConfig() HuggingFaceConfig {
	return HuggingFaceConfig{
		APIKey:      "",
		BaseURL:     "https://api-inference.huggingface.co/models",
		Timeout:     60 * time.Second,
		RetryConfig: DefaultRetryConfig(),
	}
}

// HuggingFaceModel implements the AIModel interface for HuggingFace
type HuggingFaceModel struct {
	Config    HuggingFaceConfig
	ModelName string // This is the model ID on HuggingFace, e.g., "gpt2" or "sentence-transformers/all-MiniLM-L6-v2"
	Info      ModelInfo
	Client    HTTPClient
}

// NewHuggingFaceModel creates a new HuggingFace model
func NewHuggingFaceModel(modelName string, config HuggingFaceConfig) (*HuggingFaceModel, error) {
	if config.APIKey == "" {
		// Some public models might work without API key, but it's generally recommended.
		// For this implementation, let's require it for simplicity or specific tasks.
		// return nil, ErrAPIKeyNotSet // Or allow if model is known public
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Default model info - this is very generic and should ideally be fetched or configured
	info := ModelInfo{
		ID:           fmt.Sprintf("hf-%s", strings.ReplaceAll(modelName, "/", "-")), // Sanitize ID
		Name:         modelName,
		Type:         ModelTypeHuggingFace,
		Capabilities: []ModelCapability{CapabilityCompletion}, // Default, can be overridden
		MaxTokens:    2048,                                    // Generic default
		Dimensions:   768,                                     // Common for BERT-like, ST models
		Version:      "latest",
		Description:  fmt.Sprintf("HuggingFace model: %s", modelName),
		IsLocal:      false,
	}

	// Add capabilities based on model name heuristics (very basic)
	if strings.Contains(modelName, "sentence-transformers") || strings.Contains(modelName, "embedding") {
		info.Capabilities = []ModelCapability{CapabilityEmbedding}
	}
	if strings.Contains(modelName, "gpt") || strings.Contains(modelName, "llama") || strings.Contains(modelName, "mistral") || strings.Contains(modelName, "falcon") || strings.Contains(modelName, "phi") {
		if !containsCapability(info.Capabilities, CapabilityChat) { // Avoid duplicates if already set
			info.Capabilities = append(info.Capabilities, CapabilityChat)
		}
		if !containsCapability(info.Capabilities, CapabilityCompletion) {
			info.Capabilities = append(info.Capabilities, CapabilityCompletion)
		}
	}
	if strings.Contains(modelName, "code") || strings.Contains(modelName, "starcoder") || strings.Contains(modelName, "codegen") {
		if !containsCapability(info.Capabilities, CapabilityCodeGen) {
			info.Capabilities = append(info.Capabilities, CapabilityCodeGen)
		}
	}

	return &HuggingFaceModel{
		Config:    config,
		ModelName: modelName,
		Info:      info,
		Client:    client,
	}, nil
}

// hfFeatureExtractionRequest for sentence-transformers
type hfFeatureExtractionRequest struct {
	Inputs  []string               `json:"inputs"`
	Options map[string]interface{} `json:"options,omitempty"` // e.g., {"wait_for_model": true}
}

// hfTextGenerationRequest for text generation models
type hfTextGenerationRequest struct {
	Inputs     string                 `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"` // e.g., max_length, temperature
	Options    map[string]interface{} `json:"options,omitempty"`    // e.g., {"wait_for_model": true}
}

// hfConversationalRequest for conversational models
type hfConversationalRequest struct {
	Inputs struct {
		PastUserInputs     []string `json:"past_user_inputs,omitempty"`
		GeneratedResponses []string `json:"generated_responses,omitempty"`
		Text               string   `json:"text"` // Current user input
	} `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// hfEmbeddingResponse can be [][]float32 (for sentence transformers) or other structures.
// For sentence-transformers, it's typically [][]float32.
type hfEmbeddingResponse = [][]float32 // Alias for clarity

// hfTextGenerationResponse is typically an array of objects with "generated_text".
type hfTextGenerationResponse []struct {
	GeneratedText string `json:"generated_text"`
}

// hfConversationalResponse for conversational models
type hfConversationalResponse struct {
	GeneratedText string `json:"generated_text"`
	Conversation  struct {
		PastUserInputs     []string `json:"past_user_inputs"`
		GeneratedResponses []string `json:"generated_responses"`
	} `json:"conversation,omitempty"`
	// Warnings []string `json:"warnings,omitempty"`
}

// GenerateEmbedding generates an embedding for the given text using HuggingFace
func (m *HuggingFaceModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if !containsCapability(m.Info.Capabilities, CapabilityEmbedding) {
		return nil, fmt.Errorf("model %s does not support embedding: %w", m.ModelName, ErrUnsupportedModel)
	}

	reqBody := hfFeatureExtractionRequest{
		Inputs:  []string{text},
		Options: map[string]interface{}{"wait_for_model": true},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s", m.Config.BaseURL, m.ModelName),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if m.Config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.Config.APIKey))
	}

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var embedResp hfEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		// Try to read body for more debug info if JSON decoding fails
		// resp.Body is already consumed by json.NewDecoder. Need to re-read or handle carefully.
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp) == 0 || len(embedResp[0]) == 0 {
		return nil, errors.New("no embedding data in response")
	}

	return embedResp[0], nil
}

// GenerateResponse generates a response for the given prompt using HuggingFace
func (m *HuggingFaceModel) GenerateResponse(ctx context.Context, prompt string, options GenerateOptions) (string, error) {
	if !containsCapability(m.Info.Capabilities, CapabilityCompletion) && !containsCapability(m.Info.Capabilities, CapabilityChat) {
		return "", fmt.Errorf("model %s does not support text generation: %w", m.ModelName, ErrUnsupportedModel)
	}

	// Convert options to HuggingFace text-generation parameters
	parameters := map[string]interface{}{
		"temperature":    options.Temperature,
		"max_new_tokens": options.MaxTokens, // HF uses max_new_tokens or max_length
		"top_p":          options.TopP,
		"do_sample":      options.Temperature > 0.01, // Enable sampling if temperature is not effectively zero
		// "repetition_penalty" could map to FrequencyPenalty/PresencePenalty
		// "stop_sequences": options.Stop, // HF uses "stop_sequences"
	}
	if options.MaxTokens == 0 {
		parameters["max_new_tokens"] = 250 // A reasonable default for HF if not set
	}
	if len(options.Stop) > 0 {
		parameters["stop_sequences"] = options.Stop
	}

	reqBody := hfTextGenerationRequest{
		Inputs:     prompt,
		Parameters: parameters,
		Options:    map[string]interface{}{"wait_for_model": true},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s", m.Config.BaseURL, m.ModelName),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if m.Config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.Config.APIKey))
	}

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var genResp hfTextGenerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(genResp) == 0 || genResp[0].GeneratedText == "" {
		// Sometimes HF returns an empty array or empty generated_text for certain models/inputs
		return "", errors.New("no generation data in response or empty generated_text")
	}

	return genResp[0].GeneratedText, nil
}

// GenerateChatResponse generates a response for a chat conversation using HuggingFace
func (m *HuggingFaceModel) GenerateChatResponse(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	if !containsCapability(m.Info.Capabilities, CapabilityChat) {
		// Fallback to completion if chat not explicitly supported but completion is
		if containsCapability(m.Info.Capabilities, CapabilityCompletion) {
			// Convert messages to a single prompt string
			var promptBuilder strings.Builder
			for _, msg := range messages {
				promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
			}
			promptBuilder.WriteString("assistant:") // Prompt for assistant's turn
			return m.GenerateResponse(ctx, promptBuilder.String(), options)
		}
		return "", fmt.Errorf("model %s does not support chat: %w", m.ModelName, ErrUnsupportedModel)
	}

	// Using HuggingFace's "conversational" task structure
	var pastUserInputs []string
	var generatedResponses []string
	var currentInput string

	if len(messages) > 0 {
		// Iterate up to the second to last message to populate history
		for i := 0; i < len(messages)-1; i++ {
			if messages[i].Role == "user" {
				pastUserInputs = append(pastUserInputs, messages[i].Content)
			} else if messages[i].Role == "assistant" {
				generatedResponses = append(generatedResponses, messages[i].Content)
			}
		}
		// The last message is the current user input
		if messages[len(messages)-1].Role == "user" {
			currentInput = messages[len(messages)-1].Content
		} else {
			// This case (last message from assistant) is unusual for a request,
			// but handle defensively.
			currentInput = "" // Or perhaps the last user message if available.
			if len(pastUserInputs) > 0 {
				currentInput = pastUserInputs[len(pastUserInputs)-1]    // Re-use last user input
				pastUserInputs = pastUserInputs[:len(pastUserInputs)-1] // Pop it from history
			}
		}
	}

	parameters := map[string]interface{}{
		"temperature":    options.Temperature,
		"max_new_tokens": options.MaxTokens,
		"top_p":          options.TopP,
		// Other parameters like repetition_penalty, top_k can be added.
	}
	if options.MaxTokens == 0 {
		parameters["max_new_tokens"] = 250 // Default if not set
	}
	if len(options.Stop) > 0 {
		parameters["stop_sequences"] = options.Stop
	}

	reqBody := hfConversationalRequest{
		Inputs: struct {
			PastUserInputs     []string `json:"past_user_inputs,omitempty"`
			GeneratedResponses []string `json:"generated_responses,omitempty"`
			Text               string   `json:"text"`
		}{
			PastUserInputs:     pastUserInputs,
			GeneratedResponses: generatedResponses,
			Text:               currentInput,
		},
		Parameters: parameters,
		Options:    map[string]interface{}{"wait_for_model": true},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/%s", m.Config.BaseURL, m.ModelName),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if m.Config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.Config.APIKey))
	}

	resp, err := doWithRetry(m.Client, req, m.Config.RetryConfig)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var chatResp hfConversationalResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.GeneratedText, nil
}

// GetModelInfo returns information about the model
func (m *HuggingFaceModel) GetModelInfo() ModelInfo {
	return m.Info
}

// IsHealthy checks if the model is healthy and available using the Inference API.
// This typically means the model exists and the API is responsive.
func (m *HuggingFaceModel) IsHealthy(ctx context.Context) bool {
	// A simple way to check is to try to get the model's page or send a minimal request.
	// The Inference API itself doesn't have a generic /health endpoint for specific models.
	// We can try a GET request to the model's endpoint; it might return info or error.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet, // Some models might respond to GET with info, others might expect POST.
		fmt.Sprintf("%s/%s", m.Config.BaseURL, m.ModelName),
		nil,
	)
	if err != nil {
		return false
	}

	if m.Config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.Config.APIKey))
	}

	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Short timeout for health check
	defer cancel()
	req = req.WithContext(healthCtx)

	resp, err := m.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// For HF, a 200 OK on GET might mean the model page is found.
	// A POST with empty/minimal valid payload might be a better check if GET isn't standard.
	// If the model is loading, it might return 503, which doWithRetry would handle.
	// For simplicity, 200 OK on GET is a basic check.
	return resp.StatusCode == http.StatusOK
}

// MockModel is a mock implementation of the AIModel interface for testing
type MockModel struct {
	ID         string
	Dimensions int
}

// NewMockModel creates a new mock model
func NewMockModel(id string, dimensions int) *MockModel {
	return &MockModel{
		ID:         id,
		Dimensions: dimensions,
	}
}

// GenerateEmbedding generates a mock embedding for the given text
func (m *MockModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// This is a mock implementation that returns a simple pattern
	embedding := make([]float32, m.Dimensions)
	for i := 0; i < m.Dimensions; i++ {
		embedding[i] = float32(i) / float32(m.Dimensions) // Ensure float division
	}
	return embedding, nil
}

// GenerateResponse generates a mock response for the given prompt
func (m *MockModel) GenerateResponse(ctx context.Context, prompt string, options GenerateOptions) (string, error) {
	return "Response to: " + prompt, nil
}

// GenerateChatResponse generates a mock response for a chat conversation
func (m *MockModel) GenerateChatResponse(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	// Create a simple concatenation of the messages
	var lastMessageContent string
	if len(messages) > 0 {
		lastMessageContent = messages[len(messages)-1].Content
	}
	return "Response to chat: " + lastMessageContent, nil
}

// GetModelInfo returns information about the mock model
func (m *MockModel) GetModelInfo() ModelInfo {
	return ModelInfo{
		ID:           m.ID,
		Name:         "mock-model", // Standardized name for mock
		Type:         ModelTypeMock,
		Capabilities: []ModelCapability{CapabilityEmbedding, CapabilityCompletion, CapabilityChat},
		MaxTokens:    4096,
		Dimensions:   m.Dimensions,
		Version:      "1.0",
		Description:  "Mock model for testing",
		IsLocal:      true,
	}
}

// IsHealthy always returns true for the mock model
func (m *MockModel) IsHealthy(ctx context.Context) bool {
	return true
}

// Utility functions

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// containsCapability checks if a slice of ModelCapability contains a specific capability.
func containsCapability(capabilities []ModelCapability, capability ModelCapability) bool {
	for _, cap := range capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}
