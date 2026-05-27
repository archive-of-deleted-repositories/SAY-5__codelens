package graph

import (
	"context"
	"strconv"
	"testing"

	"github.com/graphql-go/graphql"
)

// v3: transitive call-graph traversal. The corpus contains a two level chain
// (Audit -> Report -> Balance) and a mutual-recursion cycle (isEven <-> isOdd).

func TestTransitiveCalleesRespectDepth(t *testing.T) {
	schema, st := testSchema(t)
	ctx := context.Background()
	audit, err := st.SymbolByName(ctx, "Audit")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}

	// Depth 1 reaches Report only.
	depth1 := calleeNames(t, schema, audit.ID, 1)
	if !depth1["Report"] {
		t.Errorf("depth 1 should include Report, got %v", depth1)
	}
	if depth1["Balance"] {
		t.Errorf("depth 1 should not include Balance, got %v", depth1)
	}

	// Depth 2 reaches Balance through Report.
	depth2 := calleeNames(t, schema, audit.ID, 2)
	if !depth2["Report"] || !depth2["Balance"] {
		t.Errorf("depth 2 should include Report and Balance, got %v", depth2)
	}
}

func TestMutualRecursionTerminates(t *testing.T) {
	schema, st := testSchema(t)
	ctx := context.Background()
	even, err := st.SymbolByName(ctx, "isEven")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	// A generous depth over a cycle must terminate and include the partner,
	// without returning the start symbol itself.
	names := calleeNames(t, schema, even.ID, 10)
	if !names["isOdd"] {
		t.Errorf("isEven should reach isOdd, got %v", names)
	}
	if names["isEven"] {
		t.Errorf("traversal should not list the start symbol, got %v", names)
	}
}

func calleeNames(t *testing.T, schema graphql.Schema, id int64, depth int) map[string]bool {
	t.Helper()
	q := `{ callees(symbolId: "` + strconv.FormatInt(id, 10) + `", depth: ` + strconv.Itoa(depth) + `) { name } }`
	res := exec(t, schema, q)
	out := map[string]bool{}
	for _, c := range res.Data.(map[string]interface{})["callees"].([]interface{}) {
		out[c.(map[string]interface{})["name"].(string)] = true
	}
	return out
}
