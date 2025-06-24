package ai_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelManager(t *testing.T) {
	t.Run("NewModelManager", func(t *testing.T) {
		manager := ai.NewModelManager()
		require.NotNil(t, manager, "NewModelManager should return a non-nil manager")
		assert.NotNil(t, manager.Models, "Manager.Models should be initialized")
		assert.Empty(t, manager.Models, "Newly created manager should have no models")
	})

	t.Run("RegisterAndGetModel", func(t *testing.T) {
		manager := ai.NewModelManager()
		mockModel1 := ai.NewMockModel("test-id-1", 128)
		mockModel2Info := ai.ModelInfo{ID: "test-id-2", Name: "Test Model 2", Type: ai.ModelTypeMock}
		mockModel2 := &mockAIModel{info: mockModel2Info} // Using a local mock for more control if needed

		manager.RegisterModel(mockModel1)
		manager.RegisterModel(mockModel2)

		// Test GetModel for existing models
		retrievedModel1, err := manager.GetModel("test-id-1")
		require.NoError(t, err, "GetModel for existing ID should not return an error")
		assert.Equal(t, mockModel1.GetModelInfo().ID, retrievedModel1.GetModelInfo().ID, "Retrieved model 1 ID mismatch")

		retrievedModel2, err := manager.GetModel("test-id-2")
		require.NoError(t, err, "GetModel for existing ID should not return an error")
		assert.Equal(t, mockModel2.GetModelInfo().ID, retrievedModel2.GetModelInfo().ID, "Retrieved model 2 ID mismatch")

		// Test GetModel for non-existent model
		_, err = manager.GetModel("non-existent-id")
		assert.Error(t, err, "GetModel for non-existent ID should return an error")
		assert.True(t, errors.Is(err, ai.ErrModelNotFound), "Error should be ErrModelNotFound")
	})

	t.Run("ListModels", func(t *testing.T) {
		manager := ai.NewModelManager()
		assert.Empty(t, manager.ListModels(), "ListModels on new manager should be empty")

		mockModel1 := ai.NewMockModel("list-id-1", 128)
		mockModel2 := ai.NewMockModel("list-id-2", 256)
		manager.RegisterModel(mockModel1)
		manager.RegisterModel(mockModel2)

		listedModels := manager.ListModels()
		require.Len(t, listedModels, 2, "ListModels should return all registered models")

		found1, found2 := false, false
		for _, modelInfo := range listedModels {
			if modelInfo.ID == "list-id-1" {
				found1 = true
			}
			if modelInfo.ID == "list-id-2" {
				found2 = true
			}
		}
		assert.True(t, found1, "Model list-id-1 not found in listed models")
		assert.True(t, found2, "Model list-id-2 not found in listed models")
	})
}

// mockAIModel is a local mock for testing purposes, if more control than ai.MockModel is needed.
type mockAIModel struct {
	info                 ai.ModelInfo
	generateResponseFunc func(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error)
	isHealthyFunc        func(ctx context.Context) bool
}

func (m *mockAIModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	dim := 384
	if m.info.Dimensions > 0 {
		dim = m.info.Dimensions
	}
	embedding := make([]float32, dim)
	for i := range embedding {
		embedding[i] = float32(i) * 0.01
	}
	return embedding, nil
}

func (m *mockAIModel) GenerateResponse(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error) {
	if m.generateResponseFunc != nil {
		return m.generateResponseFunc(ctx, prompt, options)
	}
	return "mock response from " + m.info.ID, nil
}

func (m *mockAIModel) GenerateChatResponse(ctx context.Context, messages []ai.Message, options ai.GenerateOptions) (string, error) {
	var lastContent string
	if len(messages) > 0 {
		lastContent = messages[len(messages)-1].Content
	}
	return "mock chat response to: " + lastContent, nil
}

func (m *mockAIModel) GetModelInfo() ai.ModelInfo {
	return m.info
}

func (m *mockAIModel) IsHealthy(ctx context.Context) bool {
	if m.isHealthyFunc != nil {
		return m.isHealthyFunc(ctx)
	}
	return true
}

