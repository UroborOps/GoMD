package locks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	mu       sync.RWMutex
	filePath string
	states   map[string]bool
}

func NewManager(vaultPath string) *Manager {
	m := &Manager{
		filePath: filepath.Join(vaultPath, ".gomd-locks.json"),
		states:   make(map[string]bool),
	}
	m.load()
	return m
}

func (m *Manager) load() {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := os.ReadFile(m.filePath)
	if err == nil {
		json.Unmarshal(data, &m.states)
	}
}

func (m *Manager) save() {
	data, err := json.MarshalIndent(m.states, "", "  ")
	if err == nil {
		os.WriteFile(m.filePath, data, 0644)
	}
}

// IsLocked checks if a path or any of its parents are explicitly locked.
func (m *Manager) IsLocked(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Clean path just in case
	path = filepath.Clean(path)
	if path == "." {
		path = ""
	}

	if path == "" {
		if state, ok := m.states[""]; ok {
			return state
		}
		return false
	}

	// Walk up the path to find the closest explicit lock state
	current := path
	for {
		if state, exists := m.states[current]; exists {
			return state
		}
		if current == "." || current == "" {
			break
		}
		current = filepath.Dir(current)
		if current == "." {
			current = ""
		}
	}
	
	if state, exists := m.states[""]; exists {
		return state
	}
	return false
}

// SetLock explicitly sets or unsets the lock for a given path.
func (m *Manager) SetLock(path string, locked bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path = filepath.Clean(path)
	if path == "." {
		path = ""
	}

	// If toggling the root lock, clear all specific sub-folder overrides
	// so the root lock strictly applies to everything.
	if path == "" {
		m.states = make(map[string]bool)
	}

	m.states[path] = locked
	m.save()
}

// GetAll returns a copy of the explicit lock states.
func (m *Manager) GetAll() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	copy := make(map[string]bool)
	for k, v := range m.states {
		copy[k] = v
	}
	return copy
}
