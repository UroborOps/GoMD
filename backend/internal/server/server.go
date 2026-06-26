// Package server sets up and runs the GoMD HTTP server.
package server

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/UroborOps/GoMD/backend/internal/api"
	"github.com/UroborOps/GoMD/backend/internal/config"
	"github.com/UroborOps/GoMD/backend/internal/fsnotify"
	"github.com/UroborOps/GoMD/backend/internal/gitsync"
	"github.com/UroborOps/GoMD/backend/internal/indexer"
	"github.com/UroborOps/GoMD/backend/internal/locks"
	"github.com/UroborOps/GoMD/backend/internal/mcp"
	"github.com/UroborOps/GoMD/backend/internal/s3backup"
	"github.com/UroborOps/GoMD/backend/internal/search"
	"github.com/UroborOps/GoMD/backend/internal/static"
)

// Server holds the HTTP server and its components.
type Server struct {
	server      *http.Server
	broadcaster *api.Broadcaster
	indexer     *indexer.Indexer
	searcher    *search.Searcher
	watcher     *fsnotify.Watcher
	cfg         *config.Config
	locks       *locks.Manager
	mcpServer   *mcp.Server
}

// New creates a new Server with the given configuration.
func New(cfg *config.Config) (*Server, error) {
	broadcaster := api.NewBroadcaster()
	indexer := indexer.NewIndexer()
	searcher := search.NewSearcher(indexer, cfg)
	lm := locks.NewManager(cfg.VaultPath)

	s := &Server{
		broadcaster: broadcaster,
		indexer:     indexer,
		searcher:    searcher,
		cfg:         cfg,
		locks:       lm,
		mcpServer:   mcp.New(cfg.VaultPath, indexer, searcher, lm),
	}

	// Initialize index
	if err := s.initIndexes(); err != nil {
		return nil, err
	}

	// Start git auto-sync worker
	gitsync.Start(cfg)

	// Start S3 backup worker
	s3backup.Start(cfg)

	// Start file watcher
	w, err := fsnotify.NewWatcher(cfg.VaultPath, broadcaster, indexer)
	if err != nil {
		log.Printf("warning: failed to start file watcher: %v", err)
		// Continue without watcher
	} else {
		s.watcher = w
	}

	// Build routes
	mux := s.routes(broadcaster, indexer, searcher)

	s.server = &http.Server{
		Addr:         cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// initIndexes walks the vault and builds initial indexes.
func (s *Server) initIndexes() error {
	if err := s.indexer.IndexFiles(s.cfg.VaultPath); err != nil {
		log.Printf("warning: failed to index vault: %v", err)
	}
	s.searcher.Rebuild()
	return nil
}

// routes sets up the HTTP router.
func (s *Server) routes(broadcaster *api.Broadcaster, idx *indexer.Indexer, sr *search.Searcher) http.Handler {
	h := api.NewHandlers(s.cfg.VaultPath, broadcaster, idx, sr, s.cfg, s.locks)

	mux := http.NewServeMux()

	// API routes
	// NOTE: Trailing slash is required for subtree matching in Go 1.22+ ServeMux.
	// /api/files matches only exact; /api/files/ matches /api/files/anything.
	// Register both for exact-list (/api/files) and subtree (/api/files/<path>).
	mux.HandleFunc("/api/files", h.FilesHandler())
	mux.HandleFunc("/api/files/", h.FilesHandler())
	mux.HandleFunc("/api/folders", h.FoldersHandler())
	mux.HandleFunc("/api/folders/", h.FoldersHandler())
	mux.HandleFunc("/api/rename", h.RenameHandler())
	mux.HandleFunc("/api/upload", h.UploadHandler())
	mux.HandleFunc("/api/download/", h.DownloadHandler())
	mux.HandleFunc("/api/search", h.SearchHandler())
	mux.HandleFunc("/api/backlinks/", h.BacklinksHandler())
	mux.HandleFunc("/api/graph", h.GraphHandler())
	mux.HandleFunc("/api/config", h.ConfigHandler())
	mux.HandleFunc("/api/tree", h.TreeHandler())
	mux.HandleFunc("/api/locks", h.LockHandler())

	// SSE endpoint
	mux.HandleFunc("/events", broadcaster.SSEHandler())

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// MCP SSE endpoints
	mux.Handle("/mcp/sse", s.mcpServer.HandleSSE())
	mux.Handle("/mcp/message", s.mcpServer.HandleMessage())

	// API Documentation (Scalar)
	mux.HandleFunc("/docs", s.serveScalarUI())
	mux.HandleFunc("/docs/openapi.yaml", s.serveOpenAPISpec())

	// Static files (frontend)
	if !s.cfg.DisableUI {
		mux.HandleFunc("/", s.staticHandler())
	}

	return mux
}

// staticHandler serves the embedded frontend static files.
func (s *Server) staticHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "" {
			path = "index.html"
		} else {
			path = path[1:] // strip leading /
		}

		// Check if file exists in embedded FS
		f, err := static.FS().Open(path)
		if err != nil {
			// SPA route — serve index.html
			if path != "index.html" {
				f2, err2 := static.FS().Open("index.html")
				if err2 != nil {
					http.NotFound(w, r)
					return
				}
				f2.Close()
				path = "index.html"
			} else {
				http.NotFound(w, r)
				return
			}
		} else {
			f.Close()
		}

		data, err := static.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", staticMimeType("/"+path))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

// staticMimeType returns the correct MIME type for a static file.
func staticMimeType(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			ext := path[i:]
			switch ext {
			case ".html":
				return "text/html; charset=utf-8"
			case ".css":
				return "text/css; charset=utf-8"
			case ".js":
				return "application/javascript; charset=utf-8"
			case ".json":
				return "application/json"
			case ".png":
				return "image/png"
			case ".jpg", ".jpeg":
				return "image/jpeg"
			case ".svg":
				return "image/svg+xml"
			case ".ico":
				return "image/x-icon"
			case ".woff":
				return "font/woff"
			case ".woff2":
				return "font/woff2"
			case ".ttf":
				return "font/ttf"
			default:
				return "application/octet-stream"
			}
		}
	}
	// No extension — default to HTML for SPA root
	if path == "index.html" || path == "" {
		return "text/html; charset=utf-8"
	}
	return "application/octet-stream"
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := s.cfg.Host + ":" + strconv.Itoa(s.cfg.Port)
	log.Printf("gomd: starting server on %s", addr)
	log.Printf("gomd: vault at %s", s.cfg.VaultPath)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	if s.watcher != nil {
		s.watcher.Stop()
	}
	return s.server.Close()
}
