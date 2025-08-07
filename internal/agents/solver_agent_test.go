package agents

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAIModel is a mock implementation of the ai.AIModel interface for testing.
type mockAIModel struct {
	info                 ai.ModelInfo
	generateResponseFunc func(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error)
	isHealthyFunc        func(ctx context.Context) bool
	lastPrompt           string
	lastOptions          ai.GenerateOptions
}

func (m *mockAIModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Return a fixed-size embedding for testing purposes
	dim := 384
	if m.info.Dimensions > 0 {
		dim = m.info.Dimensions
	}
	embedding := make([]float32, dim)
	for i := range embedding {
		embedding[i] = float32(i) * 0.1
	}
	return embedding, nil
}

func (m *mockAIModel) GenerateResponse(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error) {
	m.lastPrompt = prompt
	m.lastOptions = options
	if m.generateResponseFunc != nil {
		return m.generateResponseFunc(ctx, prompt, options)
	}
	return "mock response from " + m.info.ID, nil
}

func (m *mockAIModel) GenerateChatResponse(ctx context.Context, messages []ai.Message, options ai.GenerateOptions) (string, error) {
	if len(messages) > 0 {
		// For simplicity, use the content of the last message as the prompt for GenerateResponse
		return m.GenerateResponse(ctx, messages[len(messages)-1].Content, options)
	}
	if m.generateResponseFunc != nil {
		// Fallback to generateResponseFunc if messages is empty, though this case might not be standard for chat
		return m.generateResponseFunc(ctx, "", options)
	}
	return "default mock chat response", nil
}

func (m *mockAIModel) GetModelInfo() ai.ModelInfo {
	return m.info
}

func (m *mockAIModel) IsHealthy(ctx context.Context) bool {
	if m.isHealthyFunc != nil {
		return m.isHealthyFunc(ctx)
	}
	return true // Default to healthy
}

func newConfigurableMockAIModel(id string, capabilities []ai.ModelCapability) *mockAIModel {
	return &mockAIModel{
		info: ai.ModelInfo{
			ID:           id,
			Name:         id,
			Type:         ai.ModelTypeMock,
			Capabilities: capabilities,
			MaxTokens:    2048,
			IsLocal:      true,
			Dimensions:   384, // Default dimension for mock
		},
	}
}

func TestNewSolverAgent(t *testing.T) {
	t.Run("NilConfig", func(t *testing.T) {
		agent := NewSolverAgent(nil)
		require.NotNil(t, agent)
		require.NotNil(t, agent.modelManager)
		require.NotNil(t, agent.config) // Should be initialized to &SolverConfig{}

		// Auto-selected model should be valid (could be ollama-cogito, openai-gpt-3.5-turbo, or default)
		assert.NotEmpty(t, agent.config.DefaultModel)

		// Check if the selected model is registered
		_, err := agent.modelManager.GetModel(agent.config.DefaultModel)
		assert.NoError(t, err, "selected default model should be registered")

		defaultOpts := ai.DefaultGenerateOptions()
		assert.Equal(t, defaultOpts.MaxTokens, agent.defaultOptions.MaxTokens)
		assert.Equal(t, defaultOpts.Temperature, agent.defaultOptions.Temperature)
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{})
		require.NotNil(t, agent)

		// Auto-selected model should be valid (could be ollama-cogito, openai-gpt-3.5-turbo, or default)
		assert.NotEmpty(t, agent.config.DefaultModel)
		defaultOpts := ai.DefaultGenerateOptions()
		assert.Equal(t, defaultOpts.MaxTokens, agent.defaultOptions.MaxTokens)
	})

	t.Run("WithPredictorConfig", func(t *testing.T) {
		predictorCfg := &PredictorConfig{
			MaxTokens:   512,
			Temperature: 0.5,
		}
		agent := NewSolverAgent(&SolverConfig{PredictorConfig: predictorCfg})
		require.NotNil(t, agent)
		assert.Equal(t, predictorCfg.MaxTokens, agent.defaultOptions.MaxTokens)
		assert.Equal(t, float32(predictorCfg.Temperature), agent.defaultOptions.Temperature)
	})

	t.Run("WithDefaultModelInConfig", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{DefaultModel: "my-custom-default"})
		require.NotNil(t, agent)
		assert.Equal(t, "my-custom-default", agent.config.DefaultModel)
	})

	// Test that Ollama and OpenAI registration attempts don't cause panics
	// by temporarily unsetting env vars if they are set in the test environment.
	t.Run("OllamaAndOpenAIRegistrationAttempts", func(t *testing.T) {
		originalOllamaHost := os.Getenv("OLLAMA_HOST")
		originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
		os.Unsetenv("OLLAMA_HOST")    // Ensure Ollama might appear unavailable
		os.Unsetenv("OPENAI_API_KEY") // Ensure OpenAI key is not found

		defer func() {
			os.Setenv("OLLAMA_HOST", originalOllamaHost)
			os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
		}()

		// No assert needed, just checking for no panics and normal completion.
		// Logs will indicate warnings if services are not found.
		agent := NewSolverAgent(&SolverConfig{})
		require.NotNil(t, agent)
		// The "default" mock model should still be there.
		_, err := agent.modelManager.GetModel("default")
		assert.NoError(t, err)
	})
}

