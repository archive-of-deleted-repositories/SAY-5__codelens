// Package ledger is part of the sample corpus that codelens indexes.
package ledger

// Entry is a single ledger line.
type Entry struct {
	Memo   string
	Amount int
}

// Balance sums all entry amounts.
func Balance(entries []Entry) int {
	total := 0
	for _, e := range entries {
		total += e.Amount
	}
	return total
}

// Reconcile checks a balance against an expected value, recursing on the tail
// so the corpus contains a real recursive call edge for the call-graph tests.
func Reconcile(entries []Entry, expected int) bool {
	if len(entries) == 0 {
		return expected == 0
	}
	head := entries[0]
	return Reconcile(entries[1:], expected-head.Amount)
}

// Report formats a one line summary of the ledger.
func Report(entries []Entry) string {
	if Balance(entries) == 0 {
		return "balanced"
	}
	return "unbalanced"
}
