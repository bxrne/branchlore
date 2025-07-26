package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bxrne/branchlore/internal/types"
)

type FileSystem struct {
	config *types.Config
}

func NewFileSystem(config *types.Config) *FileSystem {
	return &FileSystem{config: config}
}

func (fs *FileSystem) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (fs *FileSystem) PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (fs *FileSystem) GetDBPath(worktreePath string) string {
	return filepath.Join(worktreePath, fs.config.DBFileName)
}

func (fs *FileSystem) GetWorktreePath(repoPath, branch string) string {
	return filepath.Join(repoPath, fs.config.WorktreeBase, branch)
}

func (fs *FileSystem) GetRepoPath() string {
	if filepath.IsAbs(fs.config.RepoPath) {
		return fs.config.RepoPath
	}

	wd, err := os.Getwd()
	if err != nil {
		return fs.config.RepoPath
	}

	return filepath.Join(wd, fs.config.RepoPath)
}

func (fs *FileSystem) Cleanup(path string) error {
	if !fs.PathExists(path) {
		return nil
	}
	return os.RemoveAll(path)
}

func (fs *FileSystem) GetSize(path string) (int64, error) {
	if !fs.PathExists(path) {
		return 0, fmt.Errorf("path does not exist: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if info.IsDir() {
		return fs.getDirSize(path)
	}

	return info.Size(), nil
}

func (fs *FileSystem) getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func (fs *FileSystem) ListFiles(path string) ([]string, error) {
	if !fs.PathExists(path) {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

func (fs *FileSystem) ListDirs(path string) ([]string, error) {
	if !fs.PathExists(path) {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

func (fs *FileSystem) CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := fs.EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	buf := make([]byte, 4096)
	for {
		n, err := sourceFile.Read(buf)
		if err != nil && err.Error() != "EOF" {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := destFile.Write(buf[:n]); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) GetBranchDBs() (map[string]string, error) {
	repoPath := fs.GetRepoPath()
	worktreesPath := filepath.Join(repoPath, fs.config.WorktreeBase)

	if !fs.PathExists(worktreesPath) {
		return make(map[string]string), nil
	}

	branches, err := fs.ListDirs(worktreesPath)
	if err != nil {
		return nil, err
	}

	dbPaths := make(map[string]string)
	for _, branch := range branches {
		worktreePath := fs.GetWorktreePath(repoPath, branch)
		dbPath := fs.GetDBPath(worktreePath)
		if fs.PathExists(dbPath) {
			dbPaths[branch] = dbPath
		}
	}

	return dbPaths, nil
}

func (fs *FileSystem) CreateTempDir() (string, error) {
	return os.MkdirTemp("", "branchlore-*")
}

func (fs *FileSystem) WriteFile(path string, content []byte) error {
	if err := fs.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func (fs *FileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
