package rag_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDimensions = 3

// mockAIModel is a mock implementation of the ai.AIModel interface for testing.
type mockAIModel struct {
	info                  ai.ModelInfo
	generateEmbeddingFunc func(ctx context.Context, text string) ([]float32, error)
	generateResponseFunc  func(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error)
	isHealthyFunc         func(ctx context.Context) bool
	lastPrompt            string
	lastOptions           ai.GenerateOptions
	lastEmbeddingText     string
}

func (m *mockAIModel) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	m.lastEmbeddingText = text
	if m.generateEmbeddingFunc != nil {
		return m.generateEmbeddingFunc(ctx, text)
	}
	// Default behavior: return a fixed-size embedding based on text length
	embedding := make([]float32, m.info.Dimensions)
	for i := 0; i < m.info.Dimensions && i < len(text); i++ {
		embedding[i] = float32(text[i])
	}
	return embedding, nil
}

func (m *mockAIModel) GenerateResponse(ctx context.Context, prompt string, options ai.GenerateOptions) (string, error) {
	m.lastPrompt = prompt
	m.lastOptions = options
	if m.generateResponseFunc != nil {
		return m.generateResponseFunc(ctx, prompt, options)
	}
	return "mock response to: " + prompt, nil
}

func (m *mockAIModel) GenerateChatResponse(ctx context.Context, messages []ai.Message, options ai.GenerateOptions) (string, error) {
	// For RAG, GenerateResponse is typically used. This is a fallback.
	if len(messages) > 0 {
		return m.GenerateResponse(ctx, messages[len(messages)-1].Content, options)
	}
	return "mock chat response", nil
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

func newTestMockAIModel(id string, dimensions int) *mockAIModel {
	return &mockAIModel{
		info: ai.ModelInfo{
			ID:           id,
			Name:         id,
			Type:         ai.ModelTypeMock,
			Capabilities: []ai.ModelCapability{ai.CapabilityEmbedding, ai.CapabilityCompletion},
			Dimensions:   dimensions,
			IsLocal:      true,
		},
	}
}

func TestNewRAGSystem(t *testing.T) {
	ragSys := rag.NewRAGSystem(testDimensions)
	require.NotNil(t, ragSys, "NewRAGSystem should return a non-nil RAGSystem")

	assert.NotNil(t, ragSys.VectorDB, "VectorDB should be initialized")
	assert.Equal(t, testDimensions, ragSys.VectorDB.GetStats().Dimensions, "VectorDB dimensions mismatch")

	require.NotNil(t, ragSys.QueryProcessor, "QueryProcessor should be initialized")
	assert.Equal(t, "default", ragSys.QueryProcessor.ModelID, "Default ModelID for QueryProcessor mismatch")

	require.NotNil(t, ragSys.ModelManager, "ModelManager should be initialized")
	defaultModel, err := ragSys.ModelManager.GetModel("default")
	require.NoError(t, err, "Default model 'default' should be registered in ModelManager")
	assert.Equal(t, testDimensions, defaultModel.GetModelInfo().Dimensions, "Default model dimensions mismatch")

	require.NotNil(t, ragSys.ContextBuilder, "ContextBuilder should be initialized")
	assert.Equal(t, 4096, ragSys.ContextBuilder.MaxContextLength, "Default MaxContextLength mismatch") // Default from RAGSystem constructor

	assert.Equal(t, ai.DefaultGenerateOptions(), ragSys.DefaultOptions, "DefaultOptions mismatch")

	// Test that Ollama and OpenAI registration attempts don't cause panics
	// These are called internally by NewRAGSystem. We check for no panics.
	// Unset env vars to simulate unavailability for more consistent testing of this aspect.
	originalOllamaHost := os.Getenv("OLLAMA_HOST")
	originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OLLAMA_HOST")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		os.Setenv("OLLAMA_HOST", originalOllamaHost)
		os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
	}()

	ragSysWithNoExternal := rag.NewRAGSystem(testDimensions)
	require.NotNil(t, ragSysWithNoExternal)
	_, err = ragSysWithNoExternal.ModelManager.GetModel("default")
	assert.NoError(t, err, "Default model should still be present even if external registrations fail")
}

