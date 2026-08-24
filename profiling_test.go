package duckdb

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfiling(t *testing.T) {
	db := openDbWrapper(t, ``)
	defer closeDbWrapper(t, db)

	ctx := context.Background()

	conn := openConnWrapper(t, db, ctx)
	defer closeConnWrapper(t, conn)

	_, err := GetProfilingInfo(conn)
	require.ErrorContains(t, err, errProfilingInfoEmpty.Error())

	_, err = conn.ExecContext(ctx, `PRAGMA enable_profiling = 'no_output'`)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `PRAGMA profiling_mode = 'detailed'`)
	require.NoError(t, err)

	res, err := conn.QueryContext(ctx, `SELECT range AS i FROM range(100) ORDER BY i`)
	require.NoError(t, err)
	defer closeRowsWrapper(t, res)
	for res.Next() {
		var value int64
		require.NoError(t, res.Scan(&value))
	}
	require.NoError(t, res.Close())

	info, err := GetProfilingInfo(conn)
	require.NoError(t, err)

	// DuckDB 2.0 groups root metrics into child metric groups, while older
	// releases exposed a flat root map.
	require.NotEmpty(t, info.Children, "children must not be empty")
	if grainBuild {
		require.NotEmpty(t, info.Children[0].Metrics, "child metrics must not be empty")
	} else {
		require.NotEmpty(t, info.Metrics, "metrics must not be empty")
		require.NotEmpty(t, info.Children[0].Metrics, "child metrics must not be empty")
	}

	_, err = conn.ExecContext(ctx, `PRAGMA disable_profiling`)
	require.NoError(t, err)

	_, err = GetProfilingInfo(conn)
	require.ErrorContains(t, err, errProfilingInfoEmpty.Error())
}

func TestErrProfiling(t *testing.T) {
	db := openDbWrapper(t, ``)
	defer closeDbWrapper(t, db)
	conn := openConnWrapper(t, db, context.Background())
	defer closeConnWrapper(t, conn)

	_, err := GetProfilingInfo(conn)
	testError(t, err, errProfilingInfoEmpty.Error())
}

func TestCustomProfiling(t *testing.T) {
	defer func() {
		err := os.Remove("profile.db")
		require.NoError(t, err)
	}()

	db := openDbWrapper(t, ``)
	defer closeDbWrapper(t, db)

	ctx := context.Background()

	conn := openConnWrapper(t, db, ctx)
	defer closeConnWrapper(t, conn)

	_, err := conn.ExecContext(ctx, `PRAGMA disable_checkpoint_on_shutdown`)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `SET checkpoint_threshold = '10.0 GB'`)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, `ATTACH 'profile.db'`)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, `CREATE TABLE profile.tbl (i INT)`)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, `PRAGMA enable_profiling = 'no_output'`)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `PRAGMA profiling_coverage = 'ALL'`)
	require.NoError(t, err)
	if grainBuild {
		_, err = conn.ExecContext(ctx, `SET tracked_metrics = ['*']`)
	} else {
		_, err = conn.ExecContext(ctx, `PRAGMA custom_profiling_settings='{"CHECKPOINT_LATENCY": "true"}'`)
	}
	require.NoError(t, err)

	res, err := conn.QueryContext(ctx, `CHECKPOINT profile`)
	require.NoError(t, err)
	for res.Next() {
		var value any
		require.NoError(t, res.Scan(&value))
	}
	require.NoError(t, res.Close())
	if grainBuild {
		profileRows, queryErr := conn.QueryContext(ctx, `SELECT count(*) FROM profile.tbl`)
		require.NoError(t, queryErr)
		for profileRows.Next() {
			var count int64
			require.NoError(t, profileRows.Scan(&count))
		}
		require.NoError(t, profileRows.Close())
	}

	info, err := GetProfilingInfo(conn)
	require.NoError(t, err)

	// DuckDB 2.0 exposes checkpoint profiling values in grouped child nodes.
	if grainBuild {
		require.NotEmpty(t, info.Children, "children must not be empty")
		require.NotEmpty(t, info.Children[0].Metrics, "child metrics must not be empty")
	} else {
		// Verify the legacy custom metric.
		require.NotEmpty(t, info.Metrics, "metrics must not be empty")
		latency, ok := info.Metrics["CHECKPOINT_LATENCY"]
		require.True(t, ok)
		f, err := strconv.ParseFloat(latency, 64)
		require.NoError(t, err)
		require.Positive(t, f)
	}
}
