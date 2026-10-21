package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngine_CSVToParquetAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "clients.csv")

	csvData := "name,age,subscribed\nAlice,30,yes\nBob,45,no\nCharlie,28,yes\n"
	if err := os.WriteFile(csvPath, []byte(csvData), 0644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}

	eng, err := NewEngine(csvPath)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	defer eng.Close()

	// 1. Check that .parquet file was generated
	if !strings.HasSuffix(eng.ParquetPath, ".parquet") {
		t.Errorf("expected parquet path, got %s", eng.ParquetPath)
	}

	// 2. Check schema discovery
	schema, err := eng.DiscoverSchema()
	if err != nil {
		t.Fatalf("DiscoverSchema failed: %v", err)
	}
	if !strings.Contains(schema, "name") || !strings.Contains(schema, "age") {
		t.Errorf("unexpected schema output: %s", schema)
	}

	// 3. Run read-only query against the normalized Parquet view
	res, err := eng.ExecuteReadOnly("SELECT COUNT(*) AS total FROM data WHERE subscribed = 'yes'")
	if err != nil {
		t.Fatalf("ExecuteReadOnly failed: %v", err)
	}

	if len(res.Rows) != 1 || res.Rows[0][0] != "2" {
		t.Errorf("expected count '2', got %+v", res.Rows)
	}
}
