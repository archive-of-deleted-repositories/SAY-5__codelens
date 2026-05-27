package semantic

import (
	"testing"

	"github.com/SAY-5/codelens/server/internal/store"
)

func TestRankPrefersSharedTokens(t *testing.T) {
	target := &store.Symbol{ID: 1, Name: "ReadFile", Kind: "func"}
	cands := []store.Symbol{
		{ID: 1, Name: "ReadFile", Kind: "func"},
		{ID: 2, Name: "WriteFile", Kind: "func"},
		{ID: 3, Name: "ReadConfig", Kind: "func"},
		{ID: 4, Name: "Balance", Kind: "func"},
	}
	got := Rank(target, cands, 3)
	if len(got) == 0 {
		t.Fatal("expected ranked results")
	}
	if got[0].Symbol.ID != 2 && got[0].Symbol.ID != 3 {
		t.Errorf("top result should share a token, got %q", got[0].Symbol.Name)
	}
	// The unrelated symbol must not appear.
	for _, r := range got {
		if r.Symbol.Name == "Balance" {
			t.Error("Balance shares no token and should be excluded")
		}
	}
}

func TestSplitIdent(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"FindReferences", []string{"Find", "References"}},
		{"symbol_id", []string{"symbol", "id"}},
		{"Total", []string{"Total"}},
	}
	for _, c := range cases {
		got := splitIdent(c.in)
		if len(got) != len(c.want) {
			t.Errorf("splitIdent(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("splitIdent(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestRankIsDeterministic(t *testing.T) {
	target := &store.Symbol{ID: 10, Name: "BalanceTotal", Kind: "func"}
	cands := []store.Symbol{
		{ID: 11, Name: "Total", Kind: "func"},
		{ID: 12, Name: "Balance", Kind: "func"},
		{ID: 13, Name: "BalanceReport", Kind: "func"},
	}
	first := Rank(target, cands, 5)
	second := Rank(target, cands, 5)
	if len(first) != len(second) {
		t.Fatalf("length differs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Symbol.ID != second[i].Symbol.ID || first[i].Score != second[i].Score {
			t.Errorf("nondeterministic at %d: %+v vs %+v", i, first[i], second[i])
		}
	}
}
