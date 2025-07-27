package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

type Manager struct {
	dataDir string
}

func NewManager(dataDir string) (*Manager, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &Manager{
		dataDir: dataDir,
	}, nil
}

func (m *Manager) InitDatabase(dbName string) error {
	dbPath := filepath.Join(m.dataDir, dbName)

	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Initialize git repository using git command
	initCmd := exec.Command("git", "init")
	initCmd.Dir = dbPath
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Create main database file
	mainDbFile := filepath.Join(dbPath, "main.db")
	if _, err := os.Create(mainDbFile); err != nil {
		return fmt.Errorf("failed to create main database file: %w", err)
	}

	// Configure git user for this repository
	configEmailCmd := exec.Command("git", "config", "user.email", "branchlore@local.dev")
	configEmailCmd.Dir = dbPath
	if err := configEmailCmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git email: %w", err)
	}

	configNameCmd := exec.Command("git", "config", "user.name", "branchlore")
	configNameCmd.Dir = dbPath
	if err := configNameCmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git name: %w", err)
	}

	// Add and commit using git command
	addCmd := exec.Command("git", "add", "main.db")
	addCmd.Dir = dbPath
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add database file: %w", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "Initial database commit")
	commitCmd.Dir = dbPath
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	return nil
}

func (m *Manager) CreateBranch(dbName, branchName string) error {
	dbPath := filepath.Join(m.dataDir, dbName)

	repo, err := git.PlainOpen(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)
	branchRef := plumbing.NewHashReference(branchRefName, head.Hash())

	if err := repo.Storer.SetReference(branchRef); err != nil {
		return fmt.Errorf("failed to create branch reference: %w", err)
	}

	branchPath := filepath.Join(dbPath, fmt.Sprintf("worktrees/%s", branchName))
	if err := os.MkdirAll(branchPath, 0755); err != nil {
		return fmt.Errorf("failed to create worktree directory: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get main worktree: %w", err)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRefName,
		Create: false,
	}); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}

func (m *Manager) DeleteBranch(dbName, branchName string) error {
	if branchName == "main" {
		return fmt.Errorf("cannot delete main branch")
	}

	dbPath := filepath.Join(m.dataDir, dbName)

	repo, err := git.PlainOpen(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)
	if err := repo.Storer.RemoveReference(branchRefName); err != nil {
		return fmt.Errorf("failed to remove branch reference: %w", err)
	}

	branchPath := filepath.Join(dbPath, fmt.Sprintf("worktrees/%s", branchName))
	if err := os.RemoveAll(branchPath); err != nil {
		return fmt.Errorf("failed to remove worktree directory: %w", err)
	}

	return nil
}

func (m *Manager) ListBranches(dbName string) ([]string, error) {
	dbPath := filepath.Join(m.dataDir, dbName)

	repo, err := git.PlainOpen(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	var branches []string
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branchName := strings.TrimPrefix(ref.Name().String(), "refs/heads/")
			branches = append(branches, branchName)
		}
		return nil
	})

	return branches, err
}

func (m *Manager) GetBranchPath(dbName, branchName string) string {
	if branchName == "main" {
		return filepath.Join(m.dataDir, dbName, "main.db")
	}
	return filepath.Join(m.dataDir, dbName, fmt.Sprintf("worktrees/%s/main.db", branchName))
}

func (m *Manager) BranchExists(dbName, branchName string) bool {
	branches, err := m.ListBranches(dbName)
	if err != nil {
		return false
	}

	for _, branch := range branches {
		if branch == branchName {
			return true
		}
	}
	return false
}