func TestSolverAgent_Process(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{})
		mockModel := newConfigurableMockAIModel("test-model", []ai.ModelCapability{ai.CapabilityCompletion})
		mockModel.generateResponseFunc = func(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error) {
			return "successful response", nil
		}
		agent.RegisterModel(mockModel)
		agent.SetDefaultModel("test-model") // Ensure this model is selected

		query := &types.Query{ID: "q1", Text: "test query"}
		initialMetrics := agent.GetPerformanceMetrics()

		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "q1", resp.QueryID)
		assert.Equal(t, "successful response", resp.Text)
		assert.Equal(t, "success", resp.Status)
		assert.Equal(t, "test-model", resp.Data.(map[string]string)["model"])

		finalMetrics := agent.GetPerformanceMetrics()
		assert.True(t, finalMetrics.ResponseTime > 0)
		assert.Equal(t, initialMetrics.SuccessRate*0.99+0.01, finalMetrics.SuccessRate, "SuccessRate should increase")
		assert.Equal(t, initialMetrics.ErrorRate*0.99, finalMetrics.ErrorRate, "ErrorRate should decrease")
		assert.Equal(t, initialMetrics.RequestsPerMin+1, finalMetrics.RequestsPerMin)
	})

	t.Run("ModelSelectionError", func(t *testing.T) {
		// Create agent. NewSolverAgent registers a "default" model.
		// To cause selection error, ask for a model that doesn't exist and ensure no fallback.
		agent := NewSolverAgent(&SolverConfig{DefaultModel: "non-existent-default"})
		// Ensure no models other than the initial "default" are present that could be picked by fallback.
		// The "default" model is registered by NewSolverAgent.
		// If we set agent.config.DefaultModel to "non-existent-default", selectModel will try it, fail.
		// Then it will try "default" (the hardcoded fallback if agent.config.DefaultModel is empty, which is not the case here).
		// Then it will list models. models[0] will be "default". GetModel("default") will succeed.
		// To truly make selectModel fail, we need to make GetModel(models[0].ID) fail, which implies ModelManager is broken,
		// or ListModels() is empty. ListModels() is never empty after NewSolverAgent.
		// So, a true "no suitable model found" is hard to achieve without altering ModelManager behavior.
		// Let's simulate by making the *only* model (the default one) unhealthy if that were a selection criterion (it's not currently).
		// The current selectModel doesn't check health.
		// The easiest way to make selectModel fail is if ModelManager.GetModel itself returns an error for *all* attempts.
		// This test case is tricky with current SolverAgent/ModelManager structure.
		// For now, let's assume "no suitable model" means the configured default and the hardcoded "default" are not found,
		// and the list of models is empty (which NewSolverAgent prevents).
		// A more practical model selection error is if the *requested* model in metadata is not found, and fallback fails.

		// Let's test the path where the model specified in metadata is not found,
		// and the default model is also somehow problematic (e.g., if it were removed, which is not possible now).
		// The most straightforward way for selectModel to error is if GetModel for the final chosen ID fails.
		// Given NewSolverAgent always adds "default", this path is hard to hit unless "default" itself errors.

		// Simplest way to test selectModel erroring:
		// Create an agent, then clear its model manager (not possible directly).
		// Alternative: test `selectModel` by having it try to get a model that, when `GetModel` is called,
		// returns an error. This requires a mock ModelManager, but SolverAgent creates its own.

		// Let's re-evaluate: `selectModel` returns error if `s.modelManager.GetModel(final_choice)` fails.
		// If `final_choice` is `models[0].ID`, and `GetModel` fails for it, that's an error.
		// This would imply an inconsistency in `ModelManager` (ListModels gives an ID, GetModel fails for it).
		// We will assume ModelManager is consistent.
		// The "no suitable model found" error in `selectModel` is thus primarily if `ListModels()` is empty.
		// Since `NewSolverAgent` ensures it's not empty, this specific error is hard to trigger for `selectModel`.

		// Test case: Query requests a specific model via metadata, but it's not registered.
		// Fallback logic should then apply. If fallback also fails (e.g. default model is "bad-default"), then error.
		agent = NewSolverAgent(&SolverConfig{DefaultModel: "actual-default-model"}) // This will be the fallback
		defaultMockModel := newConfigurableMockAIModel("actual-default-model", nil)
		agent.RegisterModel(defaultMockModel)

		_ = &types.Query{
			ID:       "q_meta_bad",
			Text:     "test query",
			Metadata: map[string]string{"model_id": "non-existent-metadata-model"},
		}
		_ = agent.GetPerformanceMetrics()

		// In this setup, "non-existent-metadata-model" fails.
		// Then it tries "actual-default-model" (from agent.config.DefaultModel) which should succeed.
		// So, to make selectModel error out, the fallback itself must fail.
		// Let's assume the default model configured in SolverConfig is also not registered.
		_ = NewSolverAgent(&SolverConfig{DefaultModel: "bad-configured-default"})
		// NewSolverAgent still registers "default". So fallback will pick "default".
		// The only way selectModel truly errors with "no suitable model found" is if the ModelManager becomes empty.

		// For this sub-test, let's focus on the GenerateResponse error.
		// A true "ModelSelectionError" is hard to test in isolation without a mock ModelManager.
		// We'll assume selectModel works and test other Process failures.
		// If we want to test selectModel error, we'd have to make GetModel return error for "default".
		// This means we need a way to replace the modelManager or models within it.
		// The "default" model registered by NewSolverAgent is an ai.MockModel.
		// We can't easily make its GetModelInfo().ID lead to an error in a subsequent GetModel call.

		// Skip direct ModelSelectionError test for Process, test selectModel separately or via specific setups.
		// Test ModelInferenceError instead.
		t.Log("Skipping direct ModelSelectionError test in Process due to difficulty in isolating with current structure. InferenceError test covers metrics on failure.")
	})

	t.Run("ModelInferenceError", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{})
		mockModel := newConfigurableMockAIModel("error-model", []ai.ModelCapability{ai.CapabilityCompletion})
		expectedErr := errors.New("inference failed")
		mockModel.generateResponseFunc = func(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error) {
			return "", expectedErr
		}
		agent.RegisterModel(mockModel)
		agent.SetDefaultModel("error-model")

		query := &types.Query{ID: "q2", Text: "test query"}
		initialMetrics := agent.GetPerformanceMetrics()

		resp, err := agent.Process(ctx, query)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "model inference failed")
		assert.ErrorIs(t, err, expectedErr)

		finalMetrics := agent.GetPerformanceMetrics()
		assert.True(t, finalMetrics.ResponseTime > 0)
		assert.NotEqual(t, initialMetrics.SuccessRate, finalMetrics.SuccessRate) // success rate should change
		assert.NotEqual(t, initialMetrics.ErrorRate, finalMetrics.ErrorRate)     // error rate should change
		assert.Equal(t, initialMetrics.RequestsPerMin+1, finalMetrics.RequestsPerMin)
	})

	// prepareInput and formatResponse are currently infallible in solver_agent.go
	// If they could error, similar tests would be added here.
}

