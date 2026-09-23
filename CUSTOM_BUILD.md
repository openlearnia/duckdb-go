# Grain DuckDB 2.0 preview driver

This fork keeps the `database/sql` API of `duckdb-go/v2` and uses the
DuckDB 2.0 preview C API through `github.com/openlearnia/duckdb-go-bindings`.
Build applications with the `duckdb_grain` tag and the matching custom
DuckDB runtime:

```sh
CGO_ENABLED=1 \
CGO_CFLAGS="-I$DUCKDB_PREFIX/include" \
CGO_LDFLAGS="-L$DUCKDB_PREFIX/lib -lduckdb" \
LD_LIBRARY_PATH="$DUCKDB_PREFIX/lib" \
go build -tags=duckdb_grain ./...
```

The DuckLake extension is loaded separately by the application using the
matching `.duckdb_extension` artifact from the paired Grain release.
