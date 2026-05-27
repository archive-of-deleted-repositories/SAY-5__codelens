package graph

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/graphql-go/graphql"

	"github.com/SAY-5/codelens/server/internal/index"
	"github.com/SAY-5/codelens/server/internal/semantic"
	"github.com/SAY-5/codelens/server/internal/store"
)

func testSchema(t *testing.T) (graphql.Schema, *store.Store) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx, "../../../db/schema.sql"); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := st.Reset(ctx); err != nil {
		t.Fatalf("reset: %v", err)
	}
	g, err := index.Parse("../../../corpus/sample")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := st.Persist(ctx, g); err != nil {
		t.Fatalf("persist: %v", err)
	}
	schema, err := NewSchema(&Resolver{Store: st, Semantic: semantic.Fake{Lister: st}})
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	return schema, st
}

func exec(t *testing.T, schema graphql.Schema, q string) *graphql.Result {
	t.Helper()
	res := graphql.Do(graphql.Params{Schema: schema, RequestString: q, Context: context.Background()})
	if len(res.Errors) > 0 {
		t.Fatalf("query errors: %v", res.Errors)
	}
	return res
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

// The schema must expose the documented query fields with the documented shape.
func TestSchemaExposesQueryContract(t *testing.T) {
	schema, _ := testSchema(t)
	res := exec(t, schema, `{
		__type(name: "Query") { fields { name } }
	}`)
	data := res.Data.(map[string]interface{})
	fields := data["__type"].(map[string]interface{})["fields"].([]interface{})
	got := map[string]bool{}
	for _, f := range fields {
		got[f.(map[string]interface{})["name"].(string)] = true
	}
	for _, want := range []string{"symbol", "definition", "references", "callers", "callees", "related"} {
		if !got[want] {
			t.Errorf("query is missing field %q", want)
		}
	}
}

func TestSymbolQueryReturnsDefinitionShape(t *testing.T) {
	schema, _ := testSchema(t)
	res := exec(t, schema, `{ symbol(name: "Add") { name kind file { path package } defLine } }`)
	sym := res.Data.(map[string]interface{})["symbol"].(map[string]interface{})
	if sym["name"] != "Add" {
		t.Errorf("name = %v, want Add", sym["name"])
	}
	if sym["kind"] != "method" {
		t.Errorf("kind = %v, want method", sym["kind"])
	}
	file := sym["file"].(map[string]interface{})
	if file["package"] != "calc" {
		t.Errorf("package = %v, want calc", file["package"])
	}
}

func TestCallersAndRelatedQueries(t *testing.T) {
	schema, st := testSchema(t)
	ctx := context.Background()
	bal, err := st.SymbolByName(ctx, "Balance")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	// Report calls Balance, so Balance has at least one caller.
	res := exec(t, schema, `{ callers(symbolId: "`+itoa(bal.ID)+`") { name } }`)
	callers := res.Data.(map[string]interface{})["callers"].([]interface{})
	if len(callers) == 0 {
		t.Error("expected at least one caller for Balance")
	}

	// related returns scored neighbors via the deterministic fake provider.
	res = exec(t, schema, `{ related(symbolId: "`+itoa(bal.ID)+`", limit: 3) { score symbol { name } } }`)
	related := res.Data.(map[string]interface{})["related"].([]interface{})
	for _, r := range related {
		row := r.(map[string]interface{})
		if row["score"].(float64) <= 0 {
			t.Error("related score should be positive")
		}
	}
}

func TestCalleesQuery(t *testing.T) {
	schema, st := testSchema(t)
	rep, err := st.SymbolByName(context.Background(), "Report")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	res := exec(t, schema, `{ callees(symbolId: "`+itoa(rep.ID)+`", depth: 1) { name } }`)
	callees := res.Data.(map[string]interface{})["callees"].([]interface{})
	found := false
	for _, c := range callees {
		if c.(map[string]interface{})["name"] == "Balance" {
			found = true
		}
	}
	if !found {
		t.Error("Report should call Balance")
	}
}

func TestReferencesQueryResolvesFiles(t *testing.T) {
	schema, st := testSchema(t)
	sym, err := st.SymbolByName(context.Background(), "Add")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	res := exec(t, schema, `{ references(symbolId: "`+itoa(sym.ID)+`") { refLine file { path } } }`)
	refs := res.Data.(map[string]interface{})["references"].([]interface{})
	if len(refs) == 0 {
		t.Fatal("expected references")
	}
	for _, r := range refs {
		row := r.(map[string]interface{})
		if row["file"].(map[string]interface{})["path"] == "" {
			t.Error("reference missing file path")
		}
	}
}