func TestMockModel(t *testing.T) {
	mockID := "test-mock-model"
	mockDims := 128
	model := ai.NewMockModel(mockID, mockDims)
	require.NotNil(t, model)

	t.Run("GetModelInfo", func(t *testing.T) {
		info := model.GetModelInfo()
		assert.Equal(t, mockID, info.ID)
		assert.Equal(t, "mock-model", info.Name) // Name is hardcoded in MockModel
		assert.Equal(t, ai.ModelTypeMock, info.Type)
		assert.Equal(t, mockDims, info.Dimensions)
		assert.True(t, info.IsLocal)
		expectedCapabilities := []ai.ModelCapability{ai.CapabilityEmbedding, ai.CapabilityCompletion, ai.CapabilityChat}
		assert.ElementsMatch(t, expectedCapabilities, info.Capabilities)
	})

	t.Run("GenerateEmbedding", func(t *testing.T) {
		embedding, err := model.GenerateEmbedding(context.Background(), "test text")
		require.NoError(t, err)
		assert.Len(t, embedding, mockDims)
		// Check a few values to ensure it's populated as expected by MockModel
		assert.Equal(t, float32(0.0), embedding[0])
		if mockDims > 1 {
			assert.Equal(t, float32(1)/float32(mockDims), embedding[1])
		}
	})

	t.Run("GenerateResponse", func(t *testing.T) {
		prompt := "hello world"
		response, err := model.GenerateResponse(context.Background(), prompt, ai.DefaultGenerateOptions())
		require.NoError(t, err)
		assert.Equal(t, "Response to: "+prompt, response)
	})

	t.Run("GenerateChatResponse", func(t *testing.T) {
		messages := []ai.Message{
			{Role: "user", Content: "first message"},
			{Role: "assistant", Content: "first response"},
			{Role: "user", Content: "second message"},
		}
		response, err := model.GenerateChatResponse(context.Background(), messages, ai.DefaultGenerateOptions())
		require.NoError(t, err)
		assert.Equal(t, "Response to chat: second message", response)

		// Test with empty messages
		emptyMessages := []ai.Message{}
		responseEmpty, errEmpty := model.GenerateChatResponse(context.Background(), emptyMessages, ai.DefaultGenerateOptions())
		require.NoError(t, errEmpty)
		assert.Equal(t, "Response to chat: ", responseEmpty)

	})

	t.Run("IsHealthy", func(t *testing.T) {
		assert.True(t, model.IsHealthy(context.Background()), "MockModel should always be healthy")
	})
}

// mockHTTPClient is a mock implementation of ai.HTTPClient for testing doWithRetry.
type mockHTTPClient struct {
	DoFunc      func(req *http.Request) (*http.Response, error)
	callCount   int32
	lastRequest *http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	atomic.AddInt32(&m.callCount, 1)
	m.lastRequest = req.Clone(req.Context()) // Clone to inspect later
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("DoFunc not implemented in mockHTTPClient")
}

func (m *mockHTTPClient) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

func (m *mockHTTPClient) ResetCallCount() {
	atomic.StoreInt32(&m.callCount, 0)
}