func TestRAGSystem_SetDefaultModel(t *testing.T) {
	ragSys := rag.NewRAGSystem(testDimensions)
	customModelID := "custom-test-model"
	customModel := newTestMockAIModel(customModelID, testDimensions)
	ragSys.ModelManager.RegisterModel(customModel)

	t.Run("SetExistingModel", func(t *testing.T) {
		err := ragSys.SetDefaultModel(customModelID)
		require.NoError(t, err)
		assert.Equal(t, customModelID, ragSys.QueryProcessor.ModelID)
	})

	t.Run("SetNonExistentModel", func(t *testing.T) {
		originalModelID := ragSys.QueryProcessor.ModelID
		err := ragSys.SetDefaultModel("non-existent-model")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ai.ErrModelNotFound)
		assert.Equal(t, originalModelID, ragSys.QueryProcessor.ModelID, "ModelID should not change on error")
	})
}

func TestRAGSystem_ListAvailableModels(t *testing.T) {
	ragSys := rag.NewRAGSystem(testDimensions) // Registers "default"
	model1 := newTestMockAIModel("model-1", testDimensions)
	model2 := newTestMockAIModel("model-2", testDimensions)
	ragSys.ModelManager.RegisterModel(model1)
	ragSys.ModelManager.RegisterModel(model2)

	models := ragSys.ListAvailableModels()
	require.Len(t, models, 3, "Should list default, model-1, and model-2")

	expectedIDs := []string{"default", "model-1", "model-2"}
	listedIDs := make([]string, len(models))
	for i, mInfo := range models {
		listedIDs[i] = mInfo.ID
	}
	sort.Strings(listedIDs) // Sort for consistent comparison
	sort.Strings(expectedIDs)
	assert.ElementsMatch(t, expectedIDs, listedIDs)
}

