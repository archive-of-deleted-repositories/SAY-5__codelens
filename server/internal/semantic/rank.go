package semantic

import (
	"sort"
	"strings"
	"unicode"

	"github.com/SAY-5/codelens/server/internal/store"
)

// Rank scores candidates against a target and returns the top results. The
// score is a deterministic blend of identifier-token overlap (Jaccard) and a
// kind-match bonus. Ties break by symbol id so the order is stable.
func Rank(target *store.Symbol, candidates []store.Symbol, limit int) []Related {
	if limit <= 0 {
		limit = 5
	}
	targetTokens := tokenSet(target.Name)
	out := make([]Related, 0, len(candidates))
	for i := range candidates {
		c := candidates[i]
		if c.ID == target.ID {
			continue
		}
		score := jaccard(targetTokens, tokenSet(c.Name))
		if score <= 0 {
			continue
		}
		if c.Kind == target.Kind {
			score += 0.1
		}
		sym := c
		out = append(out, Related{Symbol: &sym, Score: round(score)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Symbol.ID < out[j].Symbol.ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// tokenSet splits an identifier into lowercased camelCase and snake_case parts.
func tokenSet(name string) map[string]bool {
	set := map[string]bool{}
	for _, part := range splitIdent(name) {
		if part != "" {
			set[strings.ToLower(part)] = true
		}
	}
	return set
}

func splitIdent(name string) []string {
	var parts []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			parts = append(parts, cur.String())
			cur.Reset()
		}
	}
	var prev rune
	for i, r := range name {
		switch {
		case r == '_' || r == '-':
			flush()
		case unicode.IsUpper(r) && i > 0 && !unicode.IsUpper(prev):
			flush()
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
		prev = r
	}
	flush()
	return parts
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func round(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}
