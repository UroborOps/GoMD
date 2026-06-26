// Package api provides HTTP handlers for the GoMD API.
package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/UroborOps/GoMD/backend/internal/config"
	"github.com/UroborOps/GoMD/backend/internal/graph"
	"github.com/UroborOps/GoMD/backend/internal/indexer"
	"github.com/UroborOps/GoMD/backend/internal/locks"
	"github.com/UroborOps/GoMD/backend/internal/search"
)

// linkInfoFromIndexer converts indexer.Link slice to graph.LinkInfo slice.
func linkInfoFromIndexer(links []indexer.Link) []graph.LinkInfo {
	out := make([]graph.LinkInfo, len(links))
	for i, l := range links {
		out[i] = graph.LinkInfo{
			File:    l.File,
			Heading: l.Heading,
			Alias:   l.Alias,
		}
	}
	return out
}

// Handlers holds all HTTP handler functions.
type Handlers struct {
	vaultPath   string
	broadcaster *Broadcaster
	indexer     *indexer.Indexer
	searcher    *search.Searcher
	cfg         *config.Config
	locks       *locks.Manager
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(vaultPath string, broadcaster *Broadcaster, idx *indexer.Indexer, s *search.Searcher, cfg *config.Config, lm *locks.Manager) *Handlers {
	return &Handlers{
		vaultPath:   vaultPath,
		broadcaster: broadcaster,
		indexer:     idx,
		searcher:    s,
		cfg:         cfg,
		locks:       lm,
	}
}

// --- File CRUD Handler (unified) ---

// FilesHandler handles all file operations in a single endpoint.
// GET    /api/files          → list all files
// POST   /api/files          → create file (body: {path, content})
// GET    /api/files/:path    → get file content
// PUT    /api/files/:path    → update file (body: {content})
// DELETE /api/files/:path    → delete file
func (h *Handlers) FilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/files")
		path = strings.TrimPrefix(path, "/")
		path = filepath.Clean(path)

		switch r.Method {
		case http.MethodGet:
			// List all files or get single file
			if path == "" || path == "." {
				files := h.indexer.GetAllPaths()
				folders := h.indexer.GetAllFolders()
				jsonResponse(w, http.StatusOK, map[string]interface{}{
					"files":   files,
					"folders": folders,
					"count":   len(files) + len(folders),
				})
				return
			}
			// Get single file
			fullPath := filepath.Join(h.vaultPath, path)
			if !strings.HasPrefix(fullPath, h.vaultPath) {
				http.Error(w, "Path outside vault", http.StatusBadRequest)
				return
			}
			data, err := os.ReadFile(fullPath)
				if err != nil {
					http.Error(w, "File not found", http.StatusNotFound)
					return
				}
				fi := h.indexer.GetFileIndex(path)
				result := map[string]interface{}{
					"path":    path,
					"content": string(data),
				}
				if fi != nil {
					result["frontmatter"] = fi.Frontmatter
					result["title"] = fi.Title
					result["tags"] = fi.Tags
				}
				jsonResponse(w, http.StatusOK, result)

		case http.MethodPost:
			// Create file
			var req struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			if req.Path == "" {
				http.Error(w, "Path is required", http.StatusBadRequest)
				return
			}

			if h.locks.IsLocked(req.Path) {
				http.Error(w, "File is locked", http.StatusForbidden)
				return
			}

			fullPath := filepath.Join(h.vaultPath, req.Path)
			if !strings.HasPrefix(fullPath, h.vaultPath) {
				http.Error(w, "Path outside vault", http.StatusBadRequest)
				return
			}
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
				return
			}
			if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
				http.Error(w, fmt.Sprintf("Failed to write file: %v", err), http.StatusInternalServerError)
				return
			}
			h.broadcaster.Broadcast(NewEvent(EventFileCreated, req.Path, "file created"))
			jsonResponse(w, http.StatusCreated, map[string]string{"path": req.Path})

		case http.MethodPut:
			// Update file
			if path == "" || path == "." {
				http.Error(w, "Path required", http.StatusBadRequest)
				return
			}

			if h.locks.IsLocked(path) {
				http.Error(w, "File is locked", http.StatusForbidden)
				return
			}

			fullPath := filepath.Join(h.vaultPath, path)
			if !strings.HasPrefix(fullPath, h.vaultPath) {
				http.Error(w, "Path outside vault", http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			var req struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				req.Content = string(body)
			}
			if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
				http.Error(w, fmt.Sprintf("Failed to update file: %v", err), http.StatusInternalServerError)
				return
			}
			h.broadcaster.Broadcast(NewEvent(EventFileChange, path, "file updated"))
			jsonResponse(w, http.StatusOK, map[string]string{"path": path, "updated": "true"})

		case http.MethodDelete:
			// Delete file
			if path == "" || path == "." {
				http.Error(w, "Path required", http.StatusBadRequest)
				return
			}

			if h.locks.IsLocked(path) {
				http.Error(w, "File is locked", http.StatusForbidden)
				return
			}

			fullPath := filepath.Join(h.vaultPath, path)
			if !strings.HasPrefix(fullPath, h.vaultPath) {
				http.Error(w, "Path outside vault", http.StatusBadRequest)
				return
			}
			if err := os.Remove(fullPath); err != nil {
				if os.IsNotExist(err) {
					http.Error(w, "File not found", http.StatusNotFound)
					return
				}
				http.Error(w, fmt.Sprintf("Failed to delete file: %v", err), http.StatusInternalServerError)
				return
			}
			h.broadcaster.Broadcast(NewEvent(EventFileDeleted, path, "file deleted"))
			jsonResponse(w, http.StatusOK, map[string]string{"path": path, "deleted": "true"})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// --- Directory Handler ---

