package store

import (
	"context"
	"os"
	"testing"

	"github.com/SAY-5/codelens/server/internal/index"
)

// testStore connects to the DATABASE_URL Postgres, applies the schema, and
// loads the sample corpus. Tests skip when no database is configured so the
// pure-unit suite still runs offline.
func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx, "../../../db/schema.sql"); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := st.Reset(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}
	g, err := index.Parse("../../../corpus/sample")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := st.Persist(ctx, g); err != nil {
		t.Fatalf("persist: %v", err)
	}
	return st
}

func TestSymbolAndDefinition(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	sym, err := st.SymbolByName(ctx, "Total")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	if sym.Kind != index.KindFunc {
		t.Errorf("kind = %q, want func", sym.Kind)
	}
	def, err := st.Definition(ctx, sym.ID)
	if err != nil {
		t.Fatalf("definition: %v", err)
	}
	if def.ID != sym.ID {
		t.Errorf("definition id = %d, want %d", def.ID, sym.ID)
	}
}

func TestReferencesSlowAndFastAgree(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	sym, err := st.SymbolByName(ctx, "Add")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	slow, err := st.ReferencesSlow(ctx, sym.ID)
	if err != nil {
		t.Fatalf("slow: %v", err)
	}
	fast, err := st.ReferencesFast(ctx, sym.ID)
	if err != nil {
		t.Fatalf("fast: %v", err)
	}
	if len(slow) != len(fast) {
		t.Fatalf("slow %d refs, fast %d refs", len(slow), len(fast))
	}
	for i := range slow {
		if slow[i].ID != fast[i].ID || fast[i].FilePath == "" {
			t.Errorf("ref %d mismatch: slow=%+v fast=%+v", i, slow[i], fast[i])
		}
	}
}

func TestRecursiveCallEdge(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	sym, err := st.SymbolByName(ctx, "Reconcile")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	callers, err := st.Callers(ctx, sym.ID)
	if err != nil {
		t.Fatalf("callers: %v", err)
	}
	found := false
	for _, c := range callers {
		if c.ID == sym.ID {
			found = true
		}
	}
	if !found {
		t.Error("Reconcile should be its own caller")
	}
}
