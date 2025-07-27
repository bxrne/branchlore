package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/bxrne/branchlore/internal/database"
	"github.com/bxrne/branchlore/internal/git"
)

type Config struct {
	Port     string
	DataDir  string
	LogLevel string
}

type Server struct {
	config   *Config
	listener net.Listener
	dbMgr    *database.Manager
	gitMgr   *git.Manager
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func New(config *Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	gitMgr, err := git.NewManager(config.DataDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create git manager: %w", err)
	}

	dbMgr, err := database.NewManager(config.DataDir, gitMgr)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}

	return &Server{
		config: config,
		dbMgr:  dbMgr,
		gitMgr: gitMgr,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", ":"+s.config.Port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.config.Port, err)
	}
	s.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/query", s.handleQuery)
	mux.HandleFunc("/branch", s.handleBranch)
	mux.HandleFunc("/health", s.handleHealth)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server.Serve(listener)
}

func (s *Server) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dbName := r.URL.Query().Get("db")
	branch := r.URL.Query().Get("branch")
	if branch == "" {
		branch = "main"
	}

	query := r.FormValue("query")
	if query == "" {
		http.Error(w, "Query parameter required", http.StatusBadRequest)
		return
	}

	result, err := s.dbMgr.ExecuteQuery(s.ctx, dbName, branch, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

func (s *Server) handleBranch(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("db")
	action := r.URL.Query().Get("action")
	branch := r.URL.Query().Get("branch")

	switch action {
	case "create":
		if err := s.gitMgr.CreateBranch(dbName, branch); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create branch: %v", err), http.StatusInternalServerError)
			return
		}
	case "delete":
		if err := s.gitMgr.DeleteBranch(dbName, branch); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete branch: %v", err), http.StatusInternalServerError)
			return
		}
	case "list":
		branches, err := s.gitMgr.ListBranches(dbName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list branches: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"branches": %q}`, branches)
		return
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status": "healthy"}`)
}
