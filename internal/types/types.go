package types

import (
	"context"
	"time"
)

type Branch struct {
	Name      string    `json:"name"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
	IsMain    bool      `json:"is_main"`
}

type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`
}

type MergeRequest struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type MergeResult struct {
	Success   bool     `json:"success"`
	Conflicts []string `json:"conflicts,omitempty"`
	Message   string   `json:"message"`
}

type BranchStatus struct {
	Branch   Branch `json:"branch"`
	DBExists bool   `json:"db_exists"`
	DBPath   string `json:"db_path"`
	Size     int64  `json:"size"`
}

type GitRepository interface {
	Init(path string) error
	CreateBranch(name string) (*Branch, error)
	GetBranch(name string) (*Branch, error)
	ListBranches() ([]*Branch, error)
	CreateWorktree(branch string) (string, error)
	MergeBranches(source, target string) (*MergeResult, error)
	GetCurrentHash() (string, error)
}

type Database interface {
	Open(path string) error
	Close() error
	Query(ctx context.Context, sql string) (*QueryResult, error)
	Exec(ctx context.Context, sql string) error
	InitSchema() error
	GetTables() ([]string, error)
	GetSchema(table string) (string, error)
}

type Storage interface {
	EnsureDir(path string) error
	PathExists(path string) bool
	GetDBPath(worktreePath string) string
	GetWorktreePath(repoPath, branch string) string
	Cleanup(path string) error
	GetSize(path string) (int64, error)
}

type Server interface {
	Start(ctx context.Context, addr string) error
	Stop(ctx context.Context) error
	RegisterHandlers()
}

type Simulator interface {
	Run(ctx context.Context, scenario string) error
	CreateMockRepo(path string) error
	SimulateOperations(ops []string) error
	GetMetrics() map[string]interface{}
}

type Config struct {
	RepoPath     string `json:"repo_path"`
	WorktreeBase string `json:"worktree_base"`
	DBFileName   string `json:"db_filename"`
	ServerAddr   string `json:"server_addr"`
	LogLevel     string `json:"log_level"`
	Simulate     bool   `json:"simulate"`
}

type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ServiceError) Error() string {
	return e.Message
}

type APIResponse struct {
	Success bool          `json:"success"`
	Data    interface{}   `json:"data,omitempty"`
	Error   *ServiceError `json:"error,omitempty"`
}

const (
	DefaultRepoPath     = "branchlore-repo"
	DefaultWorktreeBase = "worktrees"
	DefaultDBFileName   = "db.sqlite"
	DefaultServerAddr   = ":8080"
	DefaultLogLevel     = "info"
)
