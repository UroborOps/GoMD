---
name: gomd-backend
description: Go backend development patterns for GoMD — file system operations, SSE, indexing, graph building, search
tags: [gomd, backend, go]
---

# GoMD Backend Skill

Go patterns for building the GoMD backend (Go, REST API, SSE, fsnotify, indexing).

## SSE Broadcaster Pattern

Use a channel-based broadcaster for fan-out to SSE clients:

```go
type Broadcaster struct {
    mu       sync.RWMutex
    clients  map[chan Event]struct{}
}

func (b *Broadcaster) AddClient() chan Event {
    ch := make(chan Event, 10)
    b.mu.Lock()
    b.clients[ch] = struct{}{}
    b.mu.Unlock()
    return ch
}

func (b *Broadcaster) RemoveClient(ch chan Event) {
    b.mu.Lock()
    delete(b.clients, ch)
    b.mu.Unlock()
    close(ch)
}

func (b *Broadcaster) Broadcast(evt Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for ch := range b.clients {
        select {
        case ch <- evt:
        default: // drop if client is slow
        }
    }
}
```

SSE handler:
```go
func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok { http.Error(w, "SSE not supported", 500); return }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    ch := s.broadcaster.AddClient()
    defer s.broadcaster.RemoveClient(ch)

    ctx := r.Context()
    for {
        select {
        case <-ctx.Done():
            return
        case evt := <-ch:
            data, _ := json.Marshal(evt)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
            flusher.Flush()
        }
    }
}
```

## File Watcher (fsnotify)

```go
func NewWatcher(vaultPath string, broadcaster *Broadcaster, indexer *Indexer) (*Watcher, error) {
    w := &Watcher{vaultPath: vaultPath, broadcaster: broadcaster, indexer: indexer}
    notify, err := fsnotify.NewWatcher()
    if err != nil { return nil, err }
    w.notify = notify

    // Walk vault and add all dirs
    filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, err error) error {
        if err == nil && d.IsDir() {
            notify.Add(path)
        }
        return nil
    })

    go w.run()
    return w, nil
}

func (w *Watcher) run() {
    for {
        select {
        case event, ok := <-w.notify.Events:
            if !ok { return }
            if strings.HasSuffix(event.Name, ".md") {
                evt := Event{Type: eventType(event.Op), Path: event.Name}
                w.broadcaster.Broadcast(evt)
                // debounce indexer rebuild
                go w.indexer.Rebuild()
            }
        case err, ok := <-w.notify.Errors:
            if !ok { return }
            log.Printf("watcher error: %v", err)
        }
    }
}
```

## Wiki Link Parser

```go
// ParseWikiLinks extracts all [[...]] references from markdown text
func ParseWikiLinks(content string) []Link {
    re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
    var links []Link
    for _, m := range re.FindAllStringSubmatch(content, -1) {
        raw := m[1]
        link := Link{Raw: raw}
        parts := strings.SplitN(raw, "#", 2)
        link.File = parts[0]
        if len(parts) > 1 { link.Heading = parts[1] }
        // handle alias: "file.md|alias"
        aliasParts := strings.SplitN(link.File, "|", 2)
        if len(aliasParts) == 2 { link.File, link.Alias = aliasParts[0], aliasParts[1] }
        links = append(links, link)
    }
    return links
}
```

## Indexer (In-Memory)

```go
type Indexer struct {
    mu       sync.RWMutex
    files    map[string]*FileIndex  // path → index
    inverted map[string]map[string]struct{} // term → set of file paths
    tags     map[string]map[string]struct{} // tag → set of file paths
}

func (idx *Indexer) Rebuild() {
    idx.mu.Lock()
    defer idx.mu.Unlock()

    // Walk vault, read all .md files, parse content
    // Clear old indexes
    // For each file:
    //   - Parse frontmatter (title, tags, aliases)
    //   - Parse body text → tokenize → add to inverted index
    //   - Parse wiki links → build link index
}
```

## Config

Use TOML for config:

```go
type Config struct {
    VaultPath string `toml:"vault_path"`
    Port      int    `toml:"port"`
    Theme     string `toml:"theme"`
    Host      string `toml:"host"`
}
```

Default config at `~/.gomd/config.yaml`. CLI flag `--config` to override.
