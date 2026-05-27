// Package calc is part of the sample corpus that codelens indexes.
package calc

// Op is a binary arithmetic operation.
type Op struct {
	Left  int
	Right int
}

// Add returns the sum of the operands.
func (o Op) Add() int {
	return o.Left + o.Right
}

// Sub returns the difference of the operands.
func (o Op) Sub() int {
	return o.Left - o.Right
}

// Total folds Add over a slice of operations.
func Total(ops []Op) int {
	sum := 0
	for _, op := range ops {
		sum += op.Add()
	}
	return sum
}

// Net folds Sub over a slice of operations.
func Net(ops []Op) int {
	net := 0
	for _, op := range ops {
		net += op.Sub()
	}
	return net
}
