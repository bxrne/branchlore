package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bxrne/branchlore/internal/git"
	_ "github.com/mattn/go-sqlite3"
)

type Manager struct {
	gitMgr *git.Manager
	conns  map[string]*sql.DB
}

func NewManager(dataDir string, gitMgr *git.Manager) (*Manager, error) {
	return &Manager{
		gitMgr: gitMgr,
		conns:  make(map[string]*sql.DB),
	}, nil
}

type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Error   string          `json:"error,omitempty"`
}

func (m *Manager) ExecuteQuery(ctx context.Context, dbName, branch, query string) ([]byte, error) {
	if !m.gitMgr.BranchExists(dbName, branch) {
		return nil, fmt.Errorf("branch %s does not exist", branch)
	}

	dbPath := m.gitMgr.GetBranchPath(dbName, branch)

	connKey := fmt.Sprintf("%s@%s", dbName, branch)
	db, exists := m.conns[connKey]
	if !exists {
		var err error
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		m.conns[connKey] = db
	}

	query = strings.TrimSpace(query)
	if strings.ToUpper(strings.Split(query, " ")[0]) == "SELECT" {
		return m.executeSelect(db, query)
	} else {
		return m.executeModify(db, query)
	}
}

func (m *Manager) executeSelect(db *sql.DB, query string) ([]byte, error) {
	rows, err := db.Query(query)
	if err != nil {
		result := QueryResult{Error: err.Error()}
		return json.Marshal(result)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		result := QueryResult{Error: err.Error()}
		return json.Marshal(result)
	}

	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			result := QueryResult{Error: err.Error()}
			return json.Marshal(result)
		}

		row := make([]interface{}, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = nil
			} else {
				switch v := val.(type) {
				case []byte:
					row[i] = string(v)
				default:
					row[i] = v
				}
			}
		}
		resultRows = append(resultRows, row)
	}

	result := QueryResult{
		Columns: columns,
		Rows:    resultRows,
	}

	return json.Marshal(result)
}

func (m *Manager) executeModify(db *sql.DB, query string) ([]byte, error) {
	result, err := db.Exec(query)
	if err != nil {
		queryResult := QueryResult{Error: err.Error()}
		return json.Marshal(queryResult)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	response := map[string]interface{}{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertId,
	}

	return json.Marshal(response)
}

func (m *Manager) Close() {
	for _, db := range m.conns {
		db.Close()
	}
}