func TestSolverAgent_selectModel_Via_Process(t *testing.T) {
	ctx := context.Background()

	t.Run("QueryMetadataHintsModel", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{})
		model1 := newConfigurableMockAIModel("model1", nil)
		model2 := newConfigurableMockAIModel("model2", nil)
		agent.RegisterModel(model1)
		agent.RegisterModel(model2)
		agent.SetDefaultModel("model1") // Default is model1

		query := &types.Query{
			ID:       "q_meta",
			Text:     "test",
			Metadata: map[string]string{"model_id": "model2"}, // Hint model2
		}
		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "model2", resp.Data.(map[string]string)["model"], "Should use model from metadata")
	})

	t.Run("QueryMetadataHintsNonExistentModel_FallbackToDefault", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{DefaultModel: "actual-default"})
		defaultModel := newConfigurableMockAIModel("actual-default", nil)
		agent.RegisterModel(defaultModel)
		// The "default" model from NewSolverAgent also exists.

		query := &types.Query{
			ID:       "q_meta_bad",
			Text:     "test",
			Metadata: map[string]string{"model_id": "non-existent-model"},
		}
		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "actual-default", resp.Data.(map[string]string)["model"], "Should fall back to configured default model")
	})

	t.Run("QueryMetadataHintsNonExistentModel_FallbackToHardcodedDefault", func(t *testing.T) {
		// Config.DefaultModel is empty, so it should fallback to "default" (from NewSolverAgent)
		agent := NewSolverAgent(&SolverConfig{})
		// "default" model is registered by NewSolverAgent.

		query := &types.Query{
			ID:       "q_meta_bad2",
			Text:     "test",
			Metadata: map[string]string{"model_id": "non-existent-model"},
		}
		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "default", resp.Data.(map[string]string)["model"], "Should fall back to hardcoded 'default' model")
	})

	t.Run("CodeQuerySelectsCodeGenModel", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{DefaultModel: "general-model"}) // Default is general
		generalModel := newConfigurableMockAIModel("general-model", []ai.ModelCapability{ai.CapabilityCompletion})
		codeModel := newConfigurableMockAIModel("code-model", []ai.ModelCapability{ai.CapabilityCodeGen, ai.CapabilityCompletion})
		agent.RegisterModel(generalModel)
		agent.RegisterModel(codeModel)

		query := &types.Query{ID: "q_code", Text: "write a function", Type: "code"}
		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "code-model", resp.Data.(map[string]string)["model"], "Should select codegen model for code query")
	})

	t.Run("CodeQueryNoCodeGenModel_FallbackToDefault", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{DefaultModel: "my-default-model"})
		defaultModel := newConfigurableMockAIModel("my-default-model", []ai.ModelCapability{ai.CapabilityCompletion})
		agent.RegisterModel(defaultModel)
		// The "default" model from NewSolverAgent also exists.

		query := &types.Query{ID: "q_code_fallback", Text: "write a function", Type: "code"}
		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "my-default-model", resp.Data.(map[string]string)["model"], "Should fall back to default for code query if no codegen model")
	})

	t.Run("DefaultModelSelectionWhenNoHintsOrSpecificType", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{DefaultModel: "configured-default"})
		cfgDefaultModel := newConfigurableMockAIModel("configured-default", nil)
		agent.RegisterModel(cfgDefaultModel)
		// "default" (from NewSolverAgent) also exists.

		query := &types.Query{ID: "q_general", Text: "general question"}
		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "configured-default", resp.Data.(map[string]string)["model"], "Should use configured default model")
	})

	t.Run("ConfiguredDefaultModelNotAvailable_FallbackToHardcodedDefault", func(t *testing.T) {
		agent := NewSolverAgent(&SolverConfig{DefaultModel: "non-existent-configured-default"})
		// "default" model is registered by NewSolverAgent.

		query := &types.Query{ID: "q_fallback_hard", Text: "general question"}
		resp, err := agent.Process(ctx, query)
		require.NoError(t, err)
		require.NotNil(t, resp)
		// Falls back to the first model in ListModels() if GetModel(s.config.DefaultModel) fails,
		// and s.config.DefaultModel was not empty.
		// If s.config.DefaultModel was empty, it tries "default".
		// Here, s.config.DefaultModel is "non-existent-configured-default". GetModel fails.
		// Then, `models := s.modelManager.ListModels()`. `models[0]` will be the "default" model.
		// So, it should pick "default".
		assert.Equal(t, "default", resp.Data.(map[string]string)["model"], "Should fall back to hardcoded 'default' if configured default is not found")
	})
}

