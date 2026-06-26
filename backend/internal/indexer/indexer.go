// Package indexer provides wiki link parsing, backlink computation, and in-memory indexing.
package indexer

import (
	"bytes"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Link represents a wiki link reference within a file.
type Link struct {
	Raw      string `json:"raw"`      // raw wiki link text, e.g. "doc#heading|alias"
	File     string `json:"file"`     // resolved filename (without extension)
	Heading  string `json:"heading"`  // optional heading reference
	Alias    string `json:"alias"`    // optional display alias
	Outbound bool   `json:"outbound"` // is this file linking to something?
}

// FileIndex holds the indexed data for a single file.
type FileIndex struct {
	Path      string   // relative path from vault
	Title     string   // from frontmatter or first heading
	Tags      []string // from frontmatter
	Body      string   // stripped body text for search
	Links     []Link   // outbound wiki links
	Backlinks []Link   // files that link to this one
	Frontmatter map[string]interface{}
}

// Indexer maintains the in-memory index of all vault files.
type Indexer struct {
	mu       sync.RWMutex
	vaultPath string    // used by Rebuild
	files    map[string]*FileIndex // path → index
	folders  map[string]struct{} // relative path → empty struct
	inverted map[string]map[string]struct{} // term → set of file paths
	tags     map[string]map[string]struct{} // tag → set of file paths
	links    map[string][]Link    // path → outbound links
}

// NewIndexer creates a new Indexer.
func NewIndexer() *Indexer {
	return &Indexer{
		files:    make(map[string]*FileIndex),
		folders:  make(map[string]struct{}),
		inverted: make(map[string]map[string]struct{}),
		tags:     make(map[string]map[string]struct{}),
		links:    make(map[string][]Link),
	}
}

// Rebuild walks the vault directory and rebuilds all indexes.
func (idx *Indexer) Rebuild() {
	if idx.vaultPath == "" {
		return
	}
	idx.IndexFiles(idx.vaultPath)
}

// IndexFiles indexes all markdown files in the given directory.
func (idx *Indexer) IndexFiles(vaultPath string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Store vaultPath for Rebuild() calls from watcher
	idx.vaultPath = vaultPath

	// Clear old indexes
	idx.files = make(map[string]*FileIndex)
	idx.folders = make(map[string]struct{})
	idx.inverted = make(map[string]map[string]struct{})
	idx.tags = make(map[string]map[string]struct{})
	idx.links = make(map[string][]Link)

	// Walk the vault using filepath.WalkDir (Glob ** doesn't work in Go)
	err := filepath.WalkDir(vaultPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors, continue walking
		}
		
		rel, err := filepath.Rel(vaultPath, path)
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if rel != "." && rel != "" {
				idx.folders[rel] = struct{}{}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := readFileContent(path)
		if err != nil {
			log.Printf("indexer: failed to read %s: %v", path, err)
			return nil
		}

		fi, err := idx.indexFile(rel, data)
		if err != nil {
			log.Printf("indexer: failed to index %s: %v", rel, err)
			return nil
		}

		idx.files[rel] = fi
		return nil
	})
	if err != nil {
		return err
	}

	// Compute backlinks (second pass)
	for path, fi := range idx.files {
		for _, link := range fi.Links {
			target := link.File
			if !strings.HasSuffix(target, ".md") {
				target += ".md"
			}

			// Find matching target in idx.files (supports subdirectories)
			for fileKey, targetIndex := range idx.files {
				if fileKey == target || strings.HasSuffix(fileKey, "/"+target) {
					if fileKey != path {
						backlink := link
						backlink.File = path
						backlink.Outbound = false
						targetIndex.Backlinks = append(targetIndex.Backlinks, backlink)
					}
					break // Stop after first match (assumes unique basenames for wiki links)
				}
			}
		}
	}

	log.Printf("indexer: indexed %d files", len(idx.files))
	return nil
}

// GetFileIndex returns the index for a file by path.
func (idx *Indexer) GetFileIndex(path string) *FileIndex {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.files[path]
}

// GetBacklinks returns the backlinks for a file.
func (idx *Indexer) GetBacklinks(path string) []Link {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	fi := idx.files[path]
	if fi == nil {
		return nil
	}
	
	result := make([]Link, len(fi.Backlinks))
	copy(result, fi.Backlinks)
	return result
}

// GetLinks returns the outbound links from a file.
func (idx *Indexer) GetLinks(path string) []Link {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.links[path]
}

// GetAllPaths returns all indexed file paths.
func (idx *Indexer) GetAllPaths() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	paths := make([]string, 0, len(idx.files))
	for p := range idx.files {
		paths = append(paths, p)
	}
	return paths
}

// GetAllFolders returns all indexed directory paths.
func (idx *Indexer) GetAllFolders() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	paths := make([]string, 0, len(idx.folders))
	for p := range idx.folders {
		paths = append(paths, p)
	}
	return paths
}

// GetFileCount returns the number of indexed files.
func (idx *Indexer) GetFileCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.files)
}

// GetTagFiles returns all file paths with the given tag.
func (idx *Indexer) GetTagFiles(tag string) map[string]struct{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if s, ok := idx.tags[tag]; ok {
		return s
	}
	return nil
}

