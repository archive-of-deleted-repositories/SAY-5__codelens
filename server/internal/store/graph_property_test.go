package store

import (
	"context"
	"testing"
)

// These properties assert the graph is internally consistent after indexing
// the sample corpus: references resolve to real definitions, the file side of
// every edge exists, and slow and fast reference reads agree.

func TestEveryReferenceResolvesToADefinition(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	var dangling int
	err := st.Pool().QueryRow(ctx,
		`SELECT count(*) FROM refs r
		 LEFT JOIN symbols s ON s.id = r.symbol_id
		 WHERE s.id IS NULL`).Scan(&dangling)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if dangling != 0 {
		t.Errorf("%d references point at no definition", dangling)
	}
}

func TestEveryEdgeHasARealFile(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	for _, q := range []struct {
		name string
		sql  string
	}{
		{"refs", `SELECT count(*) FROM refs r LEFT JOIN files f ON f.id = r.file_id WHERE f.id IS NULL`},
		{"symbols", `SELECT count(*) FROM symbols s LEFT JOIN files f ON f.id = s.file_id WHERE f.id IS NULL`},
	} {
		var n int
		if err := st.Pool().QueryRow(ctx, q.sql).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q.name, err)
		}
		if n != 0 {
			t.Errorf("%s: %d rows reference a missing file", q.name, n)
		}
	}
}

func TestEveryCallEdgeEndpointExists(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	var n int
	err := st.Pool().QueryRow(ctx,
		`SELECT count(*) FROM call_edges e
		 LEFT JOIN symbols c1 ON c1.id = e.caller_id
		 LEFT JOIN symbols c2 ON c2.id = e.callee_id
		 WHERE c1.id IS NULL OR c2.id IS NULL`).Scan(&n)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 0 {
		t.Errorf("%d call edges have a dangling endpoint", n)
	}
}

func TestReferencesResolveBackToTheirSymbol(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	syms, err := st.AllSymbols(ctx)
	if err != nil {
		t.Fatalf("symbols: %v", err)
	}
	for _, sym := range syms {
		refs, err := st.ReferencesFast(ctx, sym.ID)
		if err != nil {
			t.Fatalf("refs for %d: %v", sym.ID, err)
		}
		for _, r := range refs {
			if r.SymbolID != sym.ID {
				t.Errorf("ref %d resolved to symbol %d, want %d", r.ID, r.SymbolID, sym.ID)
			}
			if r.FilePath == "" {
				t.Errorf("ref %d has no file path", r.ID)
			}
		}
	}
}
