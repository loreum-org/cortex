package rag

import (
	"math"

	"github.com/loreum-org/cortex/pkg/types"
)

// GenerateIDHelper exports the unexported generateID function for testing.
func GenerateIDHelper(text string) string {
	return generateID(text)
}

// MinHelper exports the unexported min function for testing.
func MinHelper(a, b int) int {
	return min(a, b)
}

// BuildContextHelper exports the unexported buildContext method of RAGSystem for testing.
// It takes a RAGSystem instance because the original method is a receiver method.
// However, buildContext doesn't actually use any fields of RAGSystem, so we can pass nil
// if the RAGSystem instance itself is not important for the logic of buildContext.
// For safety and to match the original structure more closely if it were to change,
// it's better to have it available, even if not strictly used by the current buildContext.
func (r *RAGSystem) BuildContextHelper(docs []types.VectorDocument) string {
	// If buildContext was a standalone function, this helper would just call it.
	// Since it's a method, we call it on the provided RAGSystem instance.
	// If RAGSystem instance is not actually needed by buildContext's logic (e.g. no r.someField access),
	// this helper could be a static function that takes docs and returns string,
	// but the current buildContext is a method.
	// The current implementation of buildContext in rag_system.go does not use any fields from RAGSystem.
	// So, we can call it directly.
	return r.buildContext(docs)
}

// BuildPromptHelper exports the unexported buildPrompt method of RAGSystem for testing.
// Similar to BuildContextHelper, it's a method on RAGSystem.
func (r *RAGSystem) BuildPromptHelper(query string, context string) string {
	// The current implementation of buildPrompt in rag_system.go does not use any fields from RAGSystem.
	return r.buildPrompt(query, context)
}

// CosineSimilarityHelper exports the unexported cosineSimilarity function for testing.
func CosineSimilarityHelper(a, b []float32) float32 {
	return cosineSimilarity(a, b)
}

// SqrtHelper exports math.Sqrt for convenience in tests if needed, specifically for cosine similarity calculations.
func SqrtHelper(x float64) float64 {
	return math.Sqrt(x)
}
