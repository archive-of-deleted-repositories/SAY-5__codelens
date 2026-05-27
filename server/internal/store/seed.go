package store

import (
	"context"
	"fmt"
)

// SeedSpec describes a synthetic graph for the latency benchmark.
type SeedSpec struct {
	Files      int
	Symbols    int
	RefsPerSym int
}

// Seed builds a large synthetic graph directly in Postgres using generate_series
// so the find-references benchmark runs against a realistic row count. It spreads
// each symbol's references across many files, which is what makes the naive
// per-row file lookup expensive.
func (s *Store) Seed(ctx context.Context, spec SeedSpec) error {
	if err := s.Reset(ctx); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO files (path, package, lines)
		 SELECT 'pkg/file_' || g || '.go', 'pkg' || (g % 50), 100 + (g % 400)
		 FROM generate_series(1, $1) g`, spec.Files); err != nil {
		return fmt.Errorf("seed files: %w", err)
	}
	if _, err := s.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO symbols (name, kind, file_id, def_line, def_col, receiver)
		 SELECT 'Sym' || g, 'func',
		        1 + (g %% %d),
		        1 + (g %% 500), 1, ''
		 FROM generate_series(1, $1) g`, spec.Files),
		spec.Symbols); err != nil {
		return fmt.Errorf("seed symbols: %w", err)
	}
	// Each symbol gets RefsPerSym references scattered across files.
	if _, err := s.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO refs (symbol_id, file_id, ref_line, ref_col)
		 SELECT sym.id,
		        1 + ((sym.id * 7 + r) %% %d),
		        1 + (r %% 1000), 1
		 FROM symbols sym
		 CROSS JOIN generate_series(1, $1) r`, spec.Files),
		spec.RefsPerSym); err != nil {
		return fmt.Errorf("seed refs: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `ANALYZE files, symbols, refs`); err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	return nil
}