// FoldersHandler handles directory listing, creation, and deletion.
func (h *Handlers) FoldersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/folders")
		path = strings.TrimPrefix(path, "/")
		path = filepath.Clean(path)

		fullPath := filepath.Join(h.vaultPath, path)
		if !strings.HasPrefix(fullPath, h.vaultPath) {
			http.Error(w, "Path outside vault", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			entries, err := os.ReadDir(fullPath)
			if err != nil {
				http.Error(w, "Directory not found", http.StatusNotFound)
				return
			}

			dirs := make([]string, 0)
			files := make([]string, 0)
			for _, entry := range entries {
				if entry.IsDir() {
					dirs = append(dirs, entry.Name()+"/")
				} else if filepath.Ext(entry.Name()) == ".md" {
					files = append(files, entry.Name())
				}
			}

			jsonResponse(w, http.StatusOK, map[string]interface{}{
				"directories": dirs,
				"files":       files,
			})
		case http.MethodPost:
			var req struct {
				Path string `json:"path"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			if req.Path == "" {
				http.Error(w, "Path is required", http.StatusBadRequest)
				return
			}

			if h.locks.IsLocked(req.Path) {
				http.Error(w, "Path is locked", http.StatusForbidden)
				return
			}

			reqFullPath := filepath.Join(h.vaultPath, req.Path)
			if !strings.HasPrefix(reqFullPath, h.vaultPath) {
				http.Error(w, "Path outside vault", http.StatusBadRequest)
				return
			}
			if err := os.MkdirAll(reqFullPath, 0755); err != nil {
				http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
				return
			}
			h.indexer.Rebuild()
			h.broadcaster.Broadcast(NewEvent(EventFileCreated, req.Path, "folder created"))
			jsonResponse(w, http.StatusCreated, map[string]string{"path": req.Path})

		case http.MethodDelete:
			if path == "" || path == "." {
				http.Error(w, "Path required", http.StatusBadRequest)
				return
			}

			if h.locks.IsLocked(path) {
				http.Error(w, "Path is locked", http.StatusForbidden)
				return
			}

			if err := os.RemoveAll(fullPath); err != nil {
				http.Error(w, fmt.Sprintf("Failed to delete directory: %v", err), http.StatusInternalServerError)
				return
			}
			h.indexer.Rebuild()
			h.broadcaster.Broadcast(NewEvent(EventFileDeleted, path, "folder deleted"))
			jsonResponse(w, http.StatusOK, map[string]string{"path": path, "deleted": "true"})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// --- Rename Handler ---

// RenameHandler handles renaming/moving files and folders.
// POST /api/rename (body: {oldPath, newPath})
func (h *Handlers) RenameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OldPath string `json:"oldPath"`
			NewPath string `json:"newPath"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		if req.OldPath == "" || req.NewPath == "" {
			http.Error(w, "Old and new paths required", http.StatusBadRequest)
			return
		}

		if h.locks.IsLocked(req.OldPath) || h.locks.IsLocked(req.NewPath) {
			http.Error(w, "Path is locked", http.StatusForbidden)
			return
		}

		oldFull := filepath.Join(h.vaultPath, req.OldPath)
		newFull := filepath.Join(h.vaultPath, req.NewPath)

		if !strings.HasPrefix(oldFull, h.vaultPath) || !strings.HasPrefix(newFull, h.vaultPath) {
			http.Error(w, "Path outside vault", http.StatusBadRequest)
			return
		}

		if err := os.MkdirAll(filepath.Dir(newFull), 0755); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create parent directory: %v", err), http.StatusInternalServerError)
			return
		}

		if err := os.Rename(oldFull, newFull); err != nil {
			http.Error(w, fmt.Sprintf("Failed to rename: %v", err), http.StatusInternalServerError)
			return
		}
		
		h.indexer.Rebuild()
		h.broadcaster.Broadcast(NewEvent(EventFileDeleted, req.OldPath, "file renamed/moved"))
		h.broadcaster.Broadcast(NewEvent(EventFileCreated, req.NewPath, "file renamed/moved"))

		jsonResponse(w, http.StatusOK, map[string]string{
			"oldPath": req.OldPath,
			"newPath": req.NewPath,
			"renamed": "true",
		})
	}
}