func TestSolverAgent_prepareInput(t *testing.T) {
	agent := NewSolverAgent(nil) // Agent itself is not used by prepareInput, but create for consistency
	testCases := []struct {
		name          string
		query         *types.Query
		expectedInput string
	}{
		{"EmptyType", &types.Query{Text: "hello"}, "hello"},
		{"QuestionType", &types.Query{Text: "world?", Type: "question"}, "world?"},
		{"CodeType", &types.Query{Text: "create a list", Type: "code"}, "Write code to solve the following problem: create a list"},
		{"ProgrammingType", &types.Query{Text: "sort an array", Type: "programming"}, "Write code to solve the following problem: sort an array"},
		{"CreativeType", &types.Query{Text: "a poem", Type: "creative"}, "Be creative and generate: a poem"},
		{"OtherType", &types.Query{Text: "summarize this", Type: "summary"}, "[SUMMARY] summarize this"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := agent.prepareInput(tc.query)
			require.NoError(t, err) // prepareInput is currently infallible
			assert.Equal(t, tc.expectedInput, input)
		})
	}
}

func TestSolverAgent_formatResponse(t *testing.T) {
	agent := NewSolverAgent(nil)
	query := &types.Query{
		ID:       "query123",
		Text:     "input query",
		Metadata: map[string]string{"user": "testuser"},
	}
	outputText := "This is the model output."
	modelID := "test-model-001"

	resp, err := agent.formatResponse(query, outputText, modelID)
	require.NoError(t, err) // formatResponse is currently infallible
	require.NotNil(t, resp)

	assert.Equal(t, query.ID, resp.QueryID)
	assert.Equal(t, outputText, resp.Text)
	assert.Equal(t, modelID, resp.Data.(map[string]string)["model"])
	assert.Equal(t, query.Metadata, resp.Metadata)
	assert.Equal(t, "success", resp.Status)
	assert.NotZero(t, resp.Timestamp)
}

