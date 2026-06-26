# GoMD Development Guidelines

## Project Overview
GoMD is a self-hosted web app for managing a folder of markdown files with backlinks, graph view, full-text search, and mermaid support.

## Architecture
- **Backend**: Go, single binary, REST API + SSE for live updates
- **Frontend**: React + Vite, SPA, client-side routing
- **Data**: Plain .md files on disk, YAML frontmatter, in-memory indexes

## Backend (Go)

### File Structure
```
backend/
  internal/
    api/        — HTTP handlers, middleware, SSE broadcaster
    server/     — server setup, config loading, router
    fsnotify/   — file watcher, debounce, event dispatch
    indexer/    — wiki link parser, backlink engine
    graph/      — graph node/edge builder from link data
    search/     — inverted index, fuzzy search, tag filtering
  cmd/
    gomd/       — main.go, config parsing, server bootstrap
  go.mod
```

### Concurrency Model
- One fsnotify goroutine per vault path
- Indexer runs as a single goroutine (serializes to avoid races)
- SSE broadcaster: fan-out to connected clients via channels
- API handlers are stateless (read indexes, serve files)

### SSE Implementation
- Use `http.Server` with `Flusher` interface
- Each SSE client gets its own goroutine writing to the response
- Use a channel-based broadcaster pattern: `broadcaster.AddClient() → client receives events`
- Events: `file_change`, `file_deleted`, `file_created`, `index_ready`, `config_changed`

### Error Handling
- Vault path validation: reject paths outside the configured vault
- File operations: 404 for missing files, 400 for invalid paths
- Index errors: log and continue, don't crash the server

## Frontend (React + Vite)

### File Structure
```
frontend/src/
  components/
    FileTree/       — collapsible directory tree
    Editor/         — Monaco editor with markdown syntax
    Preview/        — rendered markdown (markdown-it + plugins)
    Graph/          — D3/cytoscape force-directed graph
    Search/         — search bar + results list
    Backlinks/      — sidebar panel showing backlinks
    Settings/       — theme toggle, vault config
  pages/
    App/            — main layout (sidebar + content)
    GraphView/      — full-page graph
    SearchResults/  — search results page
  hooks/
    useSSE/         — EventSource wrapper with reconnect
    useVault/       — file tree state + cache
    useGraph/       — graph data + interactions
  lib/
    markdown/       — markdown-it config, plugins (mermaid, math, links)
    api/            — fetch wrappers for all API endpoints
  App.tsx           — routing, providers
  main.tsx          — entry point
```

### Key Design Decisions
- Use `markdown-it` (not `marked`) for better plugin ecosystem and CommonMark compliance
- Monaco Editor for the editor pane (same as VS Code, familiar)
- D3 force-directed for graph view (more control than react-force-graph)
- KaTeX for math rendering (inline `$$` and display `$$`)
- Mermaid via `mermaid` package with `init` call in useEffect
- Client-side routing with `react-router-dom`

### State Management
- React Query (`@tanstack/react-query`) for server state (files, search, graph data)
- Local state for UI (sidebar open/close, theme, editor content)
- SSE hook manages real-time file updates

## Naming Conventions
- Go: PascalCase for exported, camelCase for unexported
- React: PascalCase for components, camelCase for hooks/functions
- Files: kebab-case for Go, PascalCase for React components
- API paths: kebab-case (`/api/file-tree`, `/api/search-results`)

## Testing
- Go: standard `testing` package, table-driven tests
- Frontend: Vitest for unit, Playwright for E2E (if time permits)

## Build & Run
```bash
# Backend only
cd backend && go run ./cmd/gomd

# Frontend dev
cd frontend && npm run dev

# Frontend build
cd frontend && npm run build

# Full build (backend embeds frontend dist)
cd backend && go build -o gomd ./cmd/gomd
./gomd --vault /path/to/vault --port 3000
```

## Progress Tracking
- Mark completed requirements in REQUIREMENTS.md
- Create a new branch per major feature
- Commit messages: `type: description` (feat:, fix:, refactor:, etc.)