// --- Search Handler ---

// SearchHandler handles search requests.
func (h *Handlers) SearchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		limitStr := r.URL.Query().Get("limit")

		limit := 20
		if limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &limit)
		}

		searchType := r.URL.Query().Get("type")

		var results []search.Result
		var err error

		if searchType == "semantic" {
			results, err = h.searcher.SemanticSearch(query, limit)
		} else {
			results, err = h.searcher.Search(query, limit)
		}

		if err != nil {
			log.Printf("search error: %v", err)
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"query":    query,
			"results":  results,
			"count":    len(results),
		})
	}
}

// --- Backlinks Handler ---

// BacklinksHandler returns backlinks for a file.
func (h *Handlers) BacklinksHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/backlinks/")
		path = filepath.Clean(path)

		backlinks := h.indexer.GetBacklinks(path)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"path":      path,
			"backlinks": backlinks,
			"count":     len(backlinks),
		})
	}
}

// --- Graph Handler ---

// GraphHandler returns graph data.
func (h *Handlers) GraphHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := graph.Build(
			h.indexer.GetFileCount(),
			h.indexer.GetAllPaths,
			func(path string) []string {
				links := h.indexer.GetBacklinks(path)
				var bl []string
				for _, l := range links {
					bl = append(bl, l.File)
				}
				return bl
			},
			func(path string) []graph.LinkInfo {
				return linkInfoFromIndexer(h.indexer.GetLinks(path))
			},
		)
		if err != nil {
			log.Printf("graph build error: %v", err)
			http.Error(w, "Graph build failed", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, g)
	}
}

// --- Transfer Handlers ---

// UploadHandler handles uploading files/folders via multipart/form-data.
func (h *Handlers) UploadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// 32 MB max memory
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Could not parse multipart form", http.StatusBadRequest)
			return
		}

		pathPrefix := r.FormValue("path") // Optional base path to upload into

		files := r.MultipartForm.File["files"]
		paths := r.MultipartForm.Value["paths"]

		for i, fileHeader := range files {
			relPath := fileHeader.Filename
			if i < len(paths) && paths[i] != "" {
				relPath = paths[i]
			}

			if pathPrefix != "" {
				relPath = filepath.Join(pathPrefix, relPath)
			}
			
			fullPath := filepath.Join(h.vaultPath, relPath)
			if !strings.HasPrefix(fullPath, h.vaultPath) {
				continue
			}

			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				continue
			}

			file, err := fileHeader.Open()
			if err != nil {
				continue
			}

			dst, err := os.Create(fullPath)
			if err == nil {
				io.Copy(dst, file)
				dst.Close()
			}
			file.Close()
		}

		h.indexer.Rebuild()
		jsonResponse(w, http.StatusOK, map[string]string{"uploaded": "true"})
	}
}