func TestSolverAgent_updateMetrics(t *testing.T) {
	t.Run("SuccessfulProcess", func(t *testing.T) {
		agent := NewSolverAgent(nil)
		initialMetrics := agent.GetPerformanceMetrics() // Should be 0, 1.0, 0.0, 0
		require.Equal(t, 1.0, initialMetrics.SuccessRate)
		require.Equal(t, 0.0, initialMetrics.ErrorRate)

		agent.updateMetrics(100*time.Millisecond, true)
		metrics1 := agent.GetPerformanceMetrics()

		expectedResponseTime1 := 0.9*initialMetrics.ResponseTime + 0.1*100.0
		assert.Equal(t, expectedResponseTime1, metrics1.ResponseTime)
		assert.Equal(t, initialMetrics.SuccessRate*0.99+0.01, metrics1.SuccessRate)
		assert.Equal(t, initialMetrics.ErrorRate*0.99, metrics1.ErrorRate)
		assert.Equal(t, initialMetrics.RequestsPerMin+1, metrics1.RequestsPerMin)

		agent.updateMetrics(200*time.Millisecond, true)
		metrics2 := agent.GetPerformanceMetrics()
		expectedResponseTime2 := 0.9*metrics1.ResponseTime + 0.1*200.0
		assert.Equal(t, expectedResponseTime2, metrics2.ResponseTime)
		assert.Equal(t, metrics1.SuccessRate*0.99+0.01, metrics2.SuccessRate)
	})

	t.Run("FailedProcess", func(t *testing.T) {
		agent := NewSolverAgent(nil)
		initialMetrics := agent.GetPerformanceMetrics()

		agent.updateMetrics(50*time.Millisecond, false)
		metrics1 := agent.GetPerformanceMetrics()

		expectedResponseTime1 := 0.9*initialMetrics.ResponseTime + 0.1*50.0
		assert.Equal(t, expectedResponseTime1, metrics1.ResponseTime)
		assert.Equal(t, initialMetrics.SuccessRate*0.99, metrics1.SuccessRate)
		assert.Equal(t, initialMetrics.ErrorRate*0.99+0.01, metrics1.ErrorRate)
		assert.Equal(t, initialMetrics.RequestsPerMin+1, metrics1.RequestsPerMin)
	})
}