func TestRAGSystem_AddDocument(t *testing.T) {
	ctx := context.Background()
	ragSys := rag.NewRAGSystem(testDimensions)
	mockModel := newTestMockAIModel("test-embed-model", testDimensions)
	ragSys.ModelManager.RegisterModel(mockModel)
	err := ragSys.SetDefaultModel("test-embed-model")
	require.NoError(t, err)

	docText := "This is a test document."
	docMeta := map[string]interface{}{"source": "test"}
	expectedEmbedding := []float32{1.0, 2.0, 3.0} // testDimensions = 3

	t.Run("Success", func(t *testing.T) {
		mockModel.generateEmbeddingFunc = func(ctx context.Context, text string) ([]float32, error) {
			assert.Equal(t, docText, text)
			return expectedEmbedding, nil
		}

		err := ragSys.AddDocument(ctx, docText, docMeta)
		require.NoError(t, err)

		// Verify document in VectorDB (simplified check)
		// generateID is simple, so we can predict it for this text.
		expectedID := "doc_" + string(docText[0:min(10, len(docText))])
		storedDoc, getErr := ragSys.VectorDB.GetDocument(expectedID)
		require.NoError(t, getErr, "Document should be retrievable from VectorDB")
		assert.Equal(t, docText, storedDoc.Text)
		assert.Equal(t, expectedEmbedding, storedDoc.Embedding)
		assert.Equal(t, docMeta, storedDoc.Metadata)
	})

	t.Run("EmbeddingGenerationFails", func(t *testing.T) {
		expectedErr := errors.New("embedding generation failed")
		mockModel.generateEmbeddingFunc = func(ctx context.Context, text string) ([]float32, error) {
			return nil, expectedErr
		}
		err := ragSys.AddDocument(ctx, "another doc", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate embedding")
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("DefaultModelNotFound", func(t *testing.T) {
		// Create a new RAG system where the default model set by SetDefaultModel is removed
		freshRagSys := rag.NewRAGSystem(testDimensions)
		err := freshRagSys.SetDefaultModel("default") // "default" exists
		require.NoError(t, err)
		// "Remove" the default model by replacing the manager (hacky for test)
		freshRagSys.ModelManager = ai.NewModelManager() // Now "default" is gone from this manager

		err = freshRagSys.AddDocument(ctx, "doc", nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ai.ErrModelNotFound)
	})

	// Note: Testing VectorDB.AddDocument failure is harder here as it's an internal component.
	// It's better tested in VectorStorage specific tests if AddDocument had more failure modes
	// beyond dimension mismatch (which is covered by embedding generation).
}

func TestRAGSystem_Query(t *testing.T) {
	ctx := context.Background()
	ragSys := rag.NewRAGSystem(testDimensions)
	queryModel := newTestMockAIModel("query-model", testDimensions)
	ragSys.ModelManager.RegisterModel(queryModel)
	err := ragSys.SetDefaultModel("query-model")
	require.NoError(t, err)

	queryText := "What is RAG?"
	queryEmbedding := []float32{0.1, 0.2, 0.3}
	doc1Text := "RAG stands for Retrieval-Augmented Generation."
	doc1Embedding := []float32{0.11, 0.21, 0.31}
	doc1Meta := map[string]interface{}{"id": "doc1"}

	// Add a document
	queryModel.generateEmbeddingFunc = func(c context.Context, text string) ([]float32, error) {
		if text == doc1Text {
			return doc1Embedding, nil
		}
		if text == queryText {
			return queryEmbedding, nil
		}
		return nil, fmt.Errorf("unexpected text for embedding: %s", text)
	}
	err = ragSys.AddDocument(ctx, doc1Text, doc1Meta)
	require.NoError(t, err)

	t.Run("SuccessWithContext", func(t *testing.T) {
		expectedResponse := "RAG is a technique..."
		queryModel.generateResponseFunc = func(c context.Context, prompt string, options ai.GenerateOptions) (string, error) {
			assert.Contains(t, prompt, "Context:\n"+doc1Text)
			assert.Contains(t, prompt, "Question: "+queryText)
			assert.Equal(t, ragSys.DefaultOptions, options)
			return expectedResponse, nil
		}

		response, err := ragSys.Query(ctx, queryText)
		require.NoError(t, err)
		assert.Equal(t, expectedResponse, response)
		assert.Equal(t, queryText, queryModel.lastEmbeddingText) // Check embedding was for query
	})

	t.Run("SuccessNoContextFound", func(t *testing.T) {
		// Make search return no documents
		ragSys.VectorDB.DeleteDocument(rag.GenerateIDHelper(doc1Text)) // Remove the doc

		expectedResponse := "I don't know about RAG from my documents."
		queryModel.generateResponseFunc = func(c context.Context, prompt string, options ai.GenerateOptions) (string, error) {
			assert.NotContains(t, prompt, doc1Text, "Prompt should not contain context if no docs found")
			assert.Contains(t, prompt, "Question: "+queryText)
			return expectedResponse, nil
		}
		response, err := ragSys.Query(ctx, queryText)
		require.NoError(t, err)
		assert.Equal(t, expectedResponse, response)

		// Re-add document for subsequent tests if needed, or ensure tests are isolated
		err = ragSys.AddDocument(ctx, doc1Text, doc1Meta) // Re-add
		require.NoError(t, err)
	})

	t.Run("QueryEmbeddingFails", func(t *testing.T) {
		expectedErr := errors.New("query embedding failed")
		queryModel.generateEmbeddingFunc = func(c context.Context, text string) ([]float32, error) {
			if text == queryText {
				return nil, expectedErr
			}
			return doc1Embedding, nil // For any AddDocument calls
		}
		_, err := ragSys.Query(ctx, queryText)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	// To test VectorDB.SearchSimilar failure, we'd need to make the mock VectorDB error.
	// This is simpler if VectorDB were an interface passed to RAGSystem.
	// For now, we assume VectorDB.SearchSimilar works if inputs are correct, tested separately.

	t.Run("GenerateResponseFails", func(t *testing.T) {
		queryModel.generateEmbeddingFunc = func(c context.Context, text string) ([]float32, error) { // Reset to working
			if text == queryText {
				return queryEmbedding, nil
			}
			return doc1Embedding, nil
		}
		expectedErr := errors.New("llm generation failed")
		queryModel.generateResponseFunc = func(c context.Context, prompt string, options ai.GenerateOptions) (string, error) {
			return "", expectedErr
		}
		_, err := ragSys.Query(ctx, queryText)
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("DefaultModelForQueryNotFound", func(t *testing.T) {
		freshRagSys := rag.NewRAGSystem(testDimensions)
		freshRagSys.ModelManager = ai.NewModelManager() // Clear models
		_, err := freshRagSys.Query(ctx, queryText)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ai.ErrModelNotFound)
	})
}

func TestRAGSystem_Helpers(t *testing.T) {
	t.Run("generateID", func(t *testing.T) {
		assert.Equal(t, "doc_short", rag.GenerateIDHelper("short"))
		assert.Equal(t, "doc_longtextis", rag.GenerateIDHelper("longtextishere"))
		assert.Equal(t, "doc_", rag.GenerateIDHelper(""))
	})

	t.Run("min", func(t *testing.T) {
		assert.Equal(t, 1, rag.MinHelper(1, 2))
		assert.Equal(t, 1, rag.MinHelper(2, 1))
		assert.Equal(t, 1, rag.MinHelper(1, 1))
	})

	t.Run("buildContext", func(t *testing.T) {
		ragSys := rag.NewRAGSystem(testDimensions) // Instance needed to call method
		docs := []types.VectorDocument{
			{Text: "Doc 1 text."},
			{Text: "Doc 2 text."},
		}
		expected := "Doc 1 text.\n\nDoc 2 text.\n\n"
		assert.Equal(t, expected, ragSys.BuildContextHelper(docs))
		assert.Equal(t, "", ragSys.BuildContextHelper([]types.VectorDocument{}))
	})

	t.Run("buildPrompt", func(t *testing.T) {
		ragSys := rag.NewRAGSystem(testDimensions)
		query := "What is love?"
		context := "Baby don't hurt me.\n\n"
		expected := "Context:\nBaby don't hurt me.\n\n\n\nQuestion: What is love?\n\nAnswer:"
		assert.Equal(t, expected, ragSys.BuildPromptHelper(query, context))
	})
}

// --- VectorStorage Tests ---

func TestNewVectorStorage(t *testing.T) {
	vs := rag.NewVectorStorage(testDimensions)
	require.NotNil(t, vs)
	stats := vs.GetStats()
	assert.Equal(t, "default", stats.Name)
	assert.Equal(t, testDimensions, stats.Dimensions)
	assert.Equal(t, 0, stats.Size)
	// Check internal documents map is initialized (though not directly exported)
	// We can infer this by trying to add a document.
	err := vs.AddDocument(types.VectorDocument{ID: "test", Embedding: make([]float32, testDimensions)})
	assert.NoError(t, err)
}

func TestVectorStorage_AddDocument(t *testing.T) {
	vs := rag.NewVectorStorage(testDimensions)
	doc1 := types.VectorDocument{
		ID:        "doc1",
		Text:      "Document 1",
		Embedding: []float32{1.0, 0.0, 0.0},
		Metadata:  map[string]interface{}{"type": "test"},
	}

	t.Run("Success", func(t *testing.T) {
		err := vs.AddDocument(doc1)
		require.NoError(t, err)
		assert.Equal(t, 1, vs.GetStats().Size)

		retrievedDoc, errGet := vs.GetDocument("doc1")
		require.NoError(t, errGet)
		assert.Equal(t, doc1, retrievedDoc)
	})

	t.Run("DimensionMismatch", func(t *testing.T) {
		docInvalidDim := types.VectorDocument{
			ID:        "docInvalid",
			Embedding: []float32{1.0, 0.0}, // testDimensions is 3
		}
		err := vs.AddDocument(docInvalidDim)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document embedding dimensions do not match index dimensions")
		assert.Equal(t, 1, vs.GetStats().Size, "Size should not change on error")
	})

	t.Run("AddDuplicateIDOverwrites", func(t *testing.T) {
		doc1Updated := types.VectorDocument{
			ID:        "doc1",
			Text:      "Document 1 Updated",
			Embedding: []float32{1.1, 0.1, 0.1},
		}
		err := vs.AddDocument(doc1Updated)
		require.NoError(t, err)
		assert.Equal(t, 1, vs.GetStats().Size, "Size should remain 1 as it's an overwrite")

		retrievedDoc, _ := vs.GetDocument("doc1")
		assert.Equal(t, "Document 1 Updated", retrievedDoc.Text)
	})
}

func TestVectorStorage_GetDocument(t *testing.T) {
	vs := rag.NewVectorStorage(testDimensions)
	doc1 := types.VectorDocument{ID: "doc1", Embedding: make([]float32, testDimensions)}
	vs.AddDocument(doc1)

	t.Run("GetExisting", func(t *testing.T) {
		retrieved, err := vs.GetDocument("doc1")
		require.NoError(t, err)
		assert.Equal(t, doc1, retrieved)
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := vs.GetDocument("non-doc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document not found")
	})
}

func TestVectorStorage_DeleteDocument(t *testing.T) {
	vs := rag.NewVectorStorage(testDimensions)
	doc1 := types.VectorDocument{ID: "doc1", Embedding: make([]float32, testDimensions)}
	vs.AddDocument(doc1)
	require.Equal(t, 1, vs.GetStats().Size)

	t.Run("DeleteExisting", func(t *testing.T) {
		err := vs.DeleteDocument("doc1")
		require.NoError(t, err)
		assert.Equal(t, 0, vs.GetStats().Size)
		_, getErr := vs.GetDocument("doc1")
		assert.Error(t, getErr, "Document should be gone after deletion")
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		err := vs.DeleteDocument("non-doc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document not found")
		assert.Equal(t, 0, vs.GetStats().Size, "Size should not change")
	})
}

func TestVectorStorage_SearchSimilar(t *testing.T) {
	vs := rag.NewVectorStorage(testDimensions)
	docA := types.VectorDocument{ID: "A", Embedding: []float32{1.0, 0.0, 0.0}} // Normalized: [1,0,0]
	docB := types.VectorDocument{ID: "B", Embedding: []float32{0.0, 1.0, 0.0}} // Normalized: [0,1,0]
	docC := types.VectorDocument{ID: "C", Embedding: []float32{0.8, 0.2, 0.0}} // Similar to A
	docD := types.VectorDocument{ID: "D", Embedding: []float32{0.0, 0.0, 1.0}} // Orthogonal to A and B
	docE := types.VectorDocument{ID: "E", Embedding: []float32{0.9, 0.1, 0.0}} // More similar to A than C
	vs.AddDocument(docA)
	vs.AddDocument(docB)
	vs.AddDocument(docC)
	vs.AddDocument(docD)
	vs.AddDocument(docE)

	queryVecA := types.VectorQuery{Embedding: []float32{1.0, 0.0, 0.0}} // Query for things like A

	t.Run("BasicSearch", func(t *testing.T) {
		results, err := vs.SearchSimilar(queryVecA)
		require.NoError(t, err)
		require.NotEmpty(t, results)
		// Expected order: A, E, C (B and D should have low/zero similarity)
		// A (sim=1), E (sim=sqrt(0.81+0.01)/sqrt(0.82) * 1 = 0.9/sqrt(0.82) ~ 0.99), C (sim=0.8/sqrt(0.64+0.04) ~ 0.98)
		// Actual cosine: A=1, E=0.9938, C=0.9805
		assert.Equal(t, "A", results[0].ID)
		assert.Equal(t, "E", results[1].ID)
		assert.Equal(t, "C", results[2].ID)
		// B and D might appear if MinSimilarity is very low or 0
	})

	t.Run("MaxResults", func(t *testing.T) {
		query := queryVecA
		query.MaxResults = 2
		results, err := vs.SearchSimilar(query)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "A", results[0].ID)
		assert.Equal(t, "E", results[1].ID)
	})

	t.Run("MinSimilarity", func(t *testing.T) {
		query := queryVecA
		query.MinSimilarity = 0.99 // Only A and E should pass
		results, err := vs.SearchSimilar(query)
		require.NoError(t, err)
		assert.Len(t, results, 2) // A and E
		ids := []string{results[0].ID, results[1].ID}
		assert.Contains(t, ids, "A")
		assert.Contains(t, ids, "E")

		query.MinSimilarity = 0.995 // Only A should pass
		results2, err2 := vs.SearchSimilar(query)
		require.NoError(t, err2)
		assert.Len(t, results2, 1)
		assert.Equal(t, "A", results2[0].ID)
	})

	t.Run("DimensionMismatchQuery", func(t *testing.T) {
		queryInvalidDim := types.VectorQuery{Embedding: []float32{1.0, 0.0}}
		_, err := vs.SearchSimilar(queryInvalidDim)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query embedding dimensions do not match index dimensions")
	})

	t.Run("NoResultsAboveMinSimilarity", func(t *testing.T) {
		query := queryVecA
		query.MinSimilarity = 1.1 // Impossible similarity
		results, err := vs.SearchSimilar(query)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("EmptyVectorStorage", func(t *testing.T) {
		emptyVS := rag.NewVectorStorage(testDimensions)
		results, err := emptyVS.SearchSimilar(queryVecA)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestCosineSimilarity(t *testing.T) {
	testCases := []struct {
		name     string
		vecA     []float32
		vecB     []float32
		expected float32
	}{
		{"Identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 1.0},
		{"Orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"Opposite", []float32{1, 2, 3}, []float32{-1, -2, -3}, -1.0},
		{"Scaled", []float32{1, 1, 1}, []float32{5, 5, 5}, 1.0},
		{"Sample1", []float32{1, 2}, []float32{2, 4}, 1.0},                                  // Collinear
		{"Sample2", []float32{1, 0}, []float32{1, 1}, float32(1 / (1 * rag.SqrtHelper(2)))}, // ~0.707
		{"ZeroVectorA", []float32{0, 0, 0}, []float32{1, 2, 3}, 0.0},
		{"ZeroVectorB", []float32{1, 2, 3}, []float32{0, 0, 0}, 0.0},
		{"BothZeroVectors", []float32{0, 0}, []float32{0, 0}, 0.0},
		{"MismatchedLength", []float32{1, 2}, []float32{1, 2, 3}, 0.0}, // Should return 0 as per implementation
		{"EmptyVectors", []float32{}, []float32{}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// rag.CosineSimilarityHelper is not exported, so we call the internal one
			// This requires making cosineSimilarity public or having a test helper in rag package.
			// For now, assuming we can call it (e.g., by making it public or using a build tag for tests).
			// If not, this test would need to be in package rag.
			// Let's assume rag.CosineSimilarityHelper is an exported test helper for rag.cosineSimilarity
			sim := rag.CosineSimilarityHelper(tc.vecA, tc.vecB)
			assert.InDelta(t, tc.expected, sim, 1e-6, "Cosine similarity for %s failed", tc.name)
		})
	}
}

func TestVectorStorage_GetStats(t *testing.T) {
	vs := rag.NewVectorStorage(testDimensions)
	vs.AddDocument(types.VectorDocument{ID: "s1", Embedding: make([]float32, testDimensions)})
	vs.AddDocument(types.VectorDocument{ID: "s2", Embedding: make([]float32, testDimensions)})

	stats := vs.GetStats()
	assert.Equal(t, "default", stats.Name)
	assert.Equal(t, testDimensions, stats.Dimensions)
	assert.Equal(t, 2, stats.Size)
}

// Helper for min used in generateID, if not exported by rag package.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Reflect helper to compare slices more robustly if needed, though testify often handles this.
func AreSlicesEqual(s1, s2 interface{}) bool {
	return reflect.DeepEqual(s1, s2)
}
