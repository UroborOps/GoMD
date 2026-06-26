// Package graph builds graph data from link information.
package graph

import (
	"path/filepath"
)

// Node represents a graph node (file).
type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group,omitempty"` // tag category for coloring
	Links int    `json:"links"`
}

// Edge represents a connection between two nodes.
type Edge struct {
	From string `json:"source"`
	To   string `json:"target"`
}

// Graph represents the complete graph data structure.
type Graph struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// Build constructs a graph from the indexer data.
func Build(fileCount int, getAllPaths func() []string, getBacklinks func(path string) []string, getLinks func(path string) []LinkInfo) (*Graph, error) {
	g := &Graph{
		Nodes: make([]*Node, 0, fileCount),
		Edges: make([]*Edge, 0),
	}

	// Build nodes
	paths := getAllPaths()
	
	// Create a map to resolve link targets (e.g. "file.md") to actual node IDs (e.g. "folder/file.md")
	pathMap := make(map[string]string)
	for _, p := range paths {
		pathMap[p] = p
		pathMap[filepath.Base(p)] = p
	}

	for _, path := range paths {
		labels := filepath.Base(path)
		if idx := len(labels); idx > 0 && labels[idx-1] == 'd' && labels[idx-2] == 'm' && labels[idx-3] == '.' {
			labels = labels[:idx-3]
		}

		backlinks := getBacklinks(path)
		links := getLinks(path)

		node := &Node{
			ID:    path,
			Label: labels,
			Links: len(backlinks) + len(links),
		}

		// Determine group from first tag
		if len(links) > 0 {
			// Use tag from frontmatter if available — skip for now, use file type
			node.Group = "file"
		}

		g.Nodes = append(g.Nodes, node)

		// Add edges
		for _, link := range links {
			target := link.File
			// Ensure target has .md extension for lookup
			if filepath.Ext(target) != ".md" {
				target = target + ".md"
			}
			
			// Resolve to actual node ID
			if resolved, ok := pathMap[target]; ok {
				g.Edges = append(g.Edges, &Edge{
					From: path,
					To:   resolved,
				})
			}
		}
	}

	return g, nil
}

// LinkInfo represents an outbound link from a file.
type LinkInfo struct {
	File   string
	Heading string
	Alias  string
}
