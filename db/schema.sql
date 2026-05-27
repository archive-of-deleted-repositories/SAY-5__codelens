-- codelens schema: symbol graph and reference graph over a parsed source tree.
-- The base schema deliberately ships without the find-references composite
-- index. The v2 layer adds it; see db/indexes.sql and the README latency notes.

CREATE TABLE IF NOT EXISTS files (
    id         BIGSERIAL PRIMARY KEY,
    path       TEXT NOT NULL UNIQUE,
    package    TEXT NOT NULL,
    lines      INTEGER NOT NULL DEFAULT 0
);

-- A symbol is a named definition (func, method, type, var, const).
CREATE TABLE IF NOT EXISTS symbols (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL,
    file_id     BIGINT NOT NULL REFERENCES files(id),
    def_line    INTEGER NOT NULL,
    def_col     INTEGER NOT NULL,
    receiver    TEXT NOT NULL DEFAULT ''
);

-- A reference is a use site that resolves to a symbol definition.
CREATE TABLE IF NOT EXISTS refs (
    id          BIGSERIAL PRIMARY KEY,
    symbol_id   BIGINT NOT NULL REFERENCES symbols(id),
    file_id     BIGINT NOT NULL REFERENCES files(id),
    ref_line    INTEGER NOT NULL,
    ref_col     INTEGER NOT NULL
);

-- A call edge connects a caller symbol to a callee symbol.
CREATE TABLE IF NOT EXISTS call_edges (
    id          BIGSERIAL PRIMARY KEY,
    caller_id   BIGINT NOT NULL REFERENCES symbols(id),
    callee_id   BIGINT NOT NULL REFERENCES symbols(id),
    UNIQUE (caller_id, callee_id)
);

-- Lookup by name is used by symbol(name) queries.
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols (name);