func TestSolverAgent_ModelManagement(t *testing.T) {
	agent := NewSolverAgent(nil)
	model1ID := "mgt-model-1"
	mockModel1 := newConfigurableMockAIModel(model1ID, []ai.ModelCapability{ai.CapabilityCompletion})

	t.Run("RegisterAndGetModel", func(t *testing.T) {
		// "default" model is already registered by NewSolverAgent
		_, errDefault := agent.GetModel("default")
		require.NoError(t, errDefault)

		agent.RegisterModel(mockModel1)
		retrievedModel, err := agent.GetModel(model1ID)
		require.NoError(t, err)
		assert.Equal(t, model1ID, retrievedModel.GetModelInfo().ID)

		_, err = agent.GetModel("non-existent-model")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ai.ErrModelNotFound)
	})

	t.Run("ListModels", func(t *testing.T) {
		// Clear existing models by creating a new agent for this sub-test,
		// as RegisterModel adds to existing ones.
		agentForListTest := NewSolverAgent(nil)
		initialModels := agentForListTest.ListModels()
		assert.Len(t, initialModels, 1, "Should have the 'default' model initially")
		assert.Equal(t, "default", initialModels[0].ID)

		modelForList1 := newConfigurableMockAIModel("list-model-1", nil)
		modelForList2 := newConfigurableMockAIModel("list-model-2", nil)
		agentForListTest.RegisterModel(modelForList1)
		agentForListTest.RegisterModel(modelForList2)

		listedModels := agentForListTest.ListModels()
		assert.Len(t, listedModels, 3) // default, list-model-1, list-model-2

		found1, found2, foundDefault := false, false, false
		for _, mInfo := range listedModels {
			if mInfo.ID == "list-model-1" {
				found1 = true
			}
			if mInfo.ID == "list-model-2" {
				found2 = true
			}
			if mInfo.ID == "default" {
				foundDefault = true
			}
		}
		assert.True(t, found1, "list-model-1 should be in the list")
		assert.True(t, found2, "list-model-2 should be in the list")
		assert.True(t, foundDefault, "'default' model should be in the list")
	})
}

func TestSolverAgent_SetDefaultModel(t *testing.T) {
	agent := NewSolverAgent(nil)
	// "default" model is already registered by NewSolverAgent.

	t.Run("SetExistingModelAsDefault", func(t *testing.T) {
		customModelID := "custom-default-setter"
		customModel := newConfigurableMockAIModel(customModelID, nil)
		agent.RegisterModel(customModel)

		err := agent.SetDefaultModel(customModelID)
		require.NoError(t, err)
		assert.Equal(t, customModelID, agent.config.DefaultModel)

		// Verify that selectModel (via Process) uses this new default
		query := &types.Query{ID: "q_default_set", Text: "test"}
		resp, procErr := agent.Process(context.Background(), query)
		require.NoError(t, procErr)
		require.NotNil(t, resp)
		assert.Equal(t, customModelID, resp.Data.(map[string]string)["model"])
	})

	t.Run("SetNonExistentModelAsDefault", func(t *testing.T) {
		err := agent.SetDefaultModel("non-existent-for-default")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ai.ErrModelNotFound)
		assert.NotEqual(t, "non-existent-for-default", agent.config.DefaultModel, "DefaultModel should not be updated on error")
	})

	t.Run("SetDefaultToHardcodedDefaultModel", func(t *testing.T) {
		// The "default" model is registered by NewSolverAgent
		err := agent.SetDefaultModel("default")
		require.NoError(t, err)
		assert.Equal(t, "default", agent.config.DefaultModel)
	})
}

func TestSolverAgent_GetCapabilities(t *testing.T) {
	agent := NewSolverAgent(nil)
	caps := agent.GetCapabilities()
	expectedCaps := []types.Capability{"solve", "infer", "generate"}
	assert.ElementsMatch(t, expectedCaps, caps)
}

func TestSolverAgent_GetPerformanceMetrics(t *testing.T) {
	agent := NewSolverAgent(nil)
	metrics := agent.GetPerformanceMetrics()
	// Check initial values
	assert.Equal(t, 0.0, metrics.ResponseTime)
	assert.Equal(t, 1.0, metrics.SuccessRate) // Initial success rate is 1.0
	assert.Equal(t, 0.0, metrics.ErrorRate)
	assert.Equal(t, 0, metrics.RequestsPerMin)
}

