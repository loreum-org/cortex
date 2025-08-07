package rag

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// VectorStorage represents a persistent vector database
type VectorStorage struct {
	documents    map[string]types.VectorDocument
	index        *types.VectorIndex
	mu           sync.RWMutex
	persistPath  string // File path for persistence
	autoSave     bool   // Auto-save on changes
	lastSaveTime time.Time
}

// NewVectorStorage creates a new vector storage instance
func NewVectorStorage(dimensions int) *VectorStorage {
	return &VectorStorage{
		documents:    make(map[string]types.VectorDocument),
		index:        &types.VectorIndex{
			Name:       "default",
			Dimensions: dimensions,
			Size:       0,
		},
		mu:           sync.RWMutex{},
		persistPath:  "data/vector_db.json", // Default persist path
		autoSave:     true,
		lastSaveTime: time.Now(),
	}
}

// NewPersistentVectorStorage creates a vector storage instance with custom persistence path
func NewPersistentVectorStorage(dimensions int, persistPath string) *VectorStorage {
	vs := &VectorStorage{
		documents:    make(map[string]types.VectorDocument),
		index:        &types.VectorIndex{
			Name:       "default",
			Dimensions: dimensions,
			Size:       0,
		},
		mu:           sync.RWMutex{},
		persistPath:  persistPath,
		autoSave:     true,
		lastSaveTime: time.Now(),
	}
	
	// Try to load existing data
	if err := vs.LoadFromDisk(); err != nil {
		log.Printf("Warning: Could not load existing vector database from %s: %v", persistPath, err)
	} else {
		log.Printf("Loaded vector database from %s with %d documents", persistPath, vs.index.Size)
	}
	
	return vs
}

// AddDocument adds a document to the vector storage
func (v *VectorStorage) AddDocument(doc types.VectorDocument) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if the document has the correct dimensions
	if len(doc.Embedding) != v.index.Dimensions {
		return errors.New("document embedding dimensions do not match index dimensions")
	}

	// Check if this is a new document or an update
	_, exists := v.documents[doc.ID]

	// Add the document to the storage
	v.documents[doc.ID] = doc

	// Only increment size if it's a new document
	if !exists {
		v.index.Size++
	}

	// Auto-save if enabled
	if v.autoSave {
		go func() {
			if err := v.SaveToDisk(); err != nil {
				log.Printf("Failed to auto-save vector database: %v", err)
			}
		}()
	}

	return nil
}

// GetDocument retrieves a document from the vector storage
func (v *VectorStorage) GetDocument(id string) (types.VectorDocument, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	doc, exists := v.documents[id]
	if !exists {
		return types.VectorDocument{}, errors.New("document not found")
	}

	return doc, nil
}

// DeleteDocument deletes a document from the vector storage
func (v *VectorStorage) DeleteDocument(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.documents[id]; !exists {
		return errors.New("document not found")
	}

	delete(v.documents, id)
	v.index.Size--

	return nil
}

// SearchSimilar searches for similar documents
func (v *VectorStorage) SearchSimilar(query types.VectorQuery) ([]types.VectorDocument, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check if the query has the correct dimensions
	if len(query.Embedding) != v.index.Dimensions {
		return nil, errors.New("query embedding dimensions do not match index dimensions")
	}

	// Calculate similarity for all documents
	type docWithScore struct {
		doc   types.VectorDocument
		score float32
	}

	results := make([]docWithScore, 0, len(v.documents))
	for _, doc := range v.documents {
		score := cosineSimilarity(query.Embedding, doc.Embedding)
		if score >= query.MinSimilarity {
			results = append(results, docWithScore{doc, score})
		}
	}

	// Sort by similarity score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Limit results to max_results
	if query.MaxResults > 0 && len(results) > query.MaxResults {
		results = results[:query.MaxResults]
	}

	// Extract just the documents
	docs := make([]types.VectorDocument, len(results))
	for i, result := range results {
		docs[i] = result.doc
	}

	return docs, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// GetStats returns statistics about the vector storage
func (v *VectorStorage) GetStats() types.VectorIndex {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return *v.index
}

// PersistentData represents the data structure for persistence
type PersistentData struct {
	Documents []types.VectorDocument `json:"documents"`
	Index     types.VectorIndex      `json:"index"`
	SavedAt   time.Time              `json:"saved_at"`
}

// SaveToDisk saves the vector database to disk
func (v *VectorStorage) SaveToDisk() error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Create data directory if it doesn't exist
	dir := filepath.Dir(v.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Convert documents map to slice for JSON serialization
	documentSlice := make([]types.VectorDocument, 0, len(v.documents))
	for _, doc := range v.documents {
		documentSlice = append(documentSlice, doc)
	}

	// Create persistent data structure
	data := PersistentData{
		Documents: documentSlice,
		Index:     *v.index,
		SavedAt:   time.Now(),
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Write to file atomically
	tempPath := v.persistPath + ".tmp"
	if err := ioutil.WriteFile(tempPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, v.persistPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	v.lastSaveTime = time.Now()
	log.Printf("Vector database saved to %s (%d documents)", v.persistPath, len(v.documents))
	return nil
}

// LoadFromDisk loads the vector database from disk
func (v *VectorStorage) LoadFromDisk() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(v.persistPath); os.IsNotExist(err) {
		return fmt.Errorf("persistence file does not exist: %s", v.persistPath)
	}

	// Read file
	jsonData, err := ioutil.ReadFile(v.persistPath)
	if err != nil {
		return fmt.Errorf("failed to read persistence file: %w", err)
	}

	// Unmarshal JSON
	var data PersistentData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	// Restore documents
	v.documents = make(map[string]types.VectorDocument)
	for _, doc := range data.Documents {
		v.documents[doc.ID] = doc
	}

	// Restore index
	v.index = &data.Index

	log.Printf("Vector database loaded from %s: %d documents, saved at %s", 
		v.persistPath, len(v.documents), data.SavedAt.Format(time.RFC3339))
	return nil
}

// SetPersistPath sets the persistence file path
func (v *VectorStorage) SetPersistPath(path string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.persistPath = path
}

// GetPersistPath returns the current persistence file path
func (v *VectorStorage) GetPersistPath() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.persistPath
}

// EnableAutoSave enables or disables auto-save
func (v *VectorStorage) EnableAutoSave(enable bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.autoSave = enable
}

// ForceSync forces a save to disk immediately
func (v *VectorStorage) ForceSync() error {
	return v.SaveToDisk()
}
