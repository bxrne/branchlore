package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bxrne/branchlore/internal/types"
)

func Load(configPath string) (*types.Config, error) {
	cfg := &types.Config{
		RepoPath:     types.DefaultRepoPath,
		WorktreeBase: types.DefaultWorktreeBase,
		DBFileName:   types.DefaultDBFileName,
		ServerAddr:   types.DefaultServerAddr,
		LogLevel:     types.DefaultLogLevel,
		Simulate:     false,
	}

	if configPath == "" {
		return cfg, nil
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(cfg *types.Config, configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func GetDefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".branchlore.json"
	}
	return filepath.Join(home, ".branchlore", "config.json")
}

func LoadFromEnv(cfg *types.Config) {
	if val := os.Getenv("BRANCHLORE_REPO_PATH"); val != "" {
		cfg.RepoPath = val
	}
	if val := os.Getenv("BRANCHLORE_WORKTREE_BASE"); val != "" {
		cfg.WorktreeBase = val
	}
	if val := os.Getenv("BRANCHLORE_DB_FILENAME"); val != "" {
		cfg.DBFileName = val
	}
	if val := os.Getenv("BRANCHLORE_SERVER_ADDR"); val != "" {
		cfg.ServerAddr = val
	}
	if val := os.Getenv("BRANCHLORE_LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}
	if val := os.Getenv("BRANCHLORE_SIMULATE"); val == "true" {
		cfg.Simulate = true
	}
}