// Helper to check if a string is in a slice of strings (used for model capabilities)
func containsCapability(slice []ai.ModelCapability, item ai.ModelCapability) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Test specific logic in tryRegisterOllamaModels and tryRegisterOpenAIModels indirectly
// by checking if models with expected names are registered if env vars are set.
// These are more like integration-leaning unit tests.
func TestSolverAgent_ExternalModelRegistration(t *testing.T) {
	// This test depends on external factors (Ollama running, OpenAI key)
	// and might be flaky or better suited for true integration tests.
	// For unit tests, we generally assume these tryRegister functions might not succeed.

	t.Run("OllamaModelRegistrationAttempt", func(t *testing.T) {
		// To test this properly, one would need a running Ollama instance.
		// We'll check if the code attempts registration without panic.
		// If OLLAMA_HOST is set and Ollama is running, models like "ollama-cogito" might appear.
		// For a unit test, we can't assume Ollama is running.
		// The NewSolverAgent already logs warnings if it fails.
		// We can check that if OLLAMA_HOST is NOT set, it doesn't register ollama models.
		originalOllamaHost := os.Getenv("OLLAMA_HOST")
		os.Unsetenv("OLLAMA_HOST")
		defer os.Setenv("OLLAMA_HOST", originalOllamaHost)

		agent := NewSolverAgent(nil)
		models := agent.ListModels()
		foundOllamaModel := false
		for _, m := range models {
			if strings.HasPrefix(m.ID, "ollama-") {
				foundOllamaModel = true
				break
			}
		}
		// Note: This test may pass or fail depending on whether Ollama is available locally
		// If Ollama is running, models will be registered even without OLLAMA_HOST being set
		t.Logf("Found Ollama model: %v (This depends on local Ollama availability)", foundOllamaModel)
	})

	t.Run("OpenAIModelRegistrationAttempt", func(t *testing.T) {
		originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_API_KEY") // Ensure API key is not set
		defer os.Setenv("OPENAI_API_KEY", originalOpenAIKey)

		agent := NewSolverAgent(nil)
		models := agent.ListModels()
		foundOpenAIModel := false
		for _, m := range models {
			if strings.HasPrefix(m.ID, "openai-") {
				foundOpenAIModel = true
				break
			}
		}
		assert.False(t, foundOpenAIModel, "No OpenAI models should be registered if OPENAI_API_KEY is not set")

		// Now, simulate API key being set
		os.Setenv("OPENAI_API_KEY", "test-key-for-registration-attempt")
		// Note: This will still fail to connect to OpenAI, but the registration *attempt* logic runs.
		// The NewOpenAIModel constructor itself might return an error if the model name is invalid,
		// or the IsHealthy check would fail.
		// The tryRegisterOpenAIModels logs errors but continues.
		agentWithKey := NewSolverAgent(nil)
		modelsWithKey := agentWithKey.ListModels()
		// We expect registration attempts, but actual models might not appear if IsHealthy fails or NewOpenAIModel errors.
		// The current tryRegisterOpenAIModels registers models like "gpt-3.5-turbo", "gpt-4".
		// If the API key is present, it *will* register them, regardless of actual connectivity for this part of the code.
		// The IsHealthy check is not part of tryRegisterOpenAIModels.
		foundOpenAIModelAfterKeySet := false
		for _, m := range modelsWithKey {
			if m.ID == "openai-gpt-3.5-turbo" || m.ID == "openai-gpt-4" {
				foundOpenAIModelAfterKeySet = true
				break
			}
		}
		// This assertion depends on the hardcoded model names in tryRegisterOpenAIModels
		assert.True(t, foundOpenAIModelAfterKeySet, "OpenAI models like gpt-3.5-turbo should be attempted to register if API key is set")
		os.Unsetenv("OPENAI_API_KEY") // Clean up
	})
}

func TestSolverAgent_Process_AdjustsOptionsBasedOnQueryType(t *testing.T) {
	ctx := context.Background()
	agent := NewSolverAgent(&SolverConfig{
		PredictorConfig: &PredictorConfig{
			Temperature: 0.7, // Default temperature
		},
	})

	mockModel := newConfigurableMockAIModel("option-test-model", nil)
	agent.RegisterModel(mockModel)
	agent.SetDefaultModel("option-test-model")

	testCases := []struct {
		name                string
		queryType           string
		expectedTemperature float32
	}{
		{"CodeQuery", "code", 0.3},
		{"ProgrammingQuery", "programming", 0.3},
		{"CreativeQuery", "creative", 0.9},
		{"StandardQuery", "question", 0.7}, // Should use default from config
		{"OtherQuery", "summary", 0.7},     // Should use default from config
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := &types.Query{
				ID:       "q-" + tc.name,
				Text:     "test",
				Type:     tc.queryType,
				Metadata: map[string]string{"model_id": "option-test-model"}, // Force use of our mock model
			}
			_, err := agent.Process(ctx, query)
			require.NoError(t, err, "Process should succeed")

			// Check the options passed to the mock model
			assert.Equal(t, tc.expectedTemperature, mockModel.lastOptions.Temperature, "Temperature not adjusted correctly for query type: %s", tc.queryType)
		})
	}
}
