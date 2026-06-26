package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nroitero/gomd/backend/internal/config"
)

// Store defines the interface for vector databases.
type Store interface {
	Upsert(id string, text string, vector []float32) error
	Delete(id string) error
	Search(vector []float32, topK int) ([]SearchResult, error)
}

// SearchResult represents a document matched by vector similarity.
type SearchResult struct {
	ID    string
	Score float32
}

// Client handles interaction with OpenAI-compatible endpoints.
type Client struct {
	url    string
	key    string
	model  string
	client *http.Client
}

// NewClient creates a new embeddings client.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		url:    cfg.OpenAIURL,
		key:    cfg.OpenAIKey,
		model:  cfg.EmbedModel,
		client: &http.Client{},
	}
}

// Embed generates a vector embedding for the given text.
func (c *Client) Embed(text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"input": text,
		"model": c.model,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Determine endpoint URL (append /embeddings if just base URL is provided)
	endpoint := c.url
	if !strings.HasSuffix(endpoint, "/embeddings") {
		// Clean up trailing slash
		endpoint = strings.TrimSuffix(endpoint, "/") + "/embeddings"
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embeddings api error (%d): %s", resp.StatusCode, string(b))
	}

	var res struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return res.Data[0].Embedding, nil
}

// CheckHealth performs a test embedding to verify the endpoint and model configuration.
func (c *Client) CheckHealth() error {
	_, err := c.Embed("test")
	return err
}

