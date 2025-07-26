package repo

import (
	"context"
	"fmt"

	"github.com/bxrne/branchlore/internal/config"
	"github.com/bxrne/branchlore/internal/database"
	"github.com/bxrne/branchlore/internal/git"
	"github.com/bxrne/branchlore/internal/storage"
	"github.com/bxrne/branchlore/internal/types"
)

func QueryBranch(branch string, query string) (string, error) {
	cfg, err := config.Load("")
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	fs := storage.NewFileSystem(cfg)
	gitRepo := git.NewRepository(fs.GetRepoPath())

	if err := gitRepo.Init(fs.GetRepoPath()); err != nil {
		return "", fmt.Errorf("failed to initialize repository: %w", err)
	}

	worktreePath, err := gitRepo.CreateWorktree(branch)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	dbPath := fs.GetDBPath(worktreePath)
	db := database.NewSQLiteDB()

	if err := db.Open(dbPath); err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if !fs.PathExists(dbPath) {
		if err := db.InitSchema(); err != nil {
			return "", fmt.Errorf("failed to initialize schema: %w", err)
		}
	}

	result, err := db.Query(context.Background(), query)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	return formatQueryResult(result), nil
}

func formatQueryResult(result *types.QueryResult) string {
	if len(result.Columns) == 0 {
		return "No results"
	}

	output := ""
	for i, col := range result.Columns {
		if i > 0 {
			output += " | "
		}
		output += fmt.Sprintf("%-15s", col)
	}
	output += "\n"

	for i := 0; i < len(result.Columns); i++ {
		if i > 0 {
			output += "-+-"
		}
		output += "---------------"
	}
	output += "\n"

	for _, row := range result.Rows {
		for i, val := range row {
			if i > 0 {
				output += " | "
			}
			output += fmt.Sprintf("%-15v", val)
		}
		output += "\n"
	}

	output += fmt.Sprintf("\n%d rows\n", result.Count)
	return output
}
