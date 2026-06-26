package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/nroitero/gomd/backend/internal/indexer"
	"github.com/nroitero/gomd/backend/internal/locks"
	"github.com/nroitero/gomd/backend/internal/search"
)

// Server wraps the MCP SDK server and connects it to GoMD state.
type Server struct {
	sse       *server.SSEServer
	vaultPath string
	indexer   *indexer.Indexer
	searcher  *search.Searcher
	locks     *locks.Manager
}

// New creates a new GoMD MCP Server.
func New(vaultPath string, idx *indexer.Indexer, sr *search.Searcher, lm *locks.Manager) *Server {
	// Create core MCP server instance
	mcpServer := server.NewMCPServer(
		"gomd",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s := &Server{
		vaultPath: vaultPath,
		indexer:   idx,
		searcher:  sr,
		locks:     lm,
	}

	// Register read_file
	readTool := mcp.NewTool("read_file",
		mcp.WithDescription("Read the contents of a markdown file in the vault."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path to the file (e.g., 'notes/idea.md')")),
	)
	mcpServer.AddTool(readTool, s.handleReadFile)

	// Register write_file
	writeTool := mcp.NewTool("write_file",
		mcp.WithDescription("Create or update a markdown file in the vault."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path to the file")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Markdown content to write")),
	)
	mcpServer.AddTool(writeTool, s.handleWriteFile)

	// Register search
	searchTool := mcp.NewTool("search",
		mcp.WithDescription("Search the vault for files containing a specific query."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Text to search for")),
	)
	mcpServer.AddTool(searchTool, s.handleSearch)

	// Register list_backlinks
	backlinksTool := mcp.NewTool("list_backlinks",
		mcp.WithDescription("List all files that link to a given file."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path to the file to check for backlinks")),
	)
	mcpServer.AddTool(backlinksTool, s.handleBacklinks)

	// Create SSE transport server wrapper
	s.sse = server.NewSSEServer(mcpServer)

	return s
}

// HandleSSE provides the HTTP handler for new SSE connections.
func (s *Server) HandleSSE() http.Handler {
	return s.sse.SSEHandler()
}

// HandleMessage provides the HTTP handler for receiving MCP JSON-RPC messages.
func (s *Server) HandleMessage() http.Handler {
	return s.sse.MessageHandler()
}

// -- Tool Handlers --

func (s *Server) handleReadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("missing path argument"), nil
	}

	fullPath := filepath.Join(s.vaultPath, filepath.Clean(path))
	if !strings.HasPrefix(fullPath, s.vaultPath) {
		return mcp.NewToolResultError("path outside vault"), nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return mcp.NewToolResultError("file not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleWriteFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, errPath := request.RequireString("path")
	content, errContent := request.RequireString("content")
	if errPath != nil || errContent != nil {
		return mcp.NewToolResultError("missing path or content argument"), nil
	}

	if s.locks.IsLocked(path) {
		return mcp.NewToolResultError("file is locked"), nil
	}

	fullPath := filepath.Join(s.vaultPath, filepath.Clean(path))
	if !strings.HasPrefix(fullPath, s.vaultPath) {
		return mcp.NewToolResultError("path outside vault"), nil
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create directories: %v", err)), nil
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to write file: %v", err)), nil
	}

	// Note: indexer rebuild and broadcasting is handled by fsnotify file watcher
	return mcp.NewToolResultText("File written successfully."), nil
}

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("missing query argument"), nil
	}

	results, err := s.searcher.Search(query, 10)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No results found."), nil
	}

	var sb strings.Builder
	for i, res := range results {
		sb.WriteString(fmt.Sprintf("%d. %s (Title: %s)\n", i+1, res.Path, res.Title))
		sb.WriteString(fmt.Sprintf("   Preview: %s\n\n", res.Content))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleBacklinks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("missing path argument"), nil
	}

	backlinks := s.indexer.GetBacklinks(filepath.Clean(path))
	if len(backlinks) == 0 {
		return mcp.NewToolResultText("No backlinks found."), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Backlinks for %s:\n", path))
	for _, link := range backlinks {
		if link.Heading != "" {
			sb.WriteString(fmt.Sprintf("- %s (heading: %s)\n", link.File, link.Heading))
		} else {
			sb.WriteString(fmt.Sprintf("- %s\n", link.File))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}
