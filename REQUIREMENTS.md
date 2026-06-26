# GoMD — Self-Hosted Markdown Knowledge Base

A lightweight, self-hosted web app for managing a folder of markdown files with backlinks, a graph view, full-text search, and mermaid support. Essentially "Obsidian with a web UI you own."

---

## Vision

- Store everything as plain `.md` files on disk
- Provide a beautiful, fast web UI for reading, editing, and navigating
- Automatically compute backlinks and build a knowledge graph
- Support rich rendering: code blocks, mermaid diagrams, LaTeX math
- Provide a Hermes plugin for agent-driven vault management

---

## Core Requirements

### 1. File System Integration

| # | Requirement | Details |
|---|---|---|
| 1.1 | Configurable vault path | Set via config file or CLI flag; default `~/.gomd/vault` |
| 1.2 | Auto-scan on startup | Walk the vault directory, index all `.md` files |
| 1.3 | Live file watching | Use `fsnotify` to detect file create/modify/delete events |
| 1.4 | Real-time updates | When a file changes, update the search index, link index, and broadcast to connected clients |
| 1.5 | File CRUD API | `GET /api/files`, `GET /api/files/:path`, `POST /api/files`, `PUT /api/files/:path`, `DELETE /api/files/:path` |
| 1.6 | Directory listing | `GET /api/folders/:path` returns subdirectories and `.md` files |
| 1.7 | Path handling | Normalize paths, reject traversal outside vault, support both URL-encoded and raw paths |

### 2. Link & Backlink Engine

| # | Requirement | Details |
|---|---|---|
| 2.1 | Wiki link parsing | Detect `[[filename]]`, `[[filename#heading]]`, `[[filename|alias]]` patterns |
| 2.2 | Auto-create pages | When `[[foo]]` is used but `foo.md` doesn't exist, offer to create it |
| 2.3 | Backlink computation | For each file, collect all other files that link to it |
| 2.4 | Outbound link detection | For each file, list all files it references |
| 2.5 | Heading-level links | `[[doc.md#section title]]` resolves to the correct anchor in the rendered page |
| 2.6 | Relative path resolution | `[[../other/note]]` resolves relative to the current file's directory |
| 2.7 | Broken link detection | Report links that reference non-existent files |

### 3. Search Engine

| # | Requirement | Details |
|---|---|---|
| 3.1 | Full-text search | Index file contents (strip frontmatter, preserve body text) |
| 3.2 | Filename search | Search matches filenames too |
| 3.3 | Fuzzy matching | Support partial/typo-tolerant search (e.g., `fzy` or similar) |
| 3.4 | Result snippets | Show context snippets with matched terms highlighted |
| 3.5 | Backlinks in results | Each search result shows which files link to it |
| 3.6 | Tag filtering | `tag:design` filters results to files with that tag |
| 3.7 | Real-time index | Search index updates live as files change |

### 4. Markdown Rendering

| # | Requirement | Details |
|---|---|---|
| 4.1 | CommonMark + GFM | Support standard markdown + GitHub Flavored Markdown (tables, strikethrough, task lists) |
| 4.2 | Frontmatter parsing | YAML frontmatter: title, date, tags, aliases, custom metadata |
| 4.3 | Code blocks | Syntax highlighting for common languages (use `highlight.js` or `prism`) |
| 4.4 | Mermaid diagrams | Parse and render mermaid code blocks as SVG |
| 4.5 | LaTeX math | `$inline$` and `$$block$$` rendering (KaTeX recommended) |
| 4.6 | Wiki links → clickable | `[[link]]` becomes clickable links that navigate within the SPA |
| 4.7 | Image embedding | Local `![alt](path/to/image.png)` renders images inline |
| 4.8 | Export/preview | `?view=preview` shows rendered markdown without editor chrome |

### 5. Graph View

| # | Requirement | Details |
|---|---|---|
| 5.1 | Force-directed graph | Nodes = files, edges = wiki links between them |
| 5.2 | Interactive | Drag nodes, zoom/pan, click to navigate |
| 5.3 | Cluster by tag | Color-code nodes by their `tags` frontmatter field |
| 5.4 | Highlight connected | Hovering a node highlights its neighbors |
| 5.5 | Orphan detection | Files with no inbound or outbound links are visually distinct |
| 5.6 | Performance | Virtualize rendering for >500 nodes; use WebGL backend if possible |

### 6. Frontend UX

| # | Requirement | Details |
|---|---|---|
| 6.1 | SPA architecture | React + Vite, client-side routing, no page reloads |
| 6.2 | File tree sidebar | Collapsible tree of folders and `.md` files, clickable |
| 6.3 | Editor + Preview modes | Split-pane editing (like Obsidian), or tabbed edit/preview |
| 6.4 | Live preview | Side-by-side rendered markdown updates as you type |
| 6.5 | Backlinks panel | Sidebar panel showing all files that link to the current file |
| 6.6 | Search panel | Search bar in sidebar, results clickable |
| 6.7 | Graph view page | Dedicated route `/graph` with full-page graph visualization |
| 6.8 | Dark/Light theme | Toggle or system preference detection |
| 6.9 | Responsive | Works on tablets (sidebar collapsible), mobile not primary target |
| 6.10 | Keyboard shortcuts | `Ctrl+P` search, `Ctrl+N` new file, `Esc` close panels |

