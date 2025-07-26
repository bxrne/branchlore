package simulator

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/bxrne/branchlore/internal/types"
)

type MockSimulator struct {
	config   *types.Config
	metrics  map[string]any
	branches map[string]*types.Branch
	dbs      map[string]map[string]any
	mu       sync.RWMutex
}

func NewMockSimulator(config *types.Config) *MockSimulator {
	return &MockSimulator{
		config:   config,
		metrics:  make(map[string]any),
		branches: make(map[string]*types.Branch),
		dbs:      make(map[string]map[string]any),
	}
}

func (s *MockSimulator) Run(ctx context.Context, scenario string) error {
	slog.Info("Starting simulation", "scenario", scenario)

	start := time.Now()
	defer func() {
		s.mu.Lock()
		s.metrics["duration"] = time.Since(start)
		s.metrics["completed_at"] = time.Now()
		s.mu.Unlock()
	}()

	switch scenario {
	case "basic":
		return s.runBasicScenario(ctx)
	case "stress":
		return s.runStressScenario(ctx)
	case "concurrent":
		return s.runConcurrentScenario(ctx)
	case "chaos":
		return s.runChaosScenario(ctx)
	default:
		return fmt.Errorf("unknown scenario: %s", scenario)
	}
}

func (s *MockSimulator) runBasicScenario(ctx context.Context) error {
	operations := []string{
		"create_repo",
		"create_branch:feature1",
		"create_branch:feature2",
		"query:feature1:SELECT 1",
		"query:feature2:INSERT INTO demo (msg) VALUES ('test')",
		"query:feature2:SELECT * FROM demo",
		"merge:feature1:main",
		"list_branches",
	}

	return s.SimulateOperations(operations)
}

func (s *MockSimulator) runStressScenario(ctx context.Context) error {
	var operations []string

	for i := 0; i < 50; i++ {
		operations = append(operations, fmt.Sprintf("create_branch:stress_%d", i))
		operations = append(operations, fmt.Sprintf("query:stress_%d:INSERT INTO demo (msg) VALUES ('stress_%d')", i, i))
	}

	for i := 0; i < 100; i++ {
		branchIdx := rand.Intn(50)
		operations = append(operations, fmt.Sprintf("query:stress_%d:SELECT * FROM demo", branchIdx))
	}

	return s.SimulateOperations(operations)
}

func (s *MockSimulator) runConcurrentScenario(ctx context.Context) error {
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ops := []string{
				fmt.Sprintf("create_branch:concurrent_%d", id),
				fmt.Sprintf("query:concurrent_%d:INSERT INTO demo (msg) VALUES ('concurrent_%d')", id, id),
				fmt.Sprintf("query:concurrent_%d:SELECT COUNT(*) FROM demo", id),
			}

			if err := s.SimulateOperations(ops); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *MockSimulator) runChaosScenario(ctx context.Context) error {
	operations := []string{
		"create_branch:chaos",
		"query:chaos:CREATE TABLE test (id INTEGER)",
		"simulate_error:network",
		"query:chaos:INSERT INTO test VALUES (1)",
		"simulate_error:disk_full",
		"query:chaos:SELECT * FROM test",
		"simulate_recovery",
		"query:chaos:INSERT INTO test VALUES (2)",
	}

	return s.SimulateOperations(operations)
}

func (s *MockSimulator) CreateMockRepo(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.branches["main"] = &types.Branch{
		Name:      "main",
		Hash:      "abc123",
		CreatedAt: time.Now(),
		IsMain:    true,
	}

	s.dbs["main"] = map[string]any{
		"tables": []string{"demo", "_branchlore_metadata"},
		"size":   1024,
	}

	s.updateMetrics("repo_created", 1)
	slog.Info("Mock repository created", "path", path)
	return nil
}

