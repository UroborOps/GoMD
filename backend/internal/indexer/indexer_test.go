package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIndexFiles_FindsFilesRecursively(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-indexer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create test files in subdirectories (include body text, not just headings)
	files := map[string]string{
		"root.md":        "# Root\n\nThis is the root file content.",
		"sub/deep.md":    "# Deep\n\nDeep content here.",
		"sub/another.md": "# Another\n\nAnother file body text.",
	}
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	idx := NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	got := len(idx.GetAllPaths())
	if got != len(files) {
		t.Errorf("indexed %d files, want %d", got, len(files))
	}

	// Verify each file is indexed with body content
	for rel := range files {
		fi := idx.GetFileIndex(rel)
		if fi == nil {
			t.Errorf("GetFileIndex(%q) == nil, want non-nil", rel)
			continue
		}
		if fi.Body == "" {
			t.Errorf("GetFileIndex(%q).Body is empty, want non-empty", rel)
		}
	}
}

func TestIndexFiles_EmptyVault(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-indexer-empty-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	idx := NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	if len(idx.GetAllPaths()) != 0 {
		t.Errorf("expected 0 files in empty vault, got %d", len(idx.GetAllPaths()))
	}
}

func TestIndexFiles_NonMarkdownSkipped(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-indexer-skip-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a mix of markdown and non-markdown files
	if err := os.WriteFile(filepath.Join(dir, "file.md"), []byte("# MD"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	idx := NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	if got := len(idx.GetAllPaths()); got != 1 {
		t.Errorf("expected 1 .md file, got %d", got)
	}
}

func TestIndexFile_FrontmatterParsing(t *testing.T) {
	idx := NewIndexer()
	content := []byte(`---
title: My Title
tags: [test, foo]
---

Body text here.
`)
	fi, err := idx.indexFile("test.md", content)
	if err != nil {
		t.Fatalf("indexFile: %v", err)
	}

	if fi.Title != "My Title" {
		t.Errorf("Title = %q, want %q", fi.Title, "My Title")
	}
	if len(fi.Tags) != 2 || fi.Tags[0] != "test" || fi.Tags[1] != "foo" {
		t.Errorf("Tags = %v, want [test, foo]", fi.Tags)
	}
}

func TestIndexFile_WikiLinks(t *testing.T) {
	idx := NewIndexer()
	content := []byte(`# Test

See [[other.md]] for details and [[other]] for short.
Also [[sub/page.md]] in subdirectory.
`)
	fi, err := idx.indexFile("test.md", content)
	if err != nil {
		t.Fatalf("indexFile: %v", err)
	}

	if fi.Links == nil || len(fi.Links) == 0 {
		t.Fatalf("Links is empty, want at least 1 link")
	}
}

func TestGetBacklinks(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-backlinks-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// file_a.md links to file_b.md
	if err := os.WriteFile(filepath.Join(dir, "file_a.md"), []byte("# A\n\n[[file_b]]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// file_b.md is the target
	if err := os.WriteFile(filepath.Join(dir, "file_b.md"), []byte("# B\n"), 0644); err != nil {
		t.Fatal(err)
	}

	idx := NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	// file_b should have a backlink from file_a
	bl := idx.GetBacklinks("file_b.md")
	if len(bl) != 1 {
		t.Errorf("GetBacklinks('file_b.md') = %d backlinks, want 1: %v", len(bl), bl)
	}
	if len(bl) > 0 && bl[0].File != "file_a.md" {
		t.Errorf("GetBacklinks('file_b.md')[0] = %q, want 'file_a.md'", bl[0].File)
	}
}

func TestRebuild_UsesStoredVaultPath(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-rebuild-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	idx := NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	// Simulate clearing the index (like a file delete event)
	idx.mu.Lock()
	idx.files = make(map[string]*FileIndex)
	idx.mu.Unlock()

	if len(idx.GetAllPaths()) != 0 {
		t.Error("expected 0 files after clearing")
	}

	// Rebuild should re-index from the stored vault path
	idx.Rebuild()

	if got := len(idx.GetAllPaths()); got != 1 {
		t.Errorf("after Rebuild, got %d files, want 1", got)
	}
}

func TestGetTagFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-tagfiles-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// File with tags
	if err := os.WriteFile(filepath.Join(dir, "tagged.md"), []byte(`---
title: Tagged File
tags: [important, review]
---

# Tagged File
`), 0644); err != nil {
		t.Fatal(err)
	}
	// File with different tags
	if err := os.WriteFile(filepath.Join(dir, "other.md"), []byte(`---
title: Other File
tags: [other]
---

# Other File
`), 0644); err != nil {
		t.Fatal(err)
	}
	// File without tags
	if err := os.WriteFile(filepath.Join(dir, "plain.md"), []byte("# Plain"), 0644); err != nil {
		t.Fatal(err)
	}

	idx := NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	// Search for "important" tag
	paths := idx.GetTagFiles("important")
	if len(paths) != 1 {
		t.Fatalf("GetTagFiles('important') = %d files, want 1: %v", len(paths), paths)
	}
	if _, has := paths["tagged.md"]; !has {
		t.Errorf("GetTagFiles('important') missing 'tagged.md': %v", paths)
	}

	// Search for non-existent tag
	paths = idx.GetTagFiles("nonexistent")
	if len(paths) != 0 {
		t.Errorf("GetTagFiles('nonexistent') = %d files, want 0", len(paths))
	}

	// Search for tag shared across files
	if err := os.WriteFile(filepath.Join(dir, "reviewed.md"), []byte(`---
title: Reviewed File
tags: [review]
---

# Reviewed File
`), 0644); err != nil {
		t.Fatal(err)
	}
	// Need to re-index
	idx.Rebuild()

	paths = idx.GetTagFiles("review")
	if len(paths) != 2 {
		t.Errorf("GetTagFiles('review') = %d files, want 2: %v", len(paths), paths)
	}
}
