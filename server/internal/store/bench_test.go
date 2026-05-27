package store

import (
	"context"
	"testing"
	"time"
)

// TestFastPathBeatsSlowPath is the v2 regression guard. On a seeded graph the
// optimized references query (covering index, single join) must be meaningfully
// faster than the naive baseline (no index, N+1 file lookups). This runs with
// the integration database and is skipped offline.
func TestFastPathBeatsSlowPath(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	// A modest graph keeps the test quick while still exposing the N+1 cost.
	if err := st.Seed(ctx, SeedSpec{Files: 500, Symbols: 800, RefsPerSym: 120}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() { _ = st.Reset(context.Background()) })

	const sampleID = 42

	// Baseline without the index.
	dropV2Indexes(t, st)
	slow := medianLatency(t, ctx, st.ReferencesSlow, sampleID)

	// Optimized with the covering index.
	if err := st.Migrate(ctx, "../../../db/indexes.sql"); err != nil {
		t.Fatalf("indexes: %v", err)
	}
	fast := medianLatency(t, ctx, st.ReferencesFast, sampleID)

	t.Logf("slow median %v, fast median %v", slow, fast)
	if fast >= slow {
		t.Errorf("fast path (%v) is not faster than slow path (%v)", fast, slow)
	}
	// Guard against the optimization silently regressing: the fast path should
	// be a large fraction faster, not a marginal win.
	if fast > slow/2 {
		t.Errorf("fast path %v should be well under half of slow %v", fast, slow)
	}
}

func dropV2Indexes(t *testing.T, st *Store) {
	t.Helper()
	ctx := context.Background()
	for _, idx := range []string{"idx_refs_symbol_covering", "idx_symbols_file"} {
		if _, err := st.Pool().Exec(ctx, "DROP INDEX IF EXISTS "+idx); err != nil {
			t.Fatalf("drop %s: %v", idx, err)
		}
	}
	if _, err := st.Pool().Exec(ctx, "ANALYZE refs, symbols"); err != nil {
		t.Fatalf("analyze: %v", err)
	}
}

func medianLatency(t *testing.T, ctx context.Context, fn func(context.Context, int64) ([]Ref, error), id int64) time.Duration {
	t.Helper()
	const n = 11
	durs := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		if _, err := fn(ctx, id); err != nil {
			t.Fatalf("query: %v", err)
		}
		durs = append(durs, time.Since(start))
	}
	for i := 0; i < len(durs); i++ {
		for j := i + 1; j < len(durs); j++ {
			if durs[j] < durs[i] {
				durs[i], durs[j] = durs[j], durs[i]
			}
		}
	}
	return durs[len(durs)/2]
}
