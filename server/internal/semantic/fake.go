package semantic

import (
	"context"

	"github.com/SAY-5/codelens/server/internal/store"
)

// CandidateLister supplies the symbols a provider may rank.
type CandidateLister interface {
	AllSymbols(ctx context.Context) ([]store.Symbol, error)
}

// Fake is a deterministic provider used in tests and offline CI. It scores
// candidates by a stable token-overlap similarity against the target name and
// kind, so the same input always yields the same ranking. No network calls.
type Fake struct {
	Lister CandidateLister
}

// Related ranks all other symbols by similarity to the target.
func (f Fake) Related(ctx context.Context, target *store.Symbol, limit int) ([]Related, error) {
	cands, err := f.Lister.AllSymbols(ctx)
	if err != nil {
		return nil, err
	}
	return Rank(target, cands, limit), nil
}
