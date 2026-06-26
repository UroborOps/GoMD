package embeddings

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type vectorRecord struct {
	ID     string    `json:"id"`
	Vector []float32 `json:"vector"`
}

// LocalStore implements an in-memory vector store backed by a JSON file.
type LocalStore struct {
	filePath string
	mu       sync.RWMutex
	records  map[string][]float32
}

// NewLocalStore initializes the local vector store.
func NewLocalStore(vaultPath string) (*LocalStore, error) {
	gomdDir := filepath.Join(vaultPath, ".gomd")
	if err := os.MkdirAll(gomdDir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(gomdDir, "embeddings.json")
	store := &LocalStore{
		filePath: filePath,
		records:  make(map[string][]float32),
	}

	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

func (s *LocalStore) load() error {
	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var list []vectorRecord
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}

	for _, r := range list {
		s.records[r.ID] = r.Vector
	}
	return nil
}

func (s *LocalStore) save() error {
	var list []vectorRecord
	for id, vec := range s.records {
		list = append(list, vectorRecord{ID: id, Vector: vec})
	}

	b, err := json.Marshal(list)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, b, 0644)
}

// Upsert adds or updates a vector in the store.
func (s *LocalStore) Upsert(id string, text string, vector []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[id] = vector
	return s.save()
}

// Delete removes a vector.
func (s *LocalStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.records, id)
	return s.save()
}

// Search performs a cosine similarity search.
func (s *LocalStore) Search(query []float32, topK int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []SearchResult
	for id, vec := range s.records {
		score := cosineSimilarity(query, vec)
		results = append(results, SearchResult{ID: id, Score: score})
	}

	// Sort descending by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dot, magA, magB float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	if magA == 0 || magB == 0 {
		return 0.0
	}

	return dot / (float32(math.Sqrt(float64(magA))) * float32(math.Sqrt(float64(magB))))
}
