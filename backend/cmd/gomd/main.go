// Gomd is a self-hosted markdown knowledge base.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/UroborOps/GoMD/backend/internal/config"
	"github.com/UroborOps/GoMD/backend/internal/gitsync"
	"github.com/UroborOps/GoMD/backend/internal/server"
)

func main() {
	// Parse CLI flags
	configPath := flag.String("config", "", "Path to config file")
	vaultPath := flag.String("vault", "", "Path to vault directory")
	port := flag.Int("port", 0, "Server port")
	host := flag.String("host", "", "Server host")
	theme := flag.String("theme", "", "UI theme")
	
	gitEnabled := flag.Bool("git-enabled", false, "Enable Git Auto-Sync")
	gitRemote := flag.String("git-remote", "", "Git Remote URL")
	gitInterval := flag.Int("git-sync-interval", 0, "Git sync interval in minutes")
	
	ragEnabled := flag.Bool("rag-enabled", false, "Enable RAG semantic search")
	openaiURL := flag.String("openai-api-url", "", "OpenAI API URL")
	openaiKey := flag.String("openai-api-key", "", "OpenAI API Key")
	embedModel := flag.String("embed-model", "", "Embedding Model Name")
	qdrantURL := flag.String("qdrant-url", "", "Qdrant URL")
	qdrantKey := flag.String("qdrant-api-key", "", "Qdrant API Key")
	
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Override with CLI flags only if explicitly provided on command line
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "vault":
			cfg.VaultPath = *vaultPath
		case "port":
			cfg.Port = *port
		case "host":
			cfg.Host = *host
		case "theme":
			cfg.Theme = *theme
		case "git-enabled":
			cfg.GitEnabled = *gitEnabled
		case "git-remote":
			cfg.GitRemote = *gitRemote
		case "git-sync-interval":
			cfg.GitSyncInterval = *gitInterval
		case "rag-enabled":
			cfg.RAGEnabled = *ragEnabled
		case "openai-api-url":
			cfg.OpenAIURL = *openaiURL
		case "openai-api-key":
			cfg.OpenAIKey = *openaiKey
		case "embed-model":
			cfg.EmbedModel = *embedModel
		case "qdrant-url":
			cfg.QdrantURL = *qdrantURL
		case "qdrant-api-key":
			cfg.QdrantKey = *qdrantKey
		}
	})

	// Ensure vault exists
	if err := os.MkdirAll(cfg.VaultPath, 0755); err != nil {
		log.Fatalf("failed to create vault directory: %v", err)
	}

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\ngomd: shutting down...")
		if err := srv.Shutdown(); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	// Start Git Auto-Sync worker
	gitsync.Start(cfg)

	fmt.Printf("gomd: vault at %s\n", cfg.VaultPath)
	fmt.Printf("gomd: http://%s:%d\n", cfg.Host, cfg.Port)
	fmt.Println("gomd: press Ctrl+C to stop")

	if err := srv.Start(); err != nil && err.Error() != "http: Server closed" {
		log.Fatalf("server error: %v", err)
	}

	fmt.Println("gomd: stopped")
}
