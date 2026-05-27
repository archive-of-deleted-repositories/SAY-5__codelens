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

## How this differs

This repo is the code-intelligence and symbol-graph tool in the set. It is not
a review agent (`reviewmate`) and not a retrieval index (`vectorsearch`,
`mcp-agentlab`). The angle here is a Go AST indexer feeding a GraphQL API, with
a measured find-references latency optimization documented below.

## License

MIT. See `LICENSE`.
