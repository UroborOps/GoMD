// Package search provides full-text search with fuzzy matching.
package search

import (
	"log"
	"strings"
	"sync"

	"github.com/nroitero/gomd/backend/internal/config"
	"github.com/nroitero/gomd/backend/internal/embeddings"
	"github.com/nroitero/gomd/backend/internal/indexer"
)

// Result represents a single search result.
type Result struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	Content   string `json:"content"`
	Score     float64 `json:"score"`
	Backlinks int    `json:"backlinks"`
}

// Searcher provides search functionality over the vault.
type Searcher struct {
	mu       sync.RWMutex
	indexer  *indexer.Indexer
	terms    map[string][]string // term → list of file paths

	// RAG
	ragEnabled  bool
	embedClient *embeddings.Client
	vectorStore embeddings.Store
}

// NewSearcher creates a new Searcher backed by the given indexer.
func NewSearcher(idx *indexer.Indexer, cfg *config.Config) *Searcher {
	s := &Searcher{
		indexer: idx,
		terms:   make(map[string][]string),
	}

	if cfg.RAGEnabled {
		s.ragEnabled = true
		s.embedClient = embeddings.NewClient(cfg)

		if err := s.embedClient.CheckHealth(); err != nil {
			log.Printf("search warning: RAG enabled but embeddings API failed: %v", err)
			log.Printf("search warning: Check your GOMD_OPENAI_API_URL and GOMD_EMBED_MODEL settings.")
		} else {
			log.Println("search: successfully connected to embeddings API")
		}

		if cfg.QdrantURL != "" {
			s.vectorStore = embeddings.NewQdrantStore(cfg)
			log.Println("search: using Qdrant vector store")
		} else {
			store, err := embeddings.NewLocalStore(cfg.VaultPath)
			if err != nil {
				log.Printf("search error: failed to init local vector store: %v", err)
			} else {
				s.vectorStore = store
				log.Println("search: using local JSON vector store")
			}
		}
	}

	return s
}

// Rebuild rebuilds the search index from the indexer.
func (s *Searcher) Rebuild() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.terms = make(map[string][]string)

	paths := s.indexer.GetAllPaths()
	for _, path := range paths {
		fi := s.indexer.GetFileIndex(path)
		if fi == nil {
			continue
		}

		// Index title
		for _, term := range tokenize(fi.Title) {
			s.terms[term] = append(s.terms[term], path)
		}

		// Index body
		for _, term := range tokenize(fi.Body) {
			s.terms[term] = append(s.terms[term], path)
		}

		// RAG: Fire and forget embedding generation (in a real app, use a worker pool)
		if s.ragEnabled && s.vectorStore != nil && s.embedClient != nil {
			go s.embedFile(path, fi.Body)
		}
	}
}

func (s *Searcher) embedFile(path, text string) {
	if text == "" {
		return
	}
	vec, err := s.embedClient.Embed(text)
	if err != nil {
		log.Printf("search warning: failed to embed %s: %v", path, err)
		return
	}
	if err := s.vectorStore.Upsert(path, text, vec); err != nil {
		log.Printf("search warning: failed to store vector for %s: %v", path, err)
	}
}

// SemanticSearch performs a RAG cosine similarity search.
func (s *Searcher) SemanticSearch(query string, limit int) ([]Result, error) {
	if !s.ragEnabled || s.vectorStore == nil {
		return nil, nil
	}

	vec, err := s.embedClient.Embed(query)
	if err != nil {
		return nil, err
	}

	vResults, err := s.vectorStore.Search(vec, limit)
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, vr := range vResults {
		fi := s.indexer.GetFileIndex(vr.ID)
		if fi == nil {
			continue // File might have been deleted
		}

		// Provide a snippet using the standard function (could be improved to match semantic terms)
		snippet := buildSnippet(fi.Body, tokenize(query))
		results = append(results, Result{
			Path:      fi.Path,
			Title:     fi.Title,
			Snippet:   snippet,
			Content:   fi.Body,
			Score:     float64(vr.Score),
			Backlinks: len(fi.Backlinks),
		})
	}

	return results, nil
}

// Search performs a search query and returns results sorted by relevance.
func (s *Searcher) Search(query string, limit int) ([]Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query == "" {
		return nil, nil
	}

	// Tokenize query
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil, nil
	}

	// Score each file by number of matching terms
	scores := make(map[string]float64)
	for _, term := range terms {
		if paths, ok := s.terms[term]; ok {
			for _, p := range paths {
				scores[p]++
			}
		}
	}

	// Also search tags
	if strings.HasPrefix(query, "tag:") {
		tag := strings.TrimPrefix(query, "tag:")
		if paths := s.indexer.GetTagFiles(tag); paths != nil {
			for p := range paths {
				scores[p] += 2.0 // boost tag matches
			}
		}
	}

	// Sort by score descending
	type scored struct {
		path  string
		score float64
	}
	var scoredResults []scored
	for p, score := range scores {
		scoredResults = append(scoredResults, scored{p, score})
	}

	// Simple sort
	for i := 0; i < len(scoredResults); i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].score > scoredResults[i].score {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	// Build results
	if limit <= 0 {
		limit = 20
	}
	results := make([]Result, 0, limit)
	for i, sr := range scoredResults {
		if i >= limit {
			break
		}
		fi := s.indexer.GetFileIndex(sr.path)
		if fi == nil {
			continue
		}

		snippet := buildSnippet(fi.Body, terms)
		results = append(results, Result{
			Path:      fi.Path,
			Title:     fi.Title,
			Snippet:   snippet,
			Content:   fi.Body,
			Score:     sr.score,
			Backlinks: len(fi.Backlinks),
		})
	}

	return results, nil
}

// buildSnippet creates a text snippet around the first matching term.
func buildSnippet(body string, terms []string) string {
	if body == "" {
		return ""
	}

	limit := 200
	if len(body) <= limit {
		return body
	}

	for _, term := range terms {
		if idx := strings.Index(strings.ToLower(body), term); idx >= 0 {
			pos := idx
			if pos > limit/2 {
				pos = limit / 2
			}
			end := pos + limit
			if end > len(body) {
				end = len(body)
			}
			return body[pos:end]
		}
	}

	return body[:limit]
}

// tokenize splits text into lowercase tokens.
func tokenize(text string) []string {
	words := strings.Fields(text)
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		tokens = append(tokens, strings.ToLower(w))
	}
	return tokens
}
