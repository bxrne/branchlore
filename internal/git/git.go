package git

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bxrne/branchlore/internal/types"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

type Repository struct {
	path string
	repo *git.Repository
}

func NewRepository(path string) *Repository {
	return &Repository{path: path}
}

func (r *Repository) Init(path string) error {
	r.path = path
	repo, err := git.PlainOpen(path)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			slog.Info("No repository found, initializing new one", "path", path)
			repo, err = git.PlainInit(path, false)
			if err != nil {
				return err
			}
			if err := r.ensureInitialCommit(repo); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	r.repo = repo
	return nil
}

func (r *Repository) ensureInitialCommit(repo *git.Repository) error {
	_, err := repo.Head()
	if err == plumbing.ErrReferenceNotFound {
		slog.Info("No HEAD found; creating initial commit")

		worktree, err := repo.Worktree()
		if err != nil {
			return err
		}

		initialFile := filepath.Join(r.path, "init.txt")
		if err := os.WriteFile(initialFile, []byte("initial"), 0644); err != nil {
			return err
		}

		if _, err := worktree.Add("init.txt"); err != nil {
			return err
		}

		_, err = worktree.Commit("Initial commit", &git.CommitOptions{
			AllowEmptyCommits: true,
			Author: &object.Signature{
				Name:  "Branchlore",
				Email: "branchlore@example.com",
				When:  time.Now(),
			},
		})
		return err
	}
	return nil
}

func (r *Repository) CreateBranch(name string) (*types.Branch, error) {
	if r.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	head, err := r.repo.Head()
	if err != nil {
		return nil, err
	}

	refName := plumbing.NewBranchReferenceName(name)
	ref := plumbing.NewHashReference(refName, head.Hash())

	if err := r.repo.Storer.SetReference(ref); err != nil {
		return nil, err
	}

	return &types.Branch{
		Name:      name,
		Hash:      head.Hash().String(),
		CreatedAt: time.Now(),
		IsMain:    name == "main" || name == "master",
	}, nil
}

func (r *Repository) GetBranch(name string) (*types.Branch, error) {
	if r.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	refName := plumbing.NewBranchReferenceName(name)
	ref, err := r.repo.Reference(refName, true)
	if err != nil {
		return nil, err
	}

	commit, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	return &types.Branch{
		Name:      name,
		Hash:      ref.Hash().String(),
		CreatedAt: commit.Author.When,
		IsMain:    name == "main" || name == "master",
	}, nil
}

func (r *Repository) ListBranches() ([]*types.Branch, error) {
	if r.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}

	var branches []*types.Branch
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branchName := ref.Name().Short()
			commit, err := r.repo.CommitObject(ref.Hash())
			if err != nil {
				return err
			}

			branches = append(branches, &types.Branch{
				Name:      branchName,
				Hash:      ref.Hash().String(),
				CreatedAt: commit.Author.When,
				IsMain:    branchName == "main" || branchName == "master",
			})
		}
		return nil
	})

	return branches, err
}

func (r *Repository) CreateWorktree(branch string) (string, error) {
	if r.repo == nil {
		return "", errors.New("repository not initialized")
	}

	absRepoPath, err := filepath.Abs(r.path)
	if err != nil {
		return "", err
	}

	worktreePath := filepath.Join(absRepoPath, "worktrees", branch)

	if _, err := os.Stat(worktreePath); err == nil {
		slog.Info("Worktree already exists", "path", worktreePath)
		return worktreePath, nil
	}

	cmd := exec.Command("git", "-C", absRepoPath, "worktree", "add", worktreePath, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed to create worktree", "output", string(output), "error", err)
		return "", err
	}

	slog.Info("Created worktree", "path", worktreePath)
	return worktreePath, nil
}

func (r *Repository) MergeBranches(source, target string) (*types.MergeResult, error) {
	if r.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	absRepoPath, err := filepath.Abs(r.path)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "-C", absRepoPath, "checkout", target)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &types.MergeResult{
			Success: false,
			Message: string(output),
		}, err
	}

	cmd = exec.Command("git", "-C", absRepoPath, "merge", source)
	output, err := cmd.CombinedOutput()

	result := &types.MergeResult{
		Success: err == nil,
		Message: string(output),
	}

	if err != nil && strings.Contains(string(output), "CONFLICT") {
		conflicts := r.parseConflicts(string(output))
		result.Conflicts = conflicts
	}

	return result, nil
}

func (r *Repository) parseConflicts(output string) []string {
	var conflicts []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "CONFLICT") {
			conflicts = append(conflicts, strings.TrimSpace(line))
		}
	}
	return conflicts
}

func (r *Repository) GetCurrentHash() (string, error) {
	if r.repo == nil {
		return "", errors.New("repository not initialized")
	}

	head, err := r.repo.Head()
	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}
