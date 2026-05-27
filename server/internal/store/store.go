// Package store provides Postgres access for the symbol and reference graph.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// File is a parsed source file.
type File struct {
	ID      int64
	Path    string
	Package string
	Lines   int
}

// Symbol is a named definition.
type Symbol struct {
	ID       int64
	Name     string
	Kind     string
	FileID   int64
	DefLine  int
	DefCol   int
	Receiver string
}

// Ref is a use site resolving to a symbol.
type Ref struct {
	ID       int64
	SymbolID int64
	FileID   int64
	RefLine  int
	RefCol   int
	FilePath string
}

// New opens a pool against the given DSN.
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Pool exposes the underlying pool for migration and seeding helpers.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// SymbolByName returns the first symbol matching an exact name, if any.
func (s *Store) SymbolByName(ctx context.Context, name string) (*Symbol, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, kind, file_id, def_line, def_col, receiver
		 FROM symbols WHERE name = $1 ORDER BY id LIMIT 1`, name)
	var sym Symbol
	err := row.Scan(&sym.ID, &sym.Name, &sym.Kind, &sym.FileID, &sym.DefLine, &sym.DefCol, &sym.Receiver)
	if err != nil {
		return nil, err
	}
	return &sym, nil
}

// Definition returns the defining symbol for an id.
func (s *Store) Definition(ctx context.Context, symbolID int64) (*Symbol, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, kind, file_id, def_line, def_col, receiver
		 FROM symbols WHERE id = $1`, symbolID)
	var sym Symbol
	err := row.Scan(&sym.ID, &sym.Name, &sym.Kind, &sym.FileID, &sym.DefLine, &sym.DefCol, &sym.Receiver)
	if err != nil {
		return nil, err
	}
	return &sym, nil
}

// ReferencesSlow is the naive baseline used by the v2 latency study.
// It fetches the raw refs and then issues one extra query per ref to resolve
// the file path (a classic N+1), with no covering index to help the join.
func (s *Store) ReferencesSlow(ctx context.Context, symbolID int64) ([]Ref, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, symbol_id, file_id, ref_line, ref_col
		 FROM refs WHERE symbol_id = $1 ORDER BY id`, symbolID)
	if err != nil {
		return nil, err
	}
	var refs []Ref
	for rows.Next() {
		var r Ref
		if err := rows.Scan(&r.ID, &r.SymbolID, &r.FileID, &r.RefLine, &r.RefCol); err != nil {
			rows.Close()
			return nil, err
		}
		refs = append(refs, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// N+1: resolve each file path with its own round trip.
	for i := range refs {
		row := s.pool.QueryRow(ctx, `SELECT path FROM files WHERE id = $1`, refs[i].FileID)
		if err := row.Scan(&refs[i].FilePath); err != nil {
			return nil, err
		}
	}
	return refs, nil
}

// ReferencesFast is the optimized path: a single indexed join that returns the
// file path alongside each ref, so there is no N+1 and no per-row round trip.
func (s *Store) ReferencesFast(ctx context.Context, symbolID int64) ([]Ref, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT r.id, r.symbol_id, r.file_id, r.ref_line, r.ref_col, f.path
		 FROM refs r JOIN files f ON f.id = r.file_id
		 WHERE r.symbol_id = $1 ORDER BY r.id`, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []Ref
	for rows.Next() {
		var r Ref
		if err := rows.Scan(&r.ID, &r.SymbolID, &r.FileID, &r.RefLine, &r.RefCol, &r.FilePath); err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, rows.Err()
}

// Callers returns the direct caller symbols of a callee.
func (s *Store) Callers(ctx context.Context, calleeID int64) ([]Symbol, error) {
	return s.callNeighbors(ctx,
		`SELECT s.id, s.name, s.kind, s.file_id, s.def_line, s.def_col, s.receiver
		 FROM call_edges e JOIN symbols s ON s.id = e.caller_id
		 WHERE e.callee_id = $1 ORDER BY s.id`, calleeID)
}

// Callees returns the direct callee symbols of a caller.
func (s *Store) Callees(ctx context.Context, callerID int64) ([]Symbol, error) {
	return s.callNeighbors(ctx,
		`SELECT s.id, s.name, s.kind, s.file_id, s.def_line, s.def_col, s.receiver
		 FROM call_edges e JOIN symbols s ON s.id = e.callee_id
		 WHERE e.caller_id = $1 ORDER BY s.id`, callerID)
}

func (s *Store) callNeighbors(ctx context.Context, sql string, id int64) ([]Symbol, error) {
	rows, err := s.pool.Query(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Symbol
	for rows.Next() {
		var sym Symbol
		if err := rows.Scan(&sym.ID, &sym.Name, &sym.Kind, &sym.FileID, &sym.DefLine, &sym.DefCol, &sym.Receiver); err != nil {
			return nil, err
		}
		out = append(out, sym)
	}
	return out, rows.Err()
}

// AllSymbols returns every symbol, used by the semantic ranking layer.
func (s *Store) AllSymbols(ctx context.Context) ([]Symbol, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, kind, file_id, def_line, def_col, receiver FROM symbols ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Symbol
	for rows.Next() {
		var sym Symbol
		if err := rows.Scan(&sym.ID, &sym.Name, &sym.Kind, &sym.FileID, &sym.DefLine, &sym.DefCol, &sym.Receiver); err != nil {
			return nil, err
		}
		out = append(out, sym)
	}
	return out, rows.Err()
}

// FilePath returns the path for a file id.
func (s *Store) FilePath(ctx context.Context, fileID int64) (string, error) {
	var p string
	err := s.pool.QueryRow(ctx, `SELECT path FROM files WHERE id = $1`, fileID).Scan(&p)
	return p, err
}
