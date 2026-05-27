// Package graph builds the GraphQL schema over the symbol and reference graph.
package graph

import (
	"context"
	"fmt"

	"github.com/graphql-go/graphql"

	"github.com/SAY-5/codelens/server/internal/cache"
	"github.com/SAY-5/codelens/server/internal/semantic"
	"github.com/SAY-5/codelens/server/internal/store"
)

// Resolver holds the dependencies the GraphQL fields read from.
type Resolver struct {
	Store    *store.Store
	Cache    *cache.Cache
	Semantic semantic.Provider
	// Fast toggles the optimized find-references path. The base and v1 layers
	// run with Fast=false; the v2 layer flips it on once the index exists.
	Fast bool
}

// SchemaString is the SDL form of the API, used by the contract tests.
const SchemaString = `
type File { id: ID!, path: String!, package: String!, lines: Int! }
type Symbol { id: ID!, name: String!, kind: String!, file: File!, defLine: Int!, defCol: Int!, receiver: String! }
type Ref { id: ID!, file: File!, refLine: Int!, refCol: Int! }
type Related { symbol: Symbol!, score: Float! }
type Query {
  symbol(name: String!): Symbol
  definition(symbolId: ID!): Symbol
  references(symbolId: ID!): [Ref!]!
  callers(symbolId: ID!, depth: Int): [Symbol!]!
  callees(symbolId: ID!, depth: Int): [Symbol!]!
  related(symbolId: ID!, limit: Int): [Related!]!
}
`

// NewSchema constructs the executable schema bound to a resolver.
func NewSchema(r *Resolver) (graphql.Schema, error) {
	fileType := graphql.NewObject(graphql.ObjectConfig{
		Name: "File",
		Fields: graphql.Fields{
			"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"path":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"package": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"lines":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
	})

	symbolType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Symbol",
		Fields: graphql.Fields{
			"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"kind":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"defLine": &graphql.Field{Type: graphql.NewNonNull(graphql.Int), Resolve: field("DefLine")},
			"defCol":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int), Resolve: field("DefCol")},
			"receiver": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"file": &graphql.Field{Type: graphql.NewNonNull(fileType), Resolve: r.resolveSymbolFile},
		},
	})

	refType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Ref",
		Fields: graphql.Fields{
			"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"refLine": &graphql.Field{Type: graphql.NewNonNull(graphql.Int), Resolve: field("RefLine")},
			"refCol":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int), Resolve: field("RefCol")},
			"file": &graphql.Field{Type: graphql.NewNonNull(fileType), Resolve: r.resolveRefFile},
		},
	})

	relatedType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Related",
		Fields: graphql.Fields{
			"score": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"symbol": &graphql.Field{Type: graphql.NewNonNull(symbolType), Resolve: func(p graphql.ResolveParams) (any, error) {
				return p.Source.(relatedRow).sym, nil
			}},
		},
	})

	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"symbol": &graphql.Field{
				Type:    symbolType,
				Args:    graphql.FieldConfigArgument{"name": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)}},
				Resolve: r.resolveSymbol,
			},
			"definition": &graphql.Field{
				Type:    symbolType,
				Args:    graphql.FieldConfigArgument{"symbolId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}},
				Resolve: r.resolveDefinition,
			},
			"references": &graphql.Field{
				Type:    graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(refType))),
				Args:    graphql.FieldConfigArgument{"symbolId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}},
				Resolve: r.resolveReferences,
			},
			"callers": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(symbolType))),
				Args: graphql.FieldConfigArgument{
					"symbolId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"depth":    &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 1},
				},
				Resolve: r.resolveCallers,
			},
			"callees": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(symbolType))),
				Args: graphql.FieldConfigArgument{
					"symbolId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"depth":    &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 1},
				},
				Resolve: r.resolveCallees,
			},
			"related": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(relatedType))),
				Args: graphql.FieldConfigArgument{
					"symbolId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"limit":    &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 5},
				},
				Resolve: r.resolveRelated,
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{Query: query})
}

// field returns a resolver that reads a named struct field by reflection-free
// switch. Kept small and explicit for the few int fields that differ in casing.
func field(name string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		switch src := p.Source.(type) {
		case *store.Symbol:
			if name == "DefLine" {
				return src.DefLine, nil
			}
			return src.DefCol, nil
		case store.Symbol:
			if name == "DefLine" {
				return src.DefLine, nil
			}
			return src.DefCol, nil
		case store.Ref:
			if name == "RefLine" {
				return src.RefLine, nil
			}
			return src.RefCol, nil
		}
		return nil, fmt.Errorf("unsupported source for %s", name)
	}
}

func ctxOf(p graphql.ResolveParams) context.Context {
	if p.Context != nil {
		return p.Context
	}
	return context.Background()
}
