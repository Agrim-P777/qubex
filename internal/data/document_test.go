package data

import (
	"context"
	"math/big"
	"strings"
	"testing"
)

func TestInputAndPrecision(t *testing.T) {
	d, e := Read(context.Background(), strings.NewReader(`{"n":123456789012345678901234567890}`), "test", 1000, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := d.Value.(map[string]any)["n"].(*big.Int); !ok {
		t.Fatalf("precision lost: %#v", d.Value)
	}
	for _, input := range []string{"{} {}", "{", "true false"} {
		if _, e := Read(context.Background(), strings.NewReader(input), "test", 1000, nil); e == nil {
			t.Fatal(input)
		}
	}
	if _, e := Read(context.Background(), strings.NewReader(`{"large":"value"}`), "test", 5, nil); e == nil {
		t.Fatal("limit not enforced")
	}
}
func TestCancellationAndSize(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := Read(ctx, strings.NewReader(`{}`), "test", 100, nil); e == nil {
		t.Fatal("not cancelled")
	}
	n, e := Size(context.Background(), map[string]any{"a": []any{1, 2}}, 1000)
	if e != nil || n != 11 {
		t.Fatal(n, e)
	}
	n, e = Size(context.Background(), strings.Repeat("x", 100), 10)
	if e != nil || n <= 10 {
		t.Fatal(n, e)
	}
}
