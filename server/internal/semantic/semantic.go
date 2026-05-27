// Package semantic defines the related-symbols provider seam. The default
// provider talks to the Python sidecar over HTTP; tests and offline runs use
// a deterministic in-process fake.
package semantic

import (
	"context"

	"github.com/SAY-5/codelens/server/internal/store"
)

// Related is a ranked neighbor of a symbol.
type Related struct {
	Symbol *store.Symbol
	Score  float64
}

// Provider returns symbols related to a target, highest score first.
type Provider interface {
	Related(ctx context.Context, target *store.Symbol, limit int) ([]Related, error)
}