func TestDoWithRetry(t *testing.T) {
	testRetryConfig := ai.RetryConfig{
		MaxRetries:  2,
		InitialWait: 5 * time.Millisecond, // Shorter waits for faster tests
		MaxWait:     20 * time.Millisecond,
	}

	createTestRequest := func(ctx context.Context) *http.Request {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/test", nil)
		return req
	}

	t.Run("SuccessFirstTry", func(t *testing.T) {
		client := &mockHTTPClient{}
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("OK")),
			}, nil
		}

		req := createTestRequest(context.Background())
		resp, err := ai.DoWithRetryTesting(client, req, testRetryConfig) // Using exported for testing
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 1, client.GetCallCount())
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		assert.Equal(t, "OK", string(bodyBytes))
	})

	t.Run("SuccessAfterRetries", func(t *testing.T) {
		client := &mockHTTPClient{}
		attempts := 0
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts <= 1 { // Fail first time
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("Server Error")),
				}, nil // Note: Ollama/OpenAI clients might return error here too
			}
			// Success on second attempt
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("OK")),
			}, nil
		}

		req := createTestRequest(context.Background())
		resp, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 2, client.GetCallCount()) // Original + 1 retry
	})

	t.Run("FailureMaxRetries", func(t *testing.T) {
		client := &mockHTTPClient{}
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("Persistent Server Error")),
			}, nil // Simulating server always erroring with a response
		}

		req := createTestRequest(context.Background())
		resp, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.NoError(t, err) // doWithRetry itself doesn't error if server responds, just returns last response
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Equal(t, testRetryConfig.MaxRetries+1, client.GetCallCount())
	})

	t.Run("FailureWithErrorMaxRetries", func(t *testing.T) {
		client := &mockHTTPClient{}
		expectedErr := errors.New("network connection failed")
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			return nil, expectedErr // Simulating network error, no response
		}

		req := createTestRequest(context.Background())
		_, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, testRetryConfig.MaxRetries+1, client.GetCallCount())
	})

	t.Run("FailureNonRetryableStatus", func(t *testing.T) {
		client := &mockHTTPClient{}
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest, // 400 should not be retried
				Body:       io.NopCloser(strings.NewReader("Bad Request")),
			}, nil
		}

		req := createTestRequest(context.Background())
		resp, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, 1, client.GetCallCount())
	})

	t.Run("RetryOn429", func(t *testing.T) {
		client := &mockHTTPClient{}
		attempts := 0
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests, // 429 should be retried
					Body:       io.NopCloser(strings.NewReader("Rate Limited")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("OK")),
			}, nil
		}

		req := createTestRequest(context.Background())
		resp, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 2, client.GetCallCount())
	})

	t.Run("FailureContextCancelledDuringRequest", func(t *testing.T) {
		client := &mockHTTPClient{}
		ctx, cancel := context.WithCancel(context.Background())

		waitChan := make(chan struct{})
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			// Simulate work, then check context
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(50 * time.Millisecond): // Simulate some work
				// continue
			}
			// If context not done yet, proceed to signal that DoFunc is "stuck"
			close(waitChan)        // Signal that DoFunc is now waiting
			<-req.Context().Done() // Wait for cancellation
			return nil, req.Context().Err()
		}

		req := createTestRequest(ctx)
		go func() {
			// Wait for DoFunc to be "stuck" or a short timeout
			select {
			case <-waitChan:
				// DoFunc is now in its "waiting for context done" phase
			case <-time.After(10 * time.Millisecond):
				// If DoFunc didn't signal quickly, proceed to cancel anyway
			}
			cancel() // Cancel the context while DoFunc is "running" or between retries
		}()

		_, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded), "Error should be context related")
		// Call count can be 1 if cancelled during the first attempt's Do call.
		assert.LessOrEqual(t, client.GetCallCount(), testRetryConfig.MaxRetries+1)
		assert.GreaterOrEqual(t, client.GetCallCount(), 1)
	})

	t.Run("FailureContextCancelledBetweenRetries", func(t *testing.T) {
		client := &mockHTTPClient{}
		ctx, cancel := context.WithCancel(context.Background())
		
		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			callCount := client.GetCallCount()
			if callCount == 1 {
				// First call - return retryable error
				return &http.Response{StatusCode: http.StatusInternalServerError}, nil
			}
			// Check if context is cancelled for subsequent calls
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			default:
				// If context not cancelled yet, this is unexpected but not a failure
				return &http.Response{StatusCode: http.StatusInternalServerError}, nil
			}
		}

		req := createTestRequest(ctx)
		
		// Cancel the context immediately after starting the retry process
		go func() {
			time.Sleep(testRetryConfig.InitialWait / 4) // Cancel very early in the retry sleep
			cancel()
		}()

		_, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.Error(t, err)
		
		// The context cancellation should be detected either:
		// 1. During the next client.Do call (via context check in DoFunc)
		// 2. Or after the call when doWithRetry checks context
		assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled"), 
			"Error should be context.Canceled or indicate it, got: %v", err)
		
		// We expect 1 or 2 calls depending on timing - 1 initial call, maybe 1 retry before cancellation is detected
		callCount := client.GetCallCount()
		assert.True(t, callCount <= 2, "Should not exceed 2 calls before cancellation, got: %d", callCount)
	})

	t.Run("RequestCloning", func(t *testing.T) {
		client := &mockHTTPClient{}
		originalBodyContent := "request body"
		var firstReqAddr, secondReqAddr uintptr // To store addresses of request objects

		client.DoFunc = func(req *http.Request) (*http.Response, error) {
			if client.GetCallCount() == 1 {
				firstReqAddr = uintptr(unsafe.Pointer(req)) // Get address of request
				bodyBytes, _ := io.ReadAll(req.Body)
				req.Body.Close()
				assert.Equal(t, originalBodyContent, string(bodyBytes), "Body of first request")
				// Restore body for potential next read if cloning didn't provide new one
				// req.Body = io.NopCloser(bytes.NewBufferString(originalBodyContent))
				return &http.Response{StatusCode: http.StatusInternalServerError}, nil
			}
			// Second attempt
			secondReqAddr = uintptr(unsafe.Pointer(req))
			bodyBytes, _ := io.ReadAll(req.Body)
			req.Body.Close()
			assert.Equal(t, originalBodyContent, string(bodyBytes), "Body of second request (cloned)")
			return &http.Response{StatusCode: http.StatusOK}, nil
		}

		ctx := context.Background()
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/test", io.NopCloser(bytes.NewBufferString(originalBodyContent)))

		// Set GetBody for the original request so it can be cloned
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBufferString(originalBodyContent)), nil
		}

		_, err := ai.DoWithRetryTesting(client, req, testRetryConfig)
		require.NoError(t, err)
		require.Equal(t, 2, client.GetCallCount())
		assert.NotEqual(t, firstReqAddr, secondReqAddr, "Request objects should be different across retries due to cloning")
	})
}