func (s *MockSimulator) SimulateOperations(ops []string) error {
	s.mu.Lock()
	s.metrics["operations_total"] = len(ops)
	s.metrics["operations_completed"] = 0
	s.mu.Unlock()

	for i, op := range ops {
		if err := s.simulateOperation(op); err != nil {
			s.updateMetrics("operations_failed", 1)
			return fmt.Errorf("operation %d failed: %w", i, err)
		}

		s.mu.Lock()
		s.metrics["operations_completed"] = i + 1
		s.mu.Unlock()

		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	}

	return nil
}

func (s *MockSimulator) simulateOperation(op string) error {
	parts := parseOperation(op)
	opType := parts[0]

	switch opType {
	case "create_repo":
		return s.CreateMockRepo("mock-repo")

	case "create_branch":
		if len(parts) < 2 {
			return fmt.Errorf("create_branch requires branch name")
		}
		return s.simulateCreateBranch(parts[1])

	case "query":
		if len(parts) < 3 {
			return fmt.Errorf("query requires branch and SQL")
		}
		return s.simulateQuery(parts[1], parts[2])

	case "merge":
		if len(parts) < 3 {
			return fmt.Errorf("merge requires source and target")
		}
		return s.simulateMerge(parts[1], parts[2])

	case "list_branches":
		return s.simulateListBranches()

	case "simulate_error":
		if len(parts) < 2 {
			return fmt.Errorf("simulate_error requires error type")
		}
		return s.simulateError(parts[1])

	case "simulate_recovery":
		return s.simulateRecovery()

	default:
		return fmt.Errorf("unknown operation: %s", opType)
	}
}

func (s *MockSimulator) simulateCreateBranch(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.branches[name]; exists {
		return fmt.Errorf("branch %s already exists", name)
	}

	s.branches[name] = &types.Branch{
		Name:      name,
		Hash:      fmt.Sprintf("hash_%d", rand.Intn(999999)),
		CreatedAt: time.Now(),
		IsMain:    false,
	}

	s.dbs[name] = map[string]any{
		"tables": []string{"demo", "_branchlore_metadata"},
		"size":   512,
	}

	s.updateMetrics("branches_created", 1)
	slog.Info("Mock branch created", "name", name)
	return nil
}

func (s *MockSimulator) simulateQuery(branch, sql string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.branches[branch]; !exists {
		return fmt.Errorf("branch %s does not exist", branch)
	}

	s.updateMetrics("queries_executed", 1)

	db := s.dbs[branch]
	if db != nil {
		size := db["size"].(int) + rand.Intn(100)
		db["size"] = size
	}

	slog.Info("Mock query executed", "branch", branch, "sql", sql)
	return nil
}

func (s *MockSimulator) simulateMerge(source, target string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.branches[source]; !exists {
		return fmt.Errorf("source branch %s does not exist", source)
	}
	if _, exists := s.branches[target]; !exists {
		return fmt.Errorf("target branch %s does not exist", target)
	}

	s.updateMetrics("merges_completed", 1)
	slog.Info("Mock merge completed", "source", source, "target", target)
	return nil
}

func (s *MockSimulator) simulateListBranches() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("Mock list branches", "count", len(s.branches))
	s.updateMetrics("list_operations", 1)
	return nil
}

func (s *MockSimulator) simulateError(errorType string) error {
	s.updateMetrics("errors_simulated", 1)
	slog.Warn("Simulating error", "type", errorType)

	switch errorType {
	case "network":
		time.Sleep(100 * time.Millisecond)
	case "disk_full":
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

func (s *MockSimulator) simulateRecovery() error {
	s.updateMetrics("recoveries_simulated", 1)
	slog.Info("Simulating recovery")
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (s *MockSimulator) GetMetrics() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]any)
	for k, v := range s.metrics {
		result[k] = v
	}

	result["branch_count"] = len(s.branches)
	result["db_count"] = len(s.dbs)

	return result
}

func (s *MockSimulator) updateMetrics(key string, increment int) {
	if current, exists := s.metrics[key]; exists {
		s.metrics[key] = current.(int) + increment
	} else {
		s.metrics[key] = increment
	}
}

func parseOperation(op string) []string {
	var parts []string
	current := ""

	for _, char := range op {
		if char == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
