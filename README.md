# codelens

A code-intelligence tool over a Go source tree. It parses a target repo into a
symbol graph and a reference graph, stores them in Postgres, and serves a
GraphQL API that the React browser uses for jump-to-definition and
find-references. A Python sidecar adds a related-symbols layer behind a
pluggable provider seam.

## Components

- `server/` Go module with two binaries:
  - `cmd/indexer` parses a Go corpus with `go/parser` and `go/ast`, building
    definitions, references, and call edges, and writes them to Postgres.
  - `cmd/api` serves GraphQL (`symbol`, `definition`, `references`, `callers`,
    `callees`, `related`) with an optional Redis cache on the hot path.
- `web/` React and TypeScript browser (Vite): look up a symbol, see its
  definition and every reference across files.
- `sidecar/` Python (FastAPI) service that ranks related symbols. The scoring
  lives behind a `SemanticProvider` seam with a deterministic offline provider,
  so the whole stack runs without external calls.
- `db/` Postgres schema and indexes.
- `corpus/sample` a small Go source tree used as the indexed corpus.

## Run it

```sh
docker compose up -d --build
# api:   http://localhost:8080/graphql
# web:   http://localhost:5173
# sidecar health: http://localhost:8000/healthz
```

The `indexer` service runs once on startup to populate the graph, then the
`api` serves queries against it.

## Query examples

```graphql
{ symbol(name: "Add") { id kind file { path } defLine } }
{ references(symbolId: "1") { refLine file { path } } }
{ callers(symbolId: "1", depth: 2) { name file { path } } }
```

## Tests

```sh
# Go (store tests need DATABASE_URL; they skip without it)
cd server && go test ./...

# Python sidecar
cd sidecar && pip install ".[dev]" && pytest

# Web
cd web && npm install && npm run test
```

## find-references latency

The find-references path is the one that gets hit constantly while browsing, so
it is the path that was optimized. The numbers below are p95 over the seeded
benchmark in `server/cmd/bench`, measured on the development machine. Reproduce
them with:

```sh
cd server
go run ./cmd/bench -dbdir ../db -files 6000 -symbols 6000 -refs 900 -iters 200
```

Seeded graph: 6000 files, 6000 symbols, 900 references per symbol (about 5.4M
reference rows).

| path | p50 | p95 |
| --- | --- | --- |
| slow baseline (no index, N+1 file lookups) | 411.0 ms | 483.6 ms |
| optimized (covering index, single join) | 4.3 ms | 6.8 ms |

What made it fast:

- The baseline scanned the whole `refs` table for a symbol, then issued one
  extra `files` query per matching row (a textbook N+1). The optimized query is
  a single `refs JOIN files` that returns the file path inline, so there is one
  round trip instead of hundreds.
- A composite index on `refs (symbol_id, file_id)` with `INCLUDE (ref_line,
  ref_col)` lets Postgres seek to a symbol's rows and answer from the index
  without heap fetches. See `db/indexes.sql`.
- Hot symbols are cached in Redis behind the API, so repeat lookups skip the
  database entirely. The cache is optional; the table above is measured without
  it to isolate the database win.

CI runs a `bench-regress` smoke gate (`server/scripts/benchregress.sh`) on a
smaller seeded graph and fails if the fast path drifts to within 30% of the slow
baseline, guarding against a lost index or a reintroduced N+1.

## How this differs

This repo is the code-intelligence and symbol-graph tool in the set. It is not
a review agent (`reviewmate`) and not a retrieval index (`vectorsearch`,
`mcp-agentlab`). The angle here is a Go AST indexer feeding a GraphQL API, with
a measured find-references latency optimization documented below.

## License

MIT. See `LICENSE`.
