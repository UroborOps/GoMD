package gitsync

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nroitero/gomd/backend/internal/config"
)

// Start initializes the git repository and starts the background sync worker.
func Start(cfg *config.Config) {
	if !cfg.GitEnabled {
		return
	}

	if cfg.GitRemote == "" {
		log.Println("gitsync: enabled but no remote configured. Please set GOMD_GIT_REMOTE.")
		return
	}

	log.Printf("gitsync: initializing sync worker for %s (interval: %dm)", cfg.VaultPath, cfg.GitSyncInterval)

	// Ensure it's a git repo
	if err := initRepo(cfg.VaultPath, cfg.GitRemote); err != nil {
		log.Printf("gitsync error: failed to initialize repo: %v", err)
		return
	}

	// Do initial sync
	sync(cfg.VaultPath)

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.GitSyncInterval) * time.Minute)
		for range ticker.C {
			sync(cfg.VaultPath)
		}
	}()
}

func initRepo(vaultPath, remoteURL string) error {
	gitDir := filepath.Join(vaultPath, ".git")
	
	// Check if already a repo
	if !isDir(gitDir) {
		log.Println("gitsync: initializing new git repository...")
		if err := runGit(vaultPath, "init"); err != nil {
			return err
		}
		
		// Configure dummy user for automated commits
		runGit(vaultPath, "config", "user.name", "GoMD AutoSync")
		runGit(vaultPath, "config", "user.email", "autosync@gomd.local")
	}

	// Set/Update remote origin
	if err := runGit(vaultPath, "remote", "add", "origin", remoteURL); err != nil {
		// If remote already exists, set its URL instead
		runGit(vaultPath, "remote", "set-url", "origin", remoteURL)
	}

	// Make sure we're on a branch, e.g., main
	runGit(vaultPath, "checkout", "-B", "main")

	return nil
}

func sync(vaultPath string) {
	// Add all changes
	runGit(vaultPath, "add", "-A")

	// Commit if there are changes
	err := runGit(vaultPath, "commit", "-m", "Auto-sync from GoMD")
	hasChanges := err == nil

	// Pull with rebase
	pullErr := runGit(vaultPath, "pull", "origin", "main", "--rebase")
	if pullErr != nil {
		log.Printf("gitsync warning: failed to pull: %v", pullErr)
	}

	// Push
	if err := runGit(vaultPath, "push", "origin", "main"); err != nil {
		log.Printf("gitsync warning: failed to push: %v", err)
	} else if hasChanges || pullErr == nil {
		// Only log success if we had changes to push or successfully pulled
		// To avoid spamming, we could just log when we push our changes
		if hasChanges {
			log.Println("gitsync: successfully synced local changes to remote.")
		}
	}
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %s", strings.Join(args, " "), stderr.String())
	}
	return nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
