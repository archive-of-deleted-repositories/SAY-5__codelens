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

// Audit calls Report, which calls Balance: a two level call chain used to
// exercise transitive callee traversal.
func Audit(entries []Entry) string {
	return Report(entries)
}

// isEven and isOdd are mutually recursive, forming a cycle in the call graph
// that the bounded traversal must handle without looping forever.
func isEven(n int) bool {
	if n == 0 {
		return true
	}
	return isOdd(n - 1)
}

func isOdd(n int) bool {
	if n == 0 {
		return false
	}
	return isEven(n - 1)
}

// Parity reports whether the entry count is even, driving the mutual recursion.
func Parity(entries []Entry) bool {
	return isEven(len(entries))
}
