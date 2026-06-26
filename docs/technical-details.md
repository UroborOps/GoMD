# GoMD Technical Documentation

This document covers the technical architecture, deployment strategies, and configuration options for GoMD.

## Project Architecture

GoMD is designed as a single-binary application to make deployment as simple as possible.

- **Backend**: Go (REST API + Server-Sent Events)
- **Frontend**: React + Vite (embedded via `//go:embed` into the Go binary)
- **Data Layer**: Plain Markdown files on disk. No SQL database required.
- **Search & AI**: Qdrant (Vector DB) for RAG and semantic search.

### Project Structure

```text
gomd/
  backend/               — Go server
    internal/
      api/               — HTTP handlers, SSE broadcaster
      server/            — Server setup, config
      fsnotify/          — File watcher
      indexer/           — Wiki link parser, backlink engine
      graph/             — Graph node/edge builder
      search/            — Inverted index, fuzzy search
      s3backup/          — Automated S3 archiving
      gitsync/           — Git push/pull automation
      static/            — //go:embed frontend/dist
    cmd/gomd/            — main.go
  frontend/              — React + Vite SPA
    src/
      components/        — FileTree, Editor, Preview, Graph, Search, Backlinks
      pages/             — App, GraphView, SearchResults
      hooks/             — useSSE, useVault, useGraph
      lib/               — markdown-it config, API client
```

---

## Deployment Options

### 1. Minimal Docker Run (UI + Markdown only)

If you just want the Markdown editor and graph view without AI or backups:

```bash
docker run -p 3000:3000 -v ./vault:/app/vault gomd
```

### 2. Full Stack (Docker Compose)

The recommended way to run GoMD with all features (AI, Vector DB, S3 Backups, SSL) is using `docker-compose.yml`.

The stack includes:
- **GoMD**: The main application.
- **Traefik**: Reverse proxy for local domains (e.g., `*.fbi.com`).
- **Qdrant**: Vector database for AI memory and semantic search.
- **Llama.cpp**: Local LLM server for generating embeddings without sending data to OpenAI.
- **VersityGW**: High-performance, lightweight local S3 gateway for backups.

```bash
docker compose up -d
```

---

## Configuration

Configuration is managed via environment variables or a `config.yaml` file located in `~/.gomd/config.yaml` or `/app/config.yaml`.

### General Config

| Env Var | Description | Default |
|---|---|---|
| `GOMD_VAULT` | Path to your markdown folder | `./vault` |
| `GOMD_PORT` | HTTP Port | `3000` |
| `GOMD_DISABLE_UI` | Set to `true` to run headless (API only) | `false` |

### AI & Semantic Search (RAG)

GoMD uses Llama.cpp to generate text embeddings locally, which are stored in Qdrant for semantic search.

| Env Var | Description | Default |
|---|---|---|
| `GOMD_RAG_ENABLED` | Enable the AI Memory pipeline | `false` |
| `GOMD_EMBED_MODEL` | Model used for embeddings | `nomic-embed-text-v1.5.Q4_K_M.gguf` |
| `GOMD_QDRANT_URL` | Internal Qdrant URL | `http://qdrant:6333` |
| `GOMD_QDRANT_EXTERNAL_URL` | Public UI link to Qdrant Dashboard | `http://qdrant.fbi.com` |

### Automated Backups (S3 & VersityGW)

GoMD can automatically zip your entire vault and upload it to an S3-compatible backend.

| Env Var | Description | Default |
|---|---|---|
| `GOMD_S3_BACKUP_ENABLED` | Enable automated snapshots | `false` |
| `GOMD_S3_ENDPOINT` | S3 Server URL | `http://versitygw:7070` |
| `GOMD_S3_EXTERNAL_URL` | Public UI link to S3 Dashboard | `http://s3.fbi.com` |
| `GOMD_S3_BUCKET` | Bucket name | `gomd-backups` |
| `GOMD_S3_BACKUP_INTERVAL`| Backup frequency in minutes | `60` |
| `GOMD_S3_RETAIN_COUNT` | Number of old backups to keep | `7` |

### Git Auto-Sync

| Env Var | Description | Default |
|---|---|---|
| `GOMD_GIT_ENABLED` | Enable automatic Git commits & pushes | `false` |
| `GOMD_GIT_REMOTE` | URL of your remote Git repository | |
| `GOMD_GIT_SYNC_INTERVAL`| Sync frequency in minutes | `5` |

---

## Development Mode

To run the backend and frontend separately with hot-reloading:

```bash
# 1. Start backend (Terminal 1)
cd backend
go run ./cmd/gomd

# 2. Start frontend (Terminal 2)
cd frontend
npm install
npm run dev
```

Browse to `http://localhost:5173`. The Vite dev server will automatically proxy API requests to the Go backend running on port 3000.
