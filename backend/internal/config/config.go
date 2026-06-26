// Package config handles loading and managing GoMD configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"io"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	VaultPath  string `yaml:"vault_path"`
	Port       int    `yaml:"port"`
	Host       string `yaml:"host"`
	Theme      string `yaml:"theme"`
	DisableUI  bool   `yaml:"disable_ui"`
	ConfigPath string `yaml:"-"` // not serialized, used internally

	// Git Auto-Sync
	GitEnabled      bool   `yaml:"git_enabled"`
	GitRemote       string `yaml:"git_remote"`
	GitSyncInterval int    `yaml:"git_sync_interval"`

	// RAG / Semantic Search
	RAGEnabled bool   `yaml:"rag_enabled"`
	OpenAIURL  string `yaml:"openai_api_url"`
	OpenAIKey  string `yaml:"openai_api_key"`
	EmbedModel string `yaml:"embed_model"`
	QdrantURL  string `yaml:"qdrant_url"`
	QdrantExternalURL string `yaml:"qdrant_external_url"`
	QdrantKey  string `yaml:"qdrant_api_key"`
	
	// S3 Backup
	S3BackupEnabled   bool   `yaml:"s3_backup_enabled"`
	S3Endpoint        string `yaml:"s3_endpoint"`
	S3ExternalURL     string `yaml:"s3_external_url"`
	S3Bucket          string `yaml:"s3_bucket"`
	S3AccessKey       string `yaml:"s3_access_key"`
	S3SecretKey       string `yaml:"s3_secret_key"`
	S3Region          string `yaml:"s3_region"`
	S3BackupInterval  int    `yaml:"s3_backup_interval"` // In minutes
	S3BackupRetainCount int  `yaml:"s3_backup_retain_count"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		VaultPath:       filepath.Join(home, ".gomd", "vault"),
		Port:            3000,
		Host:            "localhost",
		Theme:           "dark",
		GitSyncInterval: 5,
		OpenAIURL:       "https://api.openai.com/v1",
		EmbedModel:      "text-embedding-3-small",
	}
}

// Load reads config from a TOML file, then overrides with CLI flags.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath != "" {
		cfg.ConfigPath = configPath
	} else {
		// Check default locations
		home, _ := os.UserHomeDir()
		defaultPaths := []string{
			filepath.Join(home, ".gomd", "config.yaml"),
			"config.yaml",
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				cfg.ConfigPath = p
				break
			}
		}
	}

	if cfg.ConfigPath != "" {
		f, err := os.Open(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("open config %s: %w", cfg.ConfigPath, err)
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", cfg.ConfigPath, err)
		}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("decode config %s: %w", cfg.ConfigPath, err)
		}
	}

	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}

	// Override with environment variables
	if port := os.Getenv("GOMD_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil && p > 0 {
			cfg.Port = p
		}
	}
	if host := os.Getenv("GOMD_HOST"); host != "" {
		cfg.Host = host
	}
	if val := os.Getenv("GOMD_DISABLE_UI"); val == "true" || val == "1" {
		cfg.DisableUI = true
	}
	if vault := os.Getenv("GOMD_VAULT"); vault != "" {
		if !filepath.IsAbs(vault) {
			home, _ := os.UserHomeDir()
			cfg.VaultPath = filepath.Join(home, vault)
		} else {
			cfg.VaultPath = vault
		}
	}

	// Git Env Vars
	if val := os.Getenv("GOMD_GIT_ENABLED"); val == "true" || val == "1" {
		cfg.GitEnabled = true
	}
	if val := os.Getenv("GOMD_GIT_REMOTE"); val != "" {
		cfg.GitRemote = val
	}
	if val := os.Getenv("GOMD_GIT_SYNC_INTERVAL"); val != "" {
		if interval, err := strconv.Atoi(val); err == nil && interval > 0 {
			cfg.GitSyncInterval = interval
		}
	}

	// RAG Env Vars
	if val := os.Getenv("GOMD_RAG_ENABLED"); val == "true" || val == "1" {
		cfg.RAGEnabled = true
	}
	if val := os.Getenv("GOMD_OPENAI_API_URL"); val != "" {
		cfg.OpenAIURL = val
	}
	if val := os.Getenv("GOMD_OPENAI_API_KEY"); val != "" {
		cfg.OpenAIKey = val
	}
	if val := os.Getenv("GOMD_EMBED_MODEL"); val != "" {
		cfg.EmbedModel = val
	}
	if val := os.Getenv("GOMD_QDRANT_URL"); val != "" {
		cfg.QdrantURL = val
	}
	if val := os.Getenv("GOMD_QDRANT_EXTERNAL_URL"); val != "" {
		cfg.QdrantExternalURL = val
	}
	if val := os.Getenv("GOMD_QDRANT_API_KEY"); val != "" {
		cfg.QdrantKey = val
	}
	if val := os.Getenv("GOMD_S3_BACKUP_ENABLED"); val != "" {
		cfg.S3BackupEnabled = val == "true" || val == "1"
	}
	if val := os.Getenv("GOMD_S3_ENDPOINT"); val != "" {
		cfg.S3Endpoint = val
	}
	if val := os.Getenv("GOMD_S3_EXTERNAL_URL"); val != "" {
		cfg.S3ExternalURL = val
	}
	if val := os.Getenv("GOMD_S3_BUCKET"); val != "" {
		cfg.S3Bucket = val
	}
	if val := os.Getenv("GOMD_S3_ACCESS_KEY"); val != "" {
		cfg.S3AccessKey = val
	}
	if val := os.Getenv("GOMD_S3_SECRET_KEY"); val != "" {
		cfg.S3SecretKey = val
	}
	if val := os.Getenv("GOMD_S3_REGION"); val != "" {
		cfg.S3Region = val
	}
	if val := os.Getenv("GOMD_S3_BACKUP_INTERVAL"); val != "" {
		if iv, err := strconv.Atoi(val); err == nil {
			cfg.S3BackupInterval = iv
		}
	}
	if val := os.Getenv("GOMD_S3_RETAIN_COUNT"); val != "" {
		if rc, err := strconv.Atoi(val); err == nil {
			cfg.S3BackupRetainCount = rc
		}
	}

	// Resolve vault path relative to home if not absolute
	if !filepath.IsAbs(cfg.VaultPath) {
		home, _ := os.UserHomeDir()
		cfg.VaultPath = filepath.Join(home, cfg.VaultPath)
	}

	return cfg, nil
}
