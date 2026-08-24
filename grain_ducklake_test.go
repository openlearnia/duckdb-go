//go:build duckdb_grain

package duckdb

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
)

// TestGrainDuckLakeExtensionLoad is an opt-in smoke test for the paired
// DuckLake extension. Set DUCKLAKE_EXTENSION_PATH to the exact extension built
// against the same DuckDB runtime before running the test.
func TestGrainDuckLakeExtensionLoad(t *testing.T) {
	extensionPath := os.Getenv("DUCKLAKE_EXTENSION_PATH")
	if extensionPath == "" {
		t.Skip("DUCKLAKE_EXTENSION_PATH is not set")
	}

	connector, err := NewConnector(":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	database := sql.OpenDB(connector)
	defer database.Close()

	quotedPath := "'" + strings.ReplaceAll(extensionPath, "'", "''") + "'"
	if _, err := database.ExecContext(context.Background(), "LOAD "+quotedPath); err != nil {
		t.Fatalf("load DuckLake extension: %v", err)
	}

	var loaded bool
	if err := database.QueryRowContext(context.Background(), `
		SELECT installed AND loaded
		FROM duckdb_extensions()
		WHERE extension_name = 'ducklake'`).Scan(&loaded); err != nil {
		t.Fatal(err)
	}
	if !loaded {
		t.Fatal("DuckLake extension did not report as loaded")
	}
}
