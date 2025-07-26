package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bxrne/branchlore/internal/metrics"
	"github.com/bxrne/branchlore/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	db   *sql.DB
	path string
}

func NewSQLiteDB() *SQLiteDB {
	return &SQLiteDB{}
}

func (s *SQLiteDB) Open(path string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	s.db = db
	s.path = path
	return nil
}

func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteDB) Query(ctx context.Context, sqlQuery string) (*types.QueryResult, error) {
	start := time.Now()
	defer func() {
		metrics.DBQueryDuration.Observe(time.Since(start).Seconds())
	}()

	if s.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		metrics.DBQueryErrors.Inc()
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		metrics.DBQueryErrors.Inc()
		return nil, err
	}

	var results [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			metrics.DBQueryErrors.Inc()
			return nil, err
		}

		row := make([]any, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = nil
			} else if b, ok := val.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		metrics.DBQueryErrors.Inc()
		return nil, err
	}

	return &types.QueryResult{
		Columns: columns,
		Rows:    results,
		Count:   len(results),
	}, nil
}

func (s *SQLiteDB) Exec(ctx context.Context, sqlQuery string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryDuration.Observe(time.Since(start).Seconds())
	}()

	if s.db == nil {
		return fmt.Errorf("database not open")
	}

	_, err := s.db.ExecContext(ctx, sqlQuery)
	if err != nil {
		metrics.DBQueryErrors.Inc()
	}
	return err
}

func (s *SQLiteDB) InitSchema() error {
	if s.db == nil {
		return fmt.Errorf("database not open")
	}

	schema := `
	CREATE TABLE IF NOT EXISTS demo (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		msg TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS _branchlore_metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	INSERT OR IGNORE INTO _branchlore_metadata (key, value) 
	VALUES ('schema_version', '1.0');
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteDB) GetTables() ([]string, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	query := `
	SELECT name FROM sqlite_master 
	WHERE type='table' AND name NOT LIKE 'sqlite_%' 
	ORDER BY name;
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func (s *SQLiteDB) GetSchema(table string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database not open")
	}

	query := `
	SELECT sql FROM sqlite_master 
	WHERE type='table' AND name=?;
	`

	var schema string
	err := s.db.QueryRow(query, table).Scan(&schema)
	if err != nil {
		return "", err
	}

	return schema, nil
}

func (s *SQLiteDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	return s.db.BeginTx(ctx, nil)
}

func (s *SQLiteDB) ExecTx(ctx context.Context, tx *sql.Tx, queries []string) error {
	start := time.Now()
	defer func() {
		metrics.DBQueryDuration.Observe(time.Since(start).Seconds())
	}()

	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, query); err != nil {
			metrics.DBQueryErrors.Inc()
			return err
		}
	}
	return nil
}

func (s *SQLiteDB) GetStats() (map[string]any, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not open")
	}

	stats := make(map[string]any)

	var pageCount, pageSize int64
	err := s.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return nil, err
	}

	stats["page_count"] = pageCount
	stats["page_size"] = pageSize
	stats["file_size"] = pageCount * pageSize

	tables, err := s.GetTables()
	if err != nil {
		return nil, err
	}
	stats["table_count"] = len(tables)
	stats["tables"] = tables

	return stats, nil
}
