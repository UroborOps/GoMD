package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/UroborOps/GoMD/backend/internal/indexer"
)

func buildGraph(dir string) (*Graph, error) {
	idx := indexer.NewIndexer()
	if err := idx.IndexFiles(dir); err != nil {
		return nil, err
	}

	return Build(
		idx.GetFileCount(),
		idx.GetAllPaths,
		func(path string) []string {
			links := idx.GetBacklinks(path)
			var bl []string
			for _, l := range links {
				bl = append(bl, l.File)
			}
			return bl
		},
		func(path string) []LinkInfo {
			links := idx.GetLinks(path)
			out := make([]LinkInfo, len(links))
			for i, l := range links {
				out[i] = LinkInfo{
					File:    l.File,
					Heading: l.Heading,
					Alias:   l.Alias,
				}
			}
			return out
		},
	)
}

func TestBuild_GraphNodes(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-graph-nodes-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub/c.md"), []byte("# C"), 0644); err != nil {
		t.Fatal(err)
	}

	g, err := buildGraph(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(g.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3", len(g.Nodes))
	}

	// Check labels are stripped of .md
	for _, n := range g.Nodes {
		if filepath.Ext(n.Label) == ".md" {
			t.Errorf("node Label %q still has .md extension", n.Label)
		}
	}
}

func TestBuild_GraphEdges(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-graph-edges-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// file_a.md links to file_b.md
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\n[[b]]"), 0644); err != nil {
		t.Fatal(err)
	}
	// file_b.md has no outbound links
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B\n\n[[c]]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.md"), []byte("# C"), 0644); err != nil {
		t.Fatal(err)
	}

	g, err := buildGraph(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// a -> b, b -> c = 2 edges
	if len(g.Edges) != 2 {
		t.Fatalf("edges = %d, want 2", len(g.Edges))
	}

	// Verify edge structure: from/to
	for _, e := range g.Edges {
		if e.From == "" {
			t.Error("edge From is empty")
		}
		if e.To == "" {
			t.Error("edge To is empty")
		}
	}

	// Verify specific edges
	foundAB := false
	foundBC := false
	for _, e := range g.Edges {
		if e.From == "a.md" && e.To == "b.md" {
			foundAB = true
		}
		if e.From == "b.md" && e.To == "c.md" {
			foundBC = true
		}
	}
	if !foundAB {
		t.Error("missing edge a.md -> b.md")
	}
	if !foundBC {
		t.Error("missing edge b.md -> c.md")
	}
}

func TestBuild_IsolatedFileNoEdges(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-graph-isolated-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "isolated.md"), []byte("# Isolated\n\nNo links here."), 0644); err != nil {
		t.Fatal(err)
	}

	g, err := buildGraph(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(g.Nodes) != 1 {
		t.Errorf("nodes = %d, want 1", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("edges = %d, want 0 for isolated file", len(g.Edges))
	}
}

func TestBuild_TargetExtensionAutoAdded(t *testing.T) {
	dir, err := os.MkdirTemp("", "gomd-graph-ext-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Links to "target" without .md extension
	if err := os.WriteFile(filepath.Join(dir, "source.md"), []byte("# Source\n\n[[target]]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "target.md"), []byte("# Target"), 0644); err != nil {
		t.Fatal(err)
	}

	g, err := buildGraph(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(g.Edges) != 1 {
		t.Fatalf("edges = %d, want 1", len(g.Edges))
	}

	// Target should have .md extension added
	e := g.Edges[0]
	if e.To != "target.md" {
		t.Errorf("edge To = %q, want %q (auto .md extension)", e.To, "target.md")
	}
}
