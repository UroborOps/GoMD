// Package fsnotify provides file watching with debounce and event dispatch.
package fsnotify

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nroitero/gomd/backend/internal/api"
	"github.com/nroitero/gomd/backend/internal/indexer"
)

// Watcher monitors the vault directory for file changes and dispatches events.
type Watcher struct {
	vaultPath   string
	broadcaster *api.Broadcaster
	indexer     *indexer.Indexer
	notify      *fsnotify.Watcher
	mu          sync.Mutex
	stopped     bool
}

// NewWatcher creates a new file watcher for the given vault path.
func NewWatcher(vaultPath string, broadcaster *api.Broadcaster, idx *indexer.Indexer) (*Watcher, error) {
	w := &Watcher{
		vaultPath:   vaultPath,
		broadcaster: broadcaster,
		indexer:     idx,
	}

	notify, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w.notify = notify

	// Walk vault and add all directories
	if err := w.walkAndAdd(vaultPath); err != nil {
		w.notify.Close()
		return nil, err
	}

	go w.run()
	return w, nil
}

// walkAndAdd recursively adds all directories under vault to the watcher.
func (w *Watcher) walkAndAdd(path string) error {
	return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors, continue walking
		}
		if d.IsDir() {
			return w.notify.Add(p)
		}
		return nil
	})
}

// AddPath adds a single path to the watcher.
func (w *Watcher) AddPath(path string) {
	w.notify.Add(path)
}

// RemovePath removes a path from the watcher.
func (w *Watcher) RemovePath(path string) {
	w.notify.Remove(path)
}

// Run starts the watcher event loop.
func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.notify.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.notify.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

// handleEvent processes a single fsnotify event with debounce.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only care about .md files
	if !strings.HasSuffix(event.Name, ".md") {
		return
	}

	// Normalize path relative to vault
	rel, err := filepath.Rel(w.vaultPath, event.Name)
	if err != nil {
		log.Printf("watcher: failed to normalize path: %v", err)
		return
	}

	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	// Determine event type
	var eventType api.EventType
	switch {
	case event.Op&fsnotify.Write != 0, event.Op&fsnotify.Create != 0:
		eventType = api.EventFileChange
	case event.Op&fsnotify.Remove != 0, event.Op&fsnotify.Rename != 0:
		eventType = api.EventFileDeleted
	default:
		return
	}

	// Debounce: if multiple events in quick succession, wait for settle
	time.Sleep(100 * time.Millisecond)

	// Re-read to handle rename/source issues
	absPath, err := filepath.Abs(filepath.Join(w.vaultPath, rel))
	if err != nil {
		absPath = event.Name
	}

	// Re-check file still exists (in case of rapid delete)
	if eventType == api.EventFileChange {
		if _, err := w.checkFile(absPath); err != nil {
			// File might have been deleted, treat as delete
			eventType = api.EventFileDeleted
			rel, _ = filepath.Rel(w.vaultPath, absPath)
		}
	}

	// Broadcast event
	evt := api.NewEvent(eventType, rel, "")
	w.broadcaster.Broadcast(evt)

	// Trigger indexer rebuild (async, debounced)
	go w.indexer.Rebuild()
}

// checkFile verifies a file exists and returns its content.
func (w *Watcher) checkFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
	w.notify.Close()
}
