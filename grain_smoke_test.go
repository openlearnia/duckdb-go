//go:build duckdb_grain

package duckdb

import (
	"context"
	"database/sql"
	"testing"
)

func TestGrainDuckDBMaterializedViewContract(t *testing.T) {
	connector, err := NewConnector(":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	database := sql.OpenDB(connector)
	defer database.Close()
	ctx := context.Background()

	for _, statement := range []string{
		"CREATE TABLE grain_mv_source (id INTEGER, amount INTEGER)",
		"INSERT INTO grain_mv_source VALUES (1, 10), (2, 20)",
		"CREATE MATERIALIZED VIEW grain_mv AS SELECT sum(amount) AS total FROM grain_mv_source",
		"INSERT INTO grain_mv_source VALUES (3, 30)",
		"REFRESH MATERIALIZED VIEW grain_mv",
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}

	var total int64
	if err := database.QueryRowContext(ctx, "SELECT total FROM grain_mv").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 60 {
		t.Fatalf("materialized-view total = %d, want 60", total)
	}

	var name string
	if err := database.QueryRowContext(ctx, "SELECT view_name FROM duckdb_materialized_views() WHERE view_name = 'grain_mv'").Scan(&name); err != nil {
		t.Fatal(err)
	}
}
