// Command indexer parses a Go source tree and writes its symbol and reference
// graph into Postgres.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/SAY-5/codelens/server/internal/index"
	"github.com/SAY-5/codelens/server/internal/store"
)

func main() {
	root := flag.String("root", "corpus/sample", "source tree to index")
	dsn := flag.String("dsn", envOr("DATABASE_URL", ""), "postgres DSN")
	schema := flag.String("schema", "db/schema.sql", "schema migration file")
	indexes := flag.String("indexes", "", "optional index migration file applied after schema")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("DATABASE_URL or -dsn is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	st, err := store.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(ctx, *schema); err != nil {
		log.Fatalf("migrate schema: %v", err)
	}
	if *indexes != "" {
		if err := st.Migrate(ctx, *indexes); err != nil {
			log.Fatalf("migrate indexes: %v", err)
		}
	}
	if err := st.Reset(ctx); err != nil {
		log.Fatalf("reset: %v", err)
	}

	g, err := index.Parse(*root)
	if err != nil {
		log.Fatalf("parse %s: %v", *root, err)
	}
	if err := st.Persist(ctx, g); err != nil {
		log.Fatalf("persist: %v", err)
	}
	log.Printf("indexed %d files, %d symbols, %d refs, %d call edges",
		len(g.Files), len(g.Defs), len(g.Refs), len(g.Calls))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
