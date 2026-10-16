package execute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/query"
	"math/big"
	"sync"
	"testing"
	"time"
)

func TestRunRecoversFromPanic(t *testing.T) {
	r := New()
	out := r.Run(context.Background(), query.New(), 1, 1, Limits{}, func(any) error { panic("boom") })
	if out.Status != "error" || out.Error == nil {
		t.Fatalf("panic in emit should surface as a clean error: %+v", out)
	}
}

func TestLimitsAndFullCollection(t *testing.T) {
	r := New()
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "i", Kind: "iterate"}, {ID: "c", Kind: "collect"}, {ID: "s", Kind: "sort", Descending: true}}}
	out := r.Run(context.Background(), q, []any{1, 3, 2}, 5, Limits{Results: 1, Bytes: 1000}, nil)
	if out.Status != "complete" || out.Revision != 5 || fmt.Sprint(out.Values) != "[[3 2 1]]" {
		t.Fatal(out)
	}
	q.Operations = q.Operations[:1]
	out = r.Run(context.Background(), q, []any{1, 2, 3}, 1, Limits{Results: 2}, nil)
	if out.Status != "truncated" || out.Count != 2 {
		t.Fatal(out)
	}
	out = r.Run(context.Background(), query.New(), "too large", 1, Limits{Bytes: 3}, nil)
	if out.Status != "truncated" || out.Count != 0 {
		t.Fatal(out)
	}
}

func TestImmutableInputAndExactIntegerArithmetic(t *testing.T) {
	r := New()
	input := map[string]any{"name": "api"}
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "u", Kind: "update", Path: query.Path{query.Field("name")}, Transform: &query.Transform{Kind: "upper"}}}}
	out := r.Run(context.Background(), q, input, 1, Limits{}, nil)
	if out.Error != nil || input["name"] != "api" {
		t.Fatal(input, out)
	}
	n, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
	q.Operations = []query.Operation{{ID: "a", Kind: "transform", Transform: &query.Transform{Kind: "add", Value: json.Number("1")}}}
	out = r.Run(context.Background(), q, n, 1, Limits{}, nil)
	if out.Error != nil || fmt.Sprint(out.Values[0]) != "123456789012345678901234567891" {
		t.Fatal(out)
	}
}

func TestCancelledRequestDoesNotWaitForBusyCoordinator(t *testing.T) {
	r := New()
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(context.Background(), query.New(), 1, 1, Limits{}, func(any) error { close(entered); <-release; return nil })
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := r.Run(ctx, query.New(), 1, 2, Limits{}, nil)
	close(release)
	<-done
	if out.Status != "cancelled" {
		t.Fatal(out)
	}
}
func TestCancellationErrorsAndConcurrentCache(t *testing.T) {
	r := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := r.Run(ctx, query.New(), 1, 1, Limits{}, nil)
	if out.Status != "cancelled" {
		t.Fatal(out)
	}
	out = r.Run(context.Background(), query.New(), 1, 1, Limits{Timeout: time.Nanosecond}, nil)
	if out.Status != "timeout" {
		t.Fatal(out)
	}
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "bad", Kind: "iterate"}}}
	out = r.Run(context.Background(), q, nil, 1, Limits{}, nil)
	if out.Error == nil || out.OperationID != "bad" {
		t.Fatal(out)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r.Run(context.Background(), query.New(), true, 1, Limits{}, nil).Error != nil {
				t.Error("run failed")
			}
		}()
	}
	wg.Wait()
	out = r.Run(context.Background(), query.New(), 1, 1, Limits{}, func(any) error { return errors.New("write failed") })
	if out.Status != "io_error" {
		t.Fatal(out)
	}
}