// Search searches the index for the given query.
func (idx *Indexer) Search(query string) map[string]*FileIndex {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if query == "" {
		return idx.files
	}

	terms := tokenize(query)
	results := make(map[string]*FileIndex)

	for _, term := range terms {
		if paths, ok := idx.inverted[term]; ok {
			for p := range paths {
				results[p] = idx.files[p]
			}
		}
	}

	return results
}

// indexFile parses a single markdown file and returns its index.
func (idx *Indexer) indexFile(rel string, data []byte) (*FileIndex, error) {
	fi := &FileIndex{
		Path:        rel,
		Frontmatter: make(map[string]interface{}),
	}

	// Parse frontmatter
	body, fm, err := parseFrontmatter(data)
	if err != nil {
		log.Printf("indexer: frontmatter parse error: %v", err)
	}
	fi.Frontmatter = fm
	if title, ok := fm["title"].(string); ok && title != "" {
		fi.Title = title
	}
	if tags, ok := fm["tags"].([]interface{}); ok {
		for _, t := range tags {
			if ts, ok := t.(string); ok {
				fi.Tags = append(fi.Tags, ts)
			}
		}
	}

	// Parse wiki links
	fi.Links = ParseWikiLinks(string(body))

	// Build body text for search
	fi.Body = extractBodyText(body)

	// Build inverted index
	for _, term := range tokenize(fi.Body) {
		if idx.inverted[term] == nil {
			idx.inverted[term] = make(map[string]struct{})
		}
		idx.inverted[term][rel] = struct{}{}
	}

	// Build tag index
	for _, tag := range fi.Tags {
		if idx.tags[tag] == nil {
			idx.tags[tag] = make(map[string]struct{})
		}
		idx.tags[tag][rel] = struct{}{}
	}

	// Store outbound links
	idx.links[rel] = fi.Links

	return fi, nil
}

// ParseWikiLinks extracts all [[...]] references from markdown text.
func ParseWikiLinks(content string) []Link {
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	var links []Link
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		raw := m[1]
		link := Link{
			Raw:      "[[" + raw + "]]",
			Outbound: true,
		}

		// Handle alias: "file.md|alias"
		aliasParts := strings.SplitN(raw, "|", 2)
		target := aliasParts[0]
		if len(aliasParts) == 2 {
			link.Alias = aliasParts[1]
		}

		// Handle heading: "file.md#heading"
		hashParts := strings.SplitN(target, "#", 2)
		link.File = hashParts[0]
		if len(hashParts) > 1 {
			link.Heading = hashParts[1]
		}

		links = append(links, link)
	}
	return links
}

// parseFrontmatter extracts YAML frontmatter and returns body bytes + parsed map.
func parseFrontmatter(data []byte) ([]byte, map[string]interface{}, error) {
	fm := make(map[string]interface{})

	if !bytes.HasPrefix(data, []byte("---\n")) {
		return data, fm, nil
	}

	end := bytes.Index(data[3:], []byte("\n---\n"))
	if end == -1 {
		return data, fm, nil
	}

	fmBytes := data[3 : 3+end]
	if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
		return data, fm, err
	}

	body := data[3+end+4:]
	return body, fm, nil
}

// extractBodyText extracts plain text from markdown for search indexing.
func extractBodyText(body []byte) string {
	// Strip common markdown syntax for a simple text extraction
	text := string(body)
	// Remove headings
	text = regexp.MustCompile(`^#{1,6}\s+.*$`).ReplaceAllString(text, "")
	// Remove wiki links but keep text
	text = regexp.MustCompile(`\[\[([^\]]+)\]\]`).ReplaceAllStringFunc(text, func(m string) string {
		inner := m[2 : len(m)-2]
		if pipe := strings.IndexByte(inner, '|'); pipe >= 0 {
			return inner[pipe+1:]
		}
		return inner
	})
	// Remove images
	text = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]+\)`).ReplaceAllString(text, "")
	// Remove link syntax
	text = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`).ReplaceAllString(text, "$1")
	// Remove bold/italic markers
	text = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`).ReplaceAllStringFunc(text, func(m string) string {
		return regexp.MustCompile(`[*]`).ReplaceAllString(m, "")
	})
	text = regexp.MustCompile(`_{1,3}([^_]+)_{1,3}`).ReplaceAllStringFunc(text, func(m string) string {
		return regexp.MustCompile(`[_]`).ReplaceAllString(m, "")
	})
	// Remove code blocks (triple backtick blocks)
	codeRe := regexp.MustCompile("(?s)```[^`]*```")
	text = codeRe.ReplaceAllString(text, "")
	text = regexp.MustCompile("`[^`]+`").ReplaceAllString(text, "")
	// Remove blockquotes
	text = regexp.MustCompile(`^>\s+.*$`).ReplaceAllString(text, "")
	// Remove horizontal rules
	text = regexp.MustCompile(`^[-*_]{3,}$`).ReplaceAllString(text, "")
	// Remove HTML comments
	text = regexp.MustCompile(`<!--[\s\S]*?-->`).ReplaceAllString(text, "")
	// Normalize whitespace
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}

// tokenize splits text into lowercase, alphanumeric tokens.
func tokenize(text string) []string {
	words := strings.Fields(text)
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, w)
		if len(w) > 1 {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func readFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}