// DownloadHandler serves files directly or streams folders as zip.
func (h *Handlers) DownloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/download")
		path = strings.TrimPrefix(path, "/")
		path = filepath.Clean(path)

		fullPath := filepath.Join(h.vaultPath, path)
		if !strings.HasPrefix(fullPath, h.vaultPath) {
			http.Error(w, "Path outside vault", http.StatusBadRequest)
			return
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if !info.IsDir() {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", info.Name()))
			http.ServeFile(w, r, fullPath)
			return
		}

		// Stream as zip
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", info.Name()))
		
		zw := zip.NewWriter(w)
		defer zw.Close()

		filepath.Walk(fullPath, func(p string, i os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			relPath, err := filepath.Rel(fullPath, p)
			if err != nil || relPath == "." {
				return nil
			}

			if i.IsDir() {
				return nil
			}

			file, err := os.Open(p)
			if err != nil {
				return err
			}
			defer file.Close()

			f, err := zw.Create(relPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(f, file)
			return err
		})
	}
}

// --- Config Handler ---

// ConfigHandler returns current server configuration.
func (h *Handlers) ConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qdrantURL := h.cfg.QdrantExternalURL
		if qdrantURL == "" {
			qdrantURL = h.cfg.QdrantURL
		}
		
		s3URL := h.cfg.S3ExternalURL
		if s3URL == "" {
			s3URL = h.cfg.S3Endpoint
		}
		
		cfg := map[string]interface{}{
			"vault_path":        h.vaultPath,
			"port":              h.cfg.Port,
			"host":              h.cfg.Host,
			"theme":             h.cfg.Theme,
			"rag_enabled":       h.cfg.RAGEnabled,
			"qdrant_url":        qdrantURL,
			"git_enabled":       h.cfg.GitEnabled,
			"git_remote":        h.cfg.GitRemote,
			"s3_backup_enabled": h.cfg.S3BackupEnabled,
			"s3_endpoint":       s3URL,
		}
		jsonResponse(w, http.StatusOK, cfg)
	}
}

// --- Tree Handler ---

// TreeNode represents a node in the full file tree.
type TreeNode struct {
	Name     string               `json:"name"`
	Path     string               `json:"path"`
	Type     string               `json:"type"` // "folder" or "file"
	Size     int64                `json:"size,omitempty"`
	Lines    int                  `json:"lines,omitempty"`
	Chars    int                  `json:"chars,omitempty"`
	ModTime  string               `json:"modTime,omitempty"`
	Children map[string]*TreeNode `json:"children,omitempty"`
}

// TreeHandler returns a fully uncollapsed tree of the vault with file metadata.
func (h *Handlers) TreeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		root := &TreeNode{
			Name:     "Vault",
			Path:     "",
			Type:     "folder",
			Children: make(map[string]*TreeNode),
		}

		err := filepath.Walk(h.vaultPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			rel, err := filepath.Rel(h.vaultPath, path)
			if err != nil || rel == "." {
				return nil
			}

			// Skip hidden files/folders like .git
			if strings.HasPrefix(filepath.Base(path), ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Only include directories and .md files
			if !info.IsDir() && filepath.Ext(path) != ".md" {
				return nil
			}

			parts := strings.Split(rel, string(filepath.Separator))
			current := root
			for i, part := range parts {
				if current.Children == nil {
					current.Children = make(map[string]*TreeNode)
				}
				if _, exists := current.Children[part]; !exists {
					isLast := (i == len(parts)-1)
					nodeType := "folder"
					if isLast && !info.IsDir() {
						nodeType = "file"
					}
					node := &TreeNode{
						Name: part,
						Path: strings.Join(parts[:i+1], "/"),
						Type: nodeType,
					}
					if isLast && !info.IsDir() {
						node.Size = info.Size()
						node.ModTime = info.ModTime().Format("2006-01-02 15:04:05")
						// Read file to get chars and lines
						data, _ := os.ReadFile(path)
						node.Chars = len(data)
						node.Lines = bytes.Count(data, []byte("\n")) + 1
					}
					current.Children[part] = node
				}
				current = current.Children[part]
			}
			return nil
		})

		if err != nil {
			log.Printf("TreeHandler error: %v", err)
			http.Error(w, "Failed to build tree", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, root)
	}
}

// LockHandler handles GET and POST to /api/locks.
// GET /api/locks -> returns map of explicit locks
// POST /api/locks -> body: {path, locked}
func (h *Handlers) LockHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			states := h.locks.GetAll()
			json.NewEncoder(w).Encode(states)
			return
		}

		if r.Method == http.MethodPost {
			var req struct {
				Path   string `json:"path"`
				Locked bool   `json:"locked"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			h.locks.SetLock(req.Path, req.Locked)
			
			// Broadcast config_changed to refresh UI
			h.broadcaster.Broadcast(NewEvent("config_changed", "locks", "Lock state updated"))
			
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Helper Functions ---

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
