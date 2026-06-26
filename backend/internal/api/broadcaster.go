// Package api provides the HTTP API and SSE broadcaster.
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// EventType represents an SSE event type.
type EventType string

const (
	EventFileChange  EventType = "file_change"
	EventFileDeleted EventType = "file_deleted"
	EventFileCreated EventType = "file_created"
	EventIndexReady  EventType = "index_ready"
	EventConfigChange EventType = "config_changed"
)

// Event represents an SSE event broadcast to clients.
type Event struct {
	Type      EventType `json:"type"`
	Path      string    `json:"path,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	Timestamp string    `json:"timestamp"`
}

// NewEvent creates an Event with the current timestamp.
func NewEvent(typ EventType, path, detail string) Event {
	return Event{
		Type:      typ,
		Path:      path,
		Detail:    detail,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Broadcaster fans out SSE events to multiple connected clients.
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[chan Event]struct{}
}

// NewBroadcaster creates a new Broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[chan Event]struct{}),
	}
}

// AddClient registers a new SSE client and returns its event channel.
func (b *Broadcaster) AddClient() chan Event {
	ch := make(chan Event, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// RemoveClient unregisters a client and closes its channel.
func (b *Broadcaster) RemoveClient(ch chan Event) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// Broadcast sends an event to all connected clients.
// Slow clients are dropped (non-blocking).
func (b *Broadcaster) Broadcast(evt Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- evt:
			// delivered
		default:
			// client is slow, drop
		}
	}
}

// SSEHandler serves the SSE endpoint.
func (b *Broadcaster) SSEHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := b.AddClient()
		defer b.RemoveClient(ch)

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-ch:
				data, err := json.Marshal(evt)
				if err != nil {
					log.Printf("broadcaster: marshal error: %v", err)
					continue
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
				flusher.Flush()
			}
		}
	}
}
