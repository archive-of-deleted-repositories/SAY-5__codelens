package store

import (
	"context"
	"fmt"

	"github.com/SAY-5/codelens/server/internal/index"
)

// Persist writes a parsed graph into Postgres in dependency order:
// files, then symbols, then refs and call edges keyed by resolved names.
func (s *Store) Persist(ctx context.Context, g *index.Graph) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fileID := map[string]int64{}
	for _, f := range g.Files {
		var id int64
		err := tx.QueryRow(ctx,
			`INSERT INTO files (path, package, lines) VALUES ($1, $2, $3)
			 ON CONFLICT (path) DO UPDATE SET package = EXCLUDED.package, lines = EXCLUDED.lines
			 RETURNING id`, f.Path, f.Package, f.Lines).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert file %s: %w", f.Path, err)
		}
		fileID[f.Path] = id
	}

	// A name may resolve to several definitions; keep them all so callers and
	// references can fan out across every matching definition.
	symbolIDs := map[string][]int64{}
	for _, d := range g.Defs {
		fid, ok := fileID[d.Path]
		if !ok {
			continue
		}
		var id int64
		err := tx.QueryRow(ctx,
			`INSERT INTO symbols (name, kind, file_id, def_line, def_col, receiver)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			d.Name, d.Kind, fid, d.Pos.Line, d.Pos.Col, d.Receiver).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert symbol %s: %w", d.Name, err)
		}
		symbolIDs[d.Name] = append(symbolIDs[d.Name], id)
	}

	for _, r := range g.Refs {
		fid, ok := fileID[r.Path]
		if !ok {
			continue
		}
		for _, sid := range symbolIDs[r.Name] {
			if _, err := tx.Exec(ctx,
				`INSERT INTO refs (symbol_id, file_id, ref_line, ref_col)
				 VALUES ($1, $2, $3, $4)`, sid, fid, r.Pos.Line, r.Pos.Col); err != nil {
				return fmt.Errorf("insert ref %s: %w", r.Name, err)
			}
		}
	}

	for _, e := range g.Calls {
		for _, callerID := range symbolIDs[e.CallerName] {
			for _, calleeID := range symbolIDs[e.CalleeName] {
				if _, err := tx.Exec(ctx,
					`INSERT INTO call_edges (caller_id, callee_id) VALUES ($1, $2)
					 ON CONFLICT (caller_id, callee_id) DO NOTHING`,
					callerID, calleeID); err != nil {
					return fmt.Errorf("insert call edge: %w", err)
				}
			}
		}
	}

	return tx.Commit(ctx)
}
