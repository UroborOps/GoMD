package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/UroborOps/GoMD/backend/internal/config"
	"github.com/UroborOps/GoMD/backend/internal/indexer"
)

func TestSearch_FindsMatchingFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-search-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create test files with different content
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# Hello World\n\nThis is file A."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("# Foo Bar\n\nThis is file B with different content."), 0644); err != nil {
		t.Fatal(err)
	}

	idx := indexer.NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	s := NewSearcher(idx, &config.Config{})
	s.Rebuild()

	// Search for "Hello"
	results, err := s.Search("Hello", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search('Hello') = %d results, want 1", len(results))
	}
	if results[0].Path != "a.md" {
		t.Errorf("Search('Hello') path = %q, want %q", results[0].Path, "a.md")
	}
	if results[0].Content == "" {
		t.Error("Search result Content is empty, want non-empty")
	}
	if results[0].Snippet == "" {
		t.Error("Search result Snippet is empty, want non-empty")
	}
}

func TestSearch_TagQuery(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-search-tag-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a file with tags
	content := `---
title: Tagged File
tags: [important, review]
---

# Important File
`
	if err := os.WriteFile(filepath.Join(dir, "tagged.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	idx := indexer.NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	s := NewSearcher(idx, &config.Config{})
	s.Rebuild()

	results, err := s.Search("tag:important", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search('tag:important') = %d results, want 1", len(results))
	}
	if results[0].Path != "tagged.md" {
		t.Errorf("tagged file path = %q, want %q", results[0].Path, "tagged.md")
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	idx := indexer.NewIndexer()
	s := NewSearcher(idx, &config.Config{})

	results, err := s.Search("", 10)
	if err != nil {
		t.Fatalf("Search(''): %v", err)
	}
	if results != nil {
		t.Error("Search('') returned non-nil results, want nil")
	}
}

func TestSearch_NoMatch(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-search-nomatch-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "file.md"), []byte("# Unique Title"), 0644); err != nil {
		t.Fatal(err)
	}

	idx := indexer.NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	s := NewSearcher(idx, &config.Config{})
	s.Rebuild()

	results, err := s.Search("xyznonexistent", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Search('xyznonexistent') = %d results, want 0", len(results))
	}
}

func TestSearch_MultipleTermMatch(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-search-multi-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// File with both terms
	if err := os.WriteFile(filepath.Join(dir, "both.md"), []byte("# Hello World\n\nBoth hello and world appear here."), 0644); err != nil {
		t.Fatal(err)
	}
	// File with only one term
	if err := os.WriteFile(filepath.Join(dir, "hello.md"), []byte("# Hello\n\nOnly hello."), 0644); err != nil {
		t.Fatal(err)
	}

	idx := indexer.NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	s := NewSearcher(idx, &config.Config{})
	s.Rebuild()

	results, err := s.Search("hello world", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Search('hello world') = %d results, want 2", len(results))
	}

	// both.md should rank higher
	if results[0].Path != "both.md" {
		t.Errorf("highest rank = %q, want %q (more terms match)", results[0].Path, "both.md")
	}
}
