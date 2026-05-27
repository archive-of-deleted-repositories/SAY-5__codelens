// Command api serves the GraphQL endpoint over the symbol and reference graph.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/SAY-5/codelens/server/internal/cache"
	"github.com/SAY-5/codelens/server/internal/graph"
	"github.com/SAY-5/codelens/server/internal/semantic"
	"github.com/SAY-5/codelens/server/internal/store"
)

func main() {
	ctx := context.Background()

	st, err := store.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	c, err := cache.New(ctx, os.Getenv("REDIS_ADDR"), 5*time.Minute)
	if err != nil {
		log.Printf("redis unavailable, running without cache: %v", err)
	}

	var provider semantic.Provider
	if base := os.Getenv("SIDECAR_URL"); base != "" {
		provider = semantic.Sidecar{BaseURL: base, Lister: st}
	} else {
		provider = semantic.Fake{Lister: st}
	}

	resolver := &graph.Resolver{
		Store:    st,
		Cache:    c,
		Semantic: provider,
		Fast:     envBool("FAST_PATH", false),
	}
	schema, err := graph.NewSchema(resolver)
	if err != nil {
		log.Fatalf("schema: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/graphql", graph.Handler(schema))

	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("api starting (fast=%v)", resolver.Fast)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func envBool(k string, def bool) bool {
	if v, ok := os.LookupEnv(k); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
