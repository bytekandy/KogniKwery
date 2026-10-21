package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

// QueryResult holds the columns, rows, and execution time of a read-only SQL query.
type QueryResult struct {
	SQL      string
	Columns  []string
	Rows     [][]string
	Duration time.Duration
}

// Engine manages an in-memory DuckDB instance backed by normalized Parquet files.
type Engine struct {
	conn        *sql.DB
	SourcePath  string
	ParquetPath string
	TableName   string
}

// NewEngine takes any tabular file (CSV, TSV, JSON, Parquet), converts it to Parquet
// inside .kq_cache/ if needed, and mounts it as a SQL view named "data".
func NewEngine(sourcePath string) (*Engine, error) {
	if _, err := os.Stat(sourcePath); err != nil {
		return nil, fmt.Errorf("data file not found: %w", err)
	}

	conn, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}

	e := &Engine{
		conn:       conn,
		SourcePath: sourcePath,
		TableName:  "data",
	}

	if err := e.normalizeToParquet(); err != nil {
		conn.Close()
		return nil, err
	}

	return e, nil
}

// normalizeToParquet converts CSV/TSV/JSON files into .kq_cache/<name>.parquet
// and mounts the resulting Parquet file as a view named `data`.
func (e *Engine) normalizeToParquet() error {
	ext := strings.ToLower(filepath.Ext(e.SourcePath))
	escapedSrc := strings.ReplaceAll(e.SourcePath, "'", "''")

	if ext == ".parquet" {
		e.ParquetPath = e.SourcePath
	} else {
		cacheDir := ".kq_cache"
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create cache dir: %w", err)
		}

		base := strings.TrimSuffix(filepath.Base(e.SourcePath), filepath.Ext(e.SourcePath))
		e.ParquetPath = filepath.Join(cacheDir, base+".parquet")
		escapedParquet := strings.ReplaceAll(e.ParquetPath, "'", "''")

		// Convert any delimited/tabular file (CSV, TSV, JSON) into Parquet via DuckDB
		copySQL := fmt.Sprintf("COPY (SELECT * FROM '%s') TO '%s' (FORMAT PARQUET)", escapedSrc, escapedParquet)
		if _, err := e.conn.Exec(copySQL); err != nil {
			return fmt.Errorf("failed to convert %s to parquet: %w", e.SourcePath, err)
		}
	}

	escapedParquet := strings.ReplaceAll(e.ParquetPath, "'", "''")
	viewSQL := fmt.Sprintf("CREATE VIEW %s AS SELECT * FROM '%s'", e.TableName, escapedParquet)
	if _, err := e.conn.Exec(viewSQL); err != nil {
		return fmt.Errorf("failed to create SQL view: %w", err)
	}

	return nil
}

// DiscoverSchema inspects the `data` view and returns a formatted schema string for the LLM.
func (e *Engine) DiscoverSchema() (string, error) {
	rows, err := e.conn.Query(fmt.Sprintf("DESCRIBE %s", e.TableName))
	if err != nil {
		return "", fmt.Errorf("failed to describe schema: %w", err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var colName, colType string
		var null, key, def, extra sql.NullString
		if err := rows.Scan(&colName, &colType, &null, &key, &def, &extra); err != nil {
			return "", err
		}
		cols = append(cols, fmt.Sprintf("%s (%s)", colName, colType))
	}

	return fmt.Sprintf("Table %s: %s", e.TableName, strings.Join(cols, ", ")), nil
}

// ExecuteReadOnly validates the SQL via our guardrail and executes it with a 60s timeout.
func (e *Engine) ExecuteReadOnly(rawSQL string) (*QueryResult, error) {
	safeSQL, err := ValidateAndNormalizeSQL(rawSQL, 1000)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	rows, err := e.conn.QueryContext(ctx, safeSQL)
	if err != nil {
		return nil, fmt.Errorf("SQL execution error: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var resultRows [][]string
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		rowStr := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				rowStr[i] = "NULL"
			} else if b, ok := v.([]byte); ok {
				rowStr[i] = string(b)
			} else {
				rowStr[i] = fmt.Sprintf("%v", v)
			}
		}
		resultRows = append(resultRows, rowStr)
	}

	return &QueryResult{
		SQL:      safeSQL,
		Columns:  cols,
		Rows:     resultRows,
		Duration: time.Since(start),
	}, nil
}

// Close releases the DuckDB connection.
func (e *Engine) Close() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}
