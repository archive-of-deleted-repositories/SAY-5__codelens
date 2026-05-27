-- v2 latency indexes for the find-references hot path.
--
-- The baseline (db/schema.sql) ships with no index on refs.symbol_id, so a
-- references lookup scans the whole refs table, and the API then issues one
-- file lookup per row (N+1). These indexes plus a single join remove both.
--
-- A composite index on (symbol_id, file_id) lets the references query seek
-- straight to a symbol's rows. INCLUDE carries ref_line and ref_col so the
-- planner can answer the query from the index without heap fetches.
CREATE INDEX IF NOT EXISTS idx_refs_symbol_covering
    ON refs (symbol_id, file_id) INCLUDE (ref_line, ref_col);

-- Speeds the symbols.file_id lookups behind the file join.
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols (file_id);

-- Call-graph traversal seeks by caller and by callee.
CREATE INDEX IF NOT EXISTS idx_call_edges_callee ON call_edges (callee_id);
CREATE INDEX IF NOT EXISTS idx_call_edges_caller ON call_edges (caller_id);