### 7. API Design

```
GET    /api/files                    → List all files (tree or flat)
GET    /api/files/:path              → Get file content + frontmatter
POST   /api/files                    → Create new file (body: { path, content })
PUT    /api/files/:path              → Update file (body: { content })
DELETE /api/files/:path              → Delete file
GET    /api/folders/:path            → List directory contents
GET    /api/graph                    → Graph data (nodes + edges)
GET    /api/search?q=term            → Search results
GET    /api/backlinks/:path          → Backlinks for a file
GET    /api/config                   → Current server config
POST   /api/config                   → Update config

# Server-Sent Events (live updates)
GET    /events                       → SSE stream: file changes, index rebuilds
```

### 8. SSE Live Updates

| # | Requirement | Details |
|---|---|---|
| 8.1 | SSE endpoint | `/events` — standard `text/event-stream`, compatible with `EventSource` |
| 8.2 | Event types | `file_change` (content updated), `file_deleted`, `file_created`, `index_ready`, `config_changed` |
| 8.3 | Event payload | `{ "type": "file_change", "path": "notes/todo.md", "timestamp": "..." }` |
| 8.4 | Frontend hook | React `useSSE` hook that reconnects on disconnect, replays on reconnect |
| 8.5 | File update | On `file_change`, re-fetch the file content and update the editor/preview |
| 8.6 | Graph refresh | On index rebuild, push `index_ready` → frontend re-fetches graph data |

### 9. Hermes Integration

| # | Requirement | Details |
|---|---|---|
| 8.1 | MCP server | Implement the Model Context Protocol to expose vault operations to AI agents |
| 8.2 | Read files | Agent can read any file in the vault |
| 8.3 | Write files | Agent can create/update files |
| 8.4 | Search | Agent can search the vault via the search API |
| 8.5 | List backlinks | Agent can discover connections between files |
| 8.6 | Skill definition | Provide a Hermes skill file (`skills/gomd.md`) for easy integration |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Frontend (React + Vite)            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────┐ │
│  │ File Tree│ │  Editor  │ │  Preview │ │  Graph View│ │
│  │ Sidebar  │ │ (Monaco) │ │(marked)  │ │ (D3/cytoscape)│
│  └──────────┘ └──────────┘ └──────────┘ └────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────────────┐ │
│  │  Search  │ │Backlinks │ │     Theme / Settings     │ │
│  │  Panel   │ │  Panel   │ │                          │ │
│  └──────────┘ └──────────┘ └──────────────────────────┘ │
└──────────────────────────┬──────────────────────────────┘
                           │ HTTP / WebSocket
┌──────────────────────────▼──────────────────────────────┐
│               Backend (Go, single binary)               │
│  ┌────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │  API   │ │ File     │ │ Indexer  │ │ Search Engine│ │
│  │Router  │ │ Watcher  │ │(links +  │ │  (in-memory  │ │
│  │        │ │(fsnotify)│ │  backlink│ │   inverted   │ │
│  │        │ │          │ │  engine) │ │   index)     │ │
│  └────────┘ └──────────┘ └──────────┘ └──────────────┘ │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Graph Builder                       │   │
│  │  (aggregates link data → graph nodes/edges)      │   │
│  └──────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Hermes MCP Server                   │   │
│  │  (exposes vault as MCP tools/resources)          │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Data Layer                          │   │
│  │  Config: TOML/YAML                               │   │
│  │  Vault: plain .md files on disk                  │   │
│  └──────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

---

## Tech Stack

| Layer | Choice | Rationale |
|---|---|---|
| Backend | Go (single binary) | Fast, small footprint, clean concurrency with fsnotify |
| Frontend | React + Vite | Familiar pattern (Keres), fast dev server |
| Markdown | `marked` + plugins (React) | Mature ecosystem, plugin system |
| Code highlight | `highlight.js` | Lightweight, many languages |
| Mermaid | `mermaid` (browser render) | Official JS renderer |
| Math | KaTeX | Fast, lighter than MathJax |
| Editor | Monaco Editor | VS Code's editor, syntax highlighting, search/replace |
| Graph | D3 force-directed or `react-force-graph` | Well-documented, interactive |
| Search | Inverted index (Go, in-memory) | Fast for typical vault sizes (<10K files) |
| File watch | `fsnotify` (Go) | Standard, reliable |
| Config | TOML | Simple, human-readable |
| IPC | HTTP + WebSocket | REST for CRUD, WS for live sync |

---

## Non-Goals (v1)

- ✗ Real-time collaboration (multiplayer editing)
- ✗ Plugin system / extension API
- ✗ Mobile app
- ✗ Import from Obsidian vault (doable later)
- ✗ Version history / diff viewer
- ✗ PDF export

---

## Success Criteria for v1

1. Open a folder of `.md` files → they appear in the UI
2. Click a file → see rendered markdown with code blocks, mermaid, math
3. Click `[[some-link]]` → navigates to that file
4. See backlinks panel showing which files link to the current file
5. Search finds files by content and filename with fuzzy matching
6. Graph view shows all files as nodes with edges between them
7. Edit a file → changes save to disk and update in real-time
8. Hermes plugin can read/write/search the vault
