package cache

import (
	"context"
	"testing"
	"time"
)

// A nil Cache is a valid no-op so the API and benchmarks run without Redis.
func TestNilCacheIsNoOp(t *testing.T) {
	ctx := context.Background()
	c, err := New(ctx, "", time.Minute)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c != nil {
		t.Fatal("empty addr should yield a nil cache")
	}
	if err := c.SetJSON(ctx, "k", map[string]int{"a": 1}); err != nil {
		t.Errorf("SetJSON on nil cache: %v", err)
	}
	var out map[string]int
	ok, err := c.GetJSON(ctx, "k", &out)
	if err != nil {
		t.Errorf("GetJSON on nil cache: %v", err)
	}
	if ok {
		t.Error("nil cache should always miss")
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close on nil cache: %v", err)
	}
}