func TestDefaultConfigs(t *testing.T) {
	t.Run("DefaultRetryConfig", func(t *testing.T) {
		cfg := ai.DefaultRetryConfig()
		assert.Equal(t, 3, cfg.MaxRetries)
		assert.Equal(t, 500*time.Millisecond, cfg.InitialWait)
		assert.Equal(t, 5*time.Second, cfg.MaxWait)
	})

	t.Run("DefaultOllamaConfig", func(t *testing.T) {
		cfg := ai.DefaultOllamaConfig()
		assert.Equal(t, "http://localhost:11434", cfg.BaseURL)
		assert.Equal(t, 30*time.Second, cfg.Timeout)
		assert.Equal(t, ai.DefaultRetryConfig(), cfg.RetryConfig)
	})

	t.Run("DefaultOpenAIConfig", func(t *testing.T) {
		cfg := ai.DefaultOpenAIConfig()
		assert.Equal(t, "", cfg.APIKey)
		assert.Equal(t, "https://api.openai.com/v1", cfg.BaseURL)
		assert.Equal(t, 60*time.Second, cfg.Timeout)
		assert.Equal(t, ai.DefaultRetryConfig(), cfg.RetryConfig)
	})

	t.Run("DefaultHuggingFaceConfig", func(t *testing.T) {
		cfg := ai.DefaultHuggingFaceConfig()
		assert.Equal(t, "", cfg.APIKey)
		assert.Equal(t, "https://api-inference.huggingface.co/models", cfg.BaseURL)
		assert.Equal(t, 60*time.Second, cfg.Timeout)
		assert.Equal(t, ai.DefaultRetryConfig(), cfg.RetryConfig)
	})

	t.Run("DefaultGenerateOptions", func(t *testing.T) {
		opts := ai.DefaultGenerateOptions()
		assert.Equal(t, float32(0.7), opts.Temperature)
		assert.Equal(t, 1024, opts.MaxTokens)
		assert.Equal(t, float32(1.0), opts.TopP)
		assert.Empty(t, opts.Stop)
		assert.NotNil(t, opts.Metadata)
	})
}

func TestErrorConstants(t *testing.T) {
	// This test is mostly to ensure the error variables are exported and non-nil.
	assert.NotNil(t, ai.ErrModelNotFound)
	assert.NotNil(t, ai.ErrInvalidRequest)
	assert.NotNil(t, ai.ErrUnsupportedModel)
	assert.NotNil(t, ai.ErrAPIKeyNotSet)
	assert.NotNil(t, ai.ErrContextExceeded)
	assert.NotNil(t, ai.ErrServiceUnavailable)
	assert.NotNil(t, ai.ErrTimeout)
	assert.NotNil(t, ai.ErrRateLimited)

	assert.Equal(t, "model not found", ai.ErrModelNotFound.Error())
}

// Note: Testing concrete model implementations (OllamaModel, OpenAIModel, HuggingFaceModel)
// would typically require actual services or more complex mocking of their HTTP interactions.
// These tests are focused on ModelManager, MockModel, and shared utilities like doWithRetry.
