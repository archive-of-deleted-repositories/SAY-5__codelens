package index

import "testing"

func TestParseSampleCorpus(t *testing.T) {
	g, err := Parse("../../../corpus/sample")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(g.Files) < 2 {
		t.Fatalf("want at least 2 files, got %d", len(g.Files))
	}

	defByName := map[string]Def{}
	for _, d := range g.Defs {
		defByName[d.Name] = d
	}
	for _, want := range []struct {
		name string
		kind string
	}{
		{"Add", KindMethod},
		{"Total", KindFunc},
		{"Entry", KindType},
		{"Reconcile", KindFunc},
	} {
		got, ok := defByName[want.name]
		if !ok {
			t.Errorf("missing definition %s", want.name)
			continue
		}
		if got.Kind != want.kind {
			t.Errorf("%s: kind = %q, want %q", want.name, got.Kind, want.kind)
		}
	}
}

func TestParseFindsRecursiveCallEdge(t *testing.T) {
	g, err := Parse("../../../corpus/sample")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found := false
	for _, e := range g.Calls {
		if e.CallerName == "Reconcile" && e.CalleeName == "Reconcile" {
			found = true
		}
	}
	if !found {
		t.Error("expected a self call edge for Reconcile")
	}
}
