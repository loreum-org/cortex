package types

// VectorQuery represents a query to the vector database
type VectorQuery struct {
	Embedding     []float32 `json:"embedding"`
	MaxResults    int       `json:"max_results"`
	MinSimilarity float32   `json:"min_similarity"`
}

// VectorDocument represents a document in the vector database
type VectorDocument struct {
	ID        string                 `json:"id"`
	Text      string                 `json:"text"`
	Embedding []float32              `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// VectorIndex represents a vector index
type VectorIndex struct {
	Name       string `json:"name"`
	Dimensions int    `json:"dimensions"`
	Size       int    `json:"size"`
}

// QueryProcessor processes queries for the RAG system
type QueryProcessor struct {
	ModelID string `json:"model_id"`
}

// ModelManager manages models for the RAG system
type ModelManager struct {
	Models map[string]interface{} `json:"models"`
}

// ContextBuilder builds context for the RAG system
type ContextBuilder struct {
	MaxContextLength int `json:"max_context_length"`
}

// RAGSystem represents the RAG system
type RAGSystem struct {
	VectorDB       interface{}     `json:"-"` // Vector database interface
	QueryProcessor *QueryProcessor `json:"query_processor"`
	ModelManager   *ModelManager   `json:"model_manager"`
	ContextBuilder *ContextBuilder `json:"context_builder"`
}

// NewRAGSystem creates a new RAG system
func NewRAGSystem() *RAGSystem {
	return &RAGSystem{
		QueryProcessor: &QueryProcessor{
			ModelID: "default",
		},
		ModelManager: &ModelManager{
			Models: make(map[string]interface{}),
		},
		ContextBuilder: &ContextBuilder{
			MaxContextLength: 4096,
		},
	}
}
