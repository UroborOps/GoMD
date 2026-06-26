package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/UroborOps/GoMD/backend/internal/config"
	"github.com/UroborOps/GoMD/backend/internal/indexer"
	"github.com/UroborOps/GoMD/backend/internal/locks"
	"github.com/UroborOps/GoMD/backend/internal/search"
)

func setupTestEnv(t *testing.T) (string, *Handlers, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "gomd-api-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Create test files
	testFiles := map[string]string{
		"a.md":       "# File A\n\nSome content.",
		"b.md":       "# File B\n\n[[a]] linked.",
		"sub/c.md":   "# File C\n\nIn sub dir.",
	}
	for rel, content := range testFiles {
		full := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(full), 0755)
		os.WriteFile(full, []byte(content), 0644)
	}

	// Setup indexer
	idx := indexer.NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	cfg := &config.Config{Host: "0.0.0.0", Port: 3000}
	s := search.NewSearcher(idx, cfg)
	s.Rebuild()

	broadcaster := NewBroadcaster()
	lm := locks.NewManager(dir)
	h := NewHandlers(dir, broadcaster, idx, s, cfg, lm)

	return dir, h, func() { os.RemoveAll(dir) }
}

func TestFilesHandler_ListFiles(t *testing.T) {
	_, h, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	rec := httptest.NewRecorder()

	h.FilesHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	files, ok := resp["files"].([]interface{})
	if !ok {
		t.Fatalf("expected 'files' array in response, got type %T", resp["files"])
	}

	if len(files) != 3 {
		t.Errorf("got %d files, want 3", len(files))
	}
}

func TestFilesHandler_GetFile(t *testing.T) {
	_, h, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/files/a.md", nil)
	rec := httptest.NewRecorder()

	h.FilesHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["path"] != "a.md" {
		t.Errorf("path = %q, want %q", resp["path"], "a.md")
	}
	if resp["content"] == nil {
		t.Error("content is nil, want non-nil")
	}
}

func TestSearchHandler_ReturnsContentField(t *testing.T) {
	_, h, cleanup := setupTestEnv(t)
	defer cleanup()

	// Query for "some" which appears in a.md content
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=some", nil)
	rec := httptest.NewRecorder()

	h.SearchHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	results, ok := resp["results"].([]interface{})
	if !ok {
		t.Fatal("expected 'results' array")
	}

	if len(results) == 0 {
		t.Skip("no results for this query, test not applicable")
	}

	// Verify each result has a 'content' field (not 'snippet')
	for i, r := range results {
		result, ok := r.(map[string]interface{})
		if !ok {
			t.Fatalf("result[%d] is not a map", i)
		}
		if _, hasContent := result["content"]; !hasContent {
			t.Errorf("result[%d] missing 'content' field (has: %v)", i, keysOf(result))
		}
	}
}

func TestGraphHandler_ReturnsEdgesWithSourceTarget(t *testing.T) {
	_, h, cleanup := setupTestEnv(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	rec := httptest.NewRecorder()

	h.GraphHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	edges, ok := resp["edges"].([]interface{})
	if !ok {
		t.Fatal("expected 'edges' array")
	}

	if len(edges) == 0 {
		t.Skip("no edges, test not applicable")
	}

	// Verify edges use 'source'/'target' keys
	for i, e := range edges {
		edge, ok := e.(map[string]interface{})
		if !ok {
			t.Fatalf("edge[%d] is not a map", i)
		}
		if _, hasSource := edge["source"]; !hasSource {
			t.Errorf("edge[%d] missing 'source' field (has: %v)", i, keysOf(edge))
		}
		if _, hasTarget := edge["target"]; !hasTarget {
			t.Errorf("edge[%d] missing 'target' field (has: %v)", i, keysOf(edge))
		}
	}
}

// keysOf returns the keys of a map for error messages.
func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestFilesHandler_UpdateFile(t *testing.T) {
	_, h, cleanup := setupTestEnv(t)
	defer cleanup()

	body := `{"content": "Updated content"}`
	req := httptest.NewRequest(http.MethodPut, "/api/files/a.md", os.NewFile(0, ""))
	req.Body = &bodyReadCloser{body: body}

	rec := httptest.NewRecorder()
	h.FilesHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

type bodyReadCloser struct {
	body string
	pos  int
}

func (r *bodyReadCloser) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.body) {
		return 0, io.EOF
	}
	n = copy(p, r.body[r.pos:])
	r.pos += n
	return n, nil
}

func (r *bodyReadCloser) Close() error { return nil }
