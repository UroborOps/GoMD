package embeddings

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nroitero/gomd/backend/internal/config"
)

// QdrantStore implements vector storage backed by an external Qdrant database.
type QdrantStore struct {
	url        string
	key        string
	collection string
	client     *http.Client
}

// NewQdrantStore initializes the Qdrant client.
func NewQdrantStore(cfg *config.Config) *QdrantStore {
	return &QdrantStore{
		url:        strings.TrimSuffix(cfg.QdrantURL, "/"),
		key:        cfg.QdrantKey,
		collection: "gomd",
		client:     &http.Client{},
	}
}

func (s *QdrantStore) doReq(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, s.url+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.key != "" {
		req.Header.Set("api-key", s.key)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (s *QdrantStore) ensureCollection(size int) error {
	// Check if exists
	_, err := s.doReq("GET", "/collections/"+s.collection, nil)
	if err == nil {
		return nil // Exists
	}

	// Create
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     size,
			"distance": "Cosine",
		},
	}
	_, err = s.doReq("PUT", "/collections/"+s.collection, payload)
	return err
}

func stringToUUID(str string) string {
	h := md5.Sum([]byte(str))
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// Upsert adds or updates a vector in Qdrant.
func (s *QdrantStore) Upsert(id string, text string, vector []float32) error {
	if err := s.ensureCollection(len(vector)); err != nil {
		return err
	}

	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":     stringToUUID(id),
				"vector": vector,
				"payload": map[string]interface{}{
					"filepath": id,
				},
			},
		},
	}

	_, err := s.doReq("PUT", "/collections/"+s.collection+"/points?wait=true", payload)
	return err
}

// Delete removes a vector.
func (s *QdrantStore) Delete(id string) error {
	payload := map[string]interface{}{
		"points": []string{stringToUUID(id)},
	}
	_, err := s.doReq("POST", "/collections/"+s.collection+"/points/delete?wait=true", payload)
	return err
}

// Search performs a cosine similarity search.
func (s *QdrantStore) Search(query []float32, topK int) ([]SearchResult, error) {
	payload := map[string]interface{}{
		"vector": query,
		"limit":  topK,
		"with_payload": true,
	}

	respBytes, err := s.doReq("POST", "/collections/"+s.collection+"/points/search", payload)
	if err != nil {
		// If collection doesn't exist yet, just return empty
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	var res struct {
		Result []struct {
			Score   float32 `json:"score"`
			Payload struct {
				Filepath string `json:"filepath"`
			} `json:"payload"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBytes, &res); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, r := range res.Result {
		results = append(results, SearchResult{
			ID:    r.Payload.Filepath,
			Score: r.Score,
		})
	}

	return results, nil
}
