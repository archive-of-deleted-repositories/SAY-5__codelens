package store

import (
	"context"
	"fmt"
	"os"
)

// Migrate runs a SQL file against the pool. Used for schema and index files.
func (s *Store) Migrate(ctx context.Context, sqlPath string) error {
	b, err := os.ReadFile(sqlPath) //nolint:gosec // path supplied by operator at startup
	if err != nil {
		return fmt.Errorf("read migration %s: %w", sqlPath, err)
	}
	if _, err := s.pool.Exec(ctx, string(b)); err != nil {
		return fmt.Errorf("apply migration %s: %w", sqlPath, err)
	}
	return nil
}

// Reset truncates all graph tables. Used by the indexer for a clean rebuild
// and by benchmark seeding.
func (s *Store) Reset(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`TRUNCATE call_edges, refs, symbols, files RESTART IDENTITY CASCADE`)
	return err
}
