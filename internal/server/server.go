package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bxrne/branchlore/internal/database"
	"github.com/bxrne/branchlore/internal/git"
	"github.com/bxrne/branchlore/internal/metrics"
	"github.com/bxrne/branchlore/internal/storage"
	"github.com/bxrne/branchlore/internal/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPServer struct {
	config  *types.Config
	git     *git.Repository
	storage *storage.FileSystem
	server  *http.Server
}

func NewHTTPServer(config *types.Config, gitRepo *git.Repository, fs *storage.FileSystem) *HTTPServer {
	return &HTTPServer{
		config:  config,
		git:     gitRepo,
		storage: fs,
	}
}

func (h *HTTPServer) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	h.RegisterHandlers(mux)

	h.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	slog.Info("Starting HTTP server", "addr", addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}
	}()

	if err := h.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (h *HTTPServer) Stop(ctx context.Context) error {
	if h.server != nil {
		return h.server.Shutdown(ctx)
	}
	return nil
}

func (h *HTTPServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.withMetrics("GET", "/health", h.handleHealth))
	mux.HandleFunc("/api/branches", h.withMetrics("", "/api/branches", h.handleBranches))
	mux.HandleFunc("/api/branches/", h.withMetrics("", "/api/branches/", h.handleBranch))
	mux.HandleFunc("/api/query", h.withMetrics("POST", "/api/query", h.handleQuery))
	mux.HandleFunc("/api/merge", h.withMetrics("POST", "/api/merge", h.handleMerge))
	mux.HandleFunc("/api/status", h.withMetrics("GET", "/api/status", h.handleStatus))
	mux.Handle("/metrics", promhttp.Handler())
}

func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data: map[string]any{
			"status":    "healthy",
			"timestamp": time.Now(),
		},
	})
}

func (h *HTTPServer) handleBranches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listBranches(w, r)
	case http.MethodPost:
		h.createBranch(w, r)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}
}

func (h *HTTPServer) listBranches(w http.ResponseWriter, r *http.Request) {
	branches, err := h.git.ListBranches()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "LIST_BRANCHES_ERROR", err.Error())
		return
	}

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data:    branches,
	})
}

func (h *HTTPServer) createBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_NAME", "Branch name is required")
		return
	}

	branch, err := h.git.CreateBranch(req.Name)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "CREATE_BRANCH_ERROR", err.Error())
		return
	}

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data:    branch,
	})
}

func (h *HTTPServer) handleBranch(w http.ResponseWriter, r *http.Request) {
	branchName := strings.TrimPrefix(r.URL.Path, "/api/branches/")
	if branchName == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_BRANCH", "Branch name is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getBranch(w, r, branchName)
	case http.MethodDelete:
		h.deleteBranch(w, r, branchName)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}
}

func (h *HTTPServer) getBranch(w http.ResponseWriter, r *http.Request, branchName string) {
	branch, err := h.git.GetBranch(branchName)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "BRANCH_NOT_FOUND", err.Error())
		return
	}

	worktreePath := h.storage.GetWorktreePath(h.storage.GetRepoPath(), branchName)
	dbPath := h.storage.GetDBPath(worktreePath)

	status := &types.BranchStatus{
		Branch:   *branch,
		DBExists: h.storage.PathExists(dbPath),
		DBPath:   dbPath,
	}

	if status.DBExists {
		size, err := h.storage.GetSize(dbPath)
		if err == nil {
			status.Size = size
		}
	}

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data:    status,
	})
}

func (h *HTTPServer) deleteBranch(w http.ResponseWriter, r *http.Request, branchName string) {
	h.writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Branch deletion not implemented")
}

func (h *HTTPServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var req struct {
		Branch string `json:"branch"`
		SQL    string `json:"sql"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.Branch == "" || req.SQL == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_PARAMS", "Branch and SQL are required")
		return
	}

	worktreePath, err := h.git.CreateWorktree(req.Branch)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "WORKTREE_ERROR", err.Error())
		return
	}

	dbPath := h.storage.GetDBPath(worktreePath)
	db := database.NewSQLiteDB()

	if err := db.Open(dbPath); err != nil {
		h.writeError(w, http.StatusInternalServerError, "DB_OPEN_ERROR", err.Error())
		return
	}
	defer db.Close()

	if !h.storage.PathExists(dbPath) {
		if err := db.InitSchema(); err != nil {
			h.writeError(w, http.StatusInternalServerError, "DB_INIT_ERROR", err.Error())
			return
		}
	}

	queryStart := time.Now()
	result, err := db.Query(r.Context(), req.SQL)
	metrics.DBQueryDuration.Observe(time.Since(queryStart).Seconds())

	if err != nil {
		metrics.DBQueryErrors.Inc()
		h.writeError(w, http.StatusBadRequest, "QUERY_ERROR", err.Error())
		return
	}

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data:    result,
	})
}

func (h *HTTPServer) handleMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var req types.MergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.Source == "" || req.Target == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_PARAMS", "Source and target branches are required")
		return
	}

	result, err := h.git.MergeBranches(req.Source, req.Target)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "MERGE_ERROR", err.Error())
		return
	}

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data:    result,
	})
}

func (h *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	branches, err := h.git.ListBranches()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "STATUS_ERROR", err.Error())
		return
	}

	dbPaths, err := h.storage.GetBranchDBs()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "STATUS_ERROR", err.Error())
		return
	}

	status := map[string]any{
		"branches":       branches,
		"branch_count":   len(branches),
		"database_count": len(dbPaths),
		"databases":      dbPaths,
		"config":         h.config,
	}

	h.writeJSON(w, types.APIResponse{
		Success: true,
		Data:    status,
	})
}

func (h *HTTPServer) writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *HTTPServer) writeError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := types.APIResponse{
		Success: false,
		Error: &types.ServiceError{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode error response", "error", err)
	}
}

func (h *HTTPServer) withMetrics(method, endpoint string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		actualMethod := method
		if actualMethod == "" {
			actualMethod = r.Method
		}

		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		handler(ww, r)

		duration := time.Since(start).Seconds()
		metrics.HTTPRequestDuration.WithLabelValues(actualMethod, endpoint).Observe(duration)
		metrics.HTTPRequestsTotal.WithLabelValues(actualMethod, endpoint, http.StatusText(ww.statusCode)).Inc()
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
