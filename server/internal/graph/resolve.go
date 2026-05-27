package graph

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/graphql-go/graphql"

	"github.com/SAY-5/codelens/server/internal/store"
)

type relatedRow struct {
	sym   *store.Symbol
	score float64
}

func argID(p graphql.ResolveParams, name string) (int64, error) {
	raw, ok := p.Args[name]
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	switch v := raw.(type) {
	case string:
		return strconv.ParseInt(v, 10, 64)
	case int:
		return int64(v), nil
	}
	return 0, fmt.Errorf("bad %s", name)
}

func argInt(p graphql.ResolveParams, name string, def int) int {
	if v, ok := p.Args[name].(int); ok {
		return v
	}
	return def
}

func (r *Resolver) resolveSymbol(p graphql.ResolveParams) (any, error) {
	name, _ := p.Args["name"].(string)
	sym, err := r.Store.SymbolByName(ctxOf(p), name)
	if err != nil {
		return nil, nil //nolint:nilerr // no symbol resolves to null, not an error
	}
	return sym, nil
}

func (r *Resolver) resolveDefinition(p graphql.ResolveParams) (any, error) {
	id, err := argID(p, "symbolId")
	if err != nil {
		return nil, err
	}
	sym, err := r.Store.Definition(ctxOf(p), id)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	return sym, nil
}

func (r *Resolver) resolveReferences(p graphql.ResolveParams) (any, error) {
	id, err := argID(p, "symbolId")
	if err != nil {
		return nil, err
	}
	ctx := ctxOf(p)
	key := fmt.Sprintf("refs:%d", id)
	if r.Fast {
		var cached []store.Ref
		if ok, _ := r.Cache.GetJSON(ctx, key, &cached); ok {
			return cached, nil
		}
		refs, err := r.Store.ReferencesFast(ctx, id)
		if err != nil {
			return nil, err
		}
		_ = r.Cache.SetJSON(ctx, key, refs)
		return refs, nil
	}
	return r.Store.ReferencesSlow(ctx, id)
}

func (r *Resolver) resolveCallers(p graphql.ResolveParams) (any, error) {
	id, err := argID(p, "symbolId")
	if err != nil {
		return nil, err
	}
	depth := argInt(p, "depth", 1)
	return r.transitive(ctxOf(p), id, depth, r.Store.Callers)
}

func (r *Resolver) resolveCallees(p graphql.ResolveParams) (any, error) {
	id, err := argID(p, "symbolId")
	if err != nil {
		return nil, err
	}
	depth := argInt(p, "depth", 1)
	return r.transitive(ctxOf(p), id, depth, r.Store.Callees)
}

func (r *Resolver) resolveRelated(p graphql.ResolveParams) (any, error) {
	id, err := argID(p, "symbolId")
	if err != nil {
		return nil, err
	}
	limit := argInt(p, "limit", 5)
	ctx := ctxOf(p)
	target, err := r.Store.Definition(ctx, id)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	ranked, err := r.Semantic.Related(ctx, target, limit)
	if err != nil {
		return nil, err
	}
	out := make([]relatedRow, 0, len(ranked))
	for _, rr := range ranked {
		out = append(out, relatedRow{sym: rr.Symbol, score: rr.Score})
	}
	return out, nil
}

func (r *Resolver) resolveSymbolFile(p graphql.ResolveParams) (any, error) {
	var fileID int64
	switch s := p.Source.(type) {
	case *store.Symbol:
		fileID = s.FileID
	case store.Symbol:
		fileID = s.FileID
	default:
		return nil, fmt.Errorf("bad symbol source")
	}
	return r.fileByID(p, fileID)
}

func (r *Resolver) resolveRefFile(p graphql.ResolveParams) (any, error) {
	ref, ok := p.Source.(store.Ref)
	if !ok {
		return nil, fmt.Errorf("bad ref source")
	}
	if ref.FilePath != "" {
		return store.File{ID: ref.FileID, Path: ref.FilePath}, nil
	}
	return r.fileByID(p, ref.FileID)
}

func (r *Resolver) fileByID(p graphql.ResolveParams, fileID int64) (any, error) {
	f, err := r.Store.FileByID(ctxOf(p), fileID)
	if err != nil {
		return nil, err
	}
	return *f, nil
}

// transitive runs a bounded BFS over the call graph, deduping visited symbols
// and tolerating cycles. depth 1 returns direct neighbors only.
func (r *Resolver) transitive(
	ctx context.Context, start int64, depth int,
	step func(context.Context, int64) ([]store.Symbol, error),
) (any, error) {
	if depth < 1 {
		depth = 1
	}
	visited := map[int64]bool{start: true}
	frontier := []int64{start}
	var result []store.Symbol
	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []int64
		for _, id := range frontier {
			neighbors, err := step(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, n := range neighbors {
				if visited[n.ID] {
					continue
				}
				visited[n.ID] = true
				result = append(result, n)
				next = append(next, n.ID)
			}
		}
		frontier = next
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
