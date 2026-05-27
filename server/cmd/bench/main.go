// Command bench measures find-references p95 latency for the slow baseline and
// the optimized fast path against a large seeded graph. It reports real numbers
// from the run; nothing is hardcoded.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/SAY-5/codelens/server/internal/cache"
	"github.com/SAY-5/codelens/server/internal/store"
)

func main() {
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "postgres DSN")
	redisAddr := flag.String("redis", os.Getenv("REDIS_ADDR"), "redis addr (optional)")
	files := flag.Int("files", 4000, "seeded files")
	symbols := flag.Int("symbols", 8000, "seeded symbols")
	refsPer := flag.Int("refs", 300, "references per symbol")
	iters := flag.Int("iters", 400, "queries per measurement")
	seed := flag.Bool("seed", true, "reseed the graph before measuring")
	dbDir := flag.String("dbdir", "db", "directory holding schema.sql and indexes.sql")
	jsonOut := flag.String("json", "", "optional path to write results as json")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("DATABASE_URL or -dsn is required")
	}
	ctx := context.Background()
	st, err := store.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	schemaPath := filepath.Join(*dbDir, "schema.sql")
	indexPath := filepath.Join(*dbDir, "indexes.sql")
	if err := st.Migrate(ctx, schemaPath); err != nil {
		log.Fatalf("schema: %v", err)
	}
	if *seed {
		log.Printf("seeding %d files, %d symbols, %d refs/symbol", *files, *symbols, *refsPer)
		if err := st.Seed(ctx, store.SeedSpec{Files: *files, Symbols: *symbols, RefsPerSym: *refsPer}); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	ids := sampleIDs(*symbols, *iters)

	// Baseline: no index, N+1 file lookups, no cache.
	dropIndexes(ctx, st)
	slow := measure(ctx, *iters, func(i int) error {
		_, err := st.ReferencesSlow(ctx, ids[i%len(ids)])
		return err
	})

	// Optimized: add the covering index, single join, optional Redis cache.
	if err := st.Migrate(ctx, indexPath); err != nil {
		log.Fatalf("indexes: %v", err)
	}
	c, _ := cache.New(ctx, *redisAddr, 5*time.Minute)
	fast := measure(ctx, *iters, func(i int) error {
		id := ids[i%len(ids)]
		key := fmt.Sprintf("refs:%d", id)
		var cached []store.Ref
		if ok, _ := c.GetJSON(ctx, key, &cached); ok {
			return nil
		}
		refs, err := st.ReferencesFast(ctx, id)
		if err != nil {
			return err
		}
		return c.SetJSON(ctx, key, refs)
	})

	report := map[string]any{
		"files": *files, "symbols": *symbols, "refs_per_symbol": *refsPer,
		"iterations":    *iters,
		"slow_p95_ms":   ms(slow.p95),
		"slow_p50_ms":   ms(slow.p50),
		"fast_p95_ms":   ms(fast.p95),
		"fast_p50_ms":   ms(fast.p50),
		"cache_enabled": c != nil,
	}
	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))
	fmt.Printf("\nfind-references p95: %.1fms -> %.1fms\n", ms(slow.p95), ms(fast.p95))

	if *jsonOut != "" {
		if err := os.WriteFile(*jsonOut, out, 0o600); err != nil {
			log.Fatalf("write json: %v", err)
		}
	}
}

func dropIndexes(ctx context.Context, st *store.Store) {
	for _, idx := range []string{
		"idx_refs_symbol_covering", "idx_symbols_file",
		"idx_call_edges_callee", "idx_call_edges_caller",
	} {
		_, _ = st.Pool().Exec(ctx, "DROP INDEX IF EXISTS "+idx)
	}
	_, _ = st.Pool().Exec(ctx, "ANALYZE refs, symbols")
}

type result struct{ p50, p95 time.Duration }

func measure(ctx context.Context, iters int, fn func(int) error) result {
	_ = ctx
	durs := make([]time.Duration, 0, iters)
	for i := 0; i < iters; i++ {
		start := time.Now()
		if err := fn(i); err != nil {
			log.Fatalf("query %d: %v", i, err)
		}
		durs = append(durs, time.Since(start))
	}
	sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
	return result{p50: pct(durs, 0.50), p95: pct(durs, 0.95)}
}

func pct(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p * float64(len(sorted)-1))
	return sorted[idx]
}

func sampleIDs(symbols, n int) []int64 {
	r := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic sampling for a benchmark, not security
	ids := make([]int64, n)
	for i := range ids {
		ids[i] = int64(1 + r.Intn(symbols))
	}
	return ids
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000.0 }
