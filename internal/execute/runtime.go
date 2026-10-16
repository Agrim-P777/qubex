package execute

import (
	"context"
	"errors"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/compiler"
	"github.com/Agrim-P777/qubex/internal/data"
	"github.com/Agrim-P777/qubex/internal/query"
	"github.com/itchyny/gojq"
	"sync"
	"time"
)

type Limits struct {
	Timeout time.Duration
	Results int
	Bytes   int64
}

func PreviewLimits() Limits { return Limits{2 * time.Second, 10000, 8 << 20} }

// RunLimits bounds headless `history run`. A finite wall-clock timeout keeps a
// runaway or hostile-input query from spinning forever; result/byte caps stay
// unlimited so a legitimate full run is not silently truncated.
func RunLimits() Limits { return Limits{Timeout: 60 * time.Second} }

type ExecutionResult struct {
	Revision    uint64
	Values      []any
	Status      string
	Error       error
	OperationID string
	Count       int
	Bytes       int64
	Duration    time.Duration
}
type Runtime struct {
	mu    sync.Mutex
	cache map[string]*gojq.Code
	gate  chan struct{}
}

func New() *Runtime { return &Runtime{cache: map[string]*gojq.Code{}, gate: make(chan struct{}, 1)} }
func (r *Runtime) code(text string) (*gojq.Code, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c := r.cache[text]; c != nil {
		return c, nil
	}
	q, e := gojq.Parse(text)
	if e != nil {
		return nil, e
	}
	c, e := gojq.Compile(q)
	if e != nil {
		return nil, e
	}
	if len(r.cache) >= 64 {
		r.cache = map[string]*gojq.Code{}
	}
	r.cache[text] = c
	return c, nil
}

// Run's optional emit callback is used by the CLI to avoid buffering results.
func (r *Runtime) Run(ctx context.Context, q query.QueryDocument, input any, revision uint64, limits Limits, emit func(any) error) (out ExecutionResult) {
	out.Revision = revision
	out.Status = "complete"
	start := time.Now()
	defer func() { out.Duration = time.Since(start) }()
	// gojq is normally robust, but a panic in the interpreter (or the emit
	// callback) must not crash a headless run with a raw stack trace.
	defer func() {
		if rec := recover(); rec != nil {
			out.Status = "error"
			out.Error = fmt.Errorf("query execution panicked: %v", rec)
		}
	}()
	if limits.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, limits.Timeout)
		defer cancel()
	}
	// Only one interpreter runs per coordinator. Superseded requests waiting
	// for a cooperative cancellation can leave without acquiring the gate.
	select {
	case r.gate <- struct{}{}:
		defer func() { <-r.gate }()
	case <-ctx.Done():
		out.Error = ctx.Err()
		out.Status = status(out.Error)
		return
	}
	if e := ctx.Err(); e != nil {
		out.Error = e
		out.Status = status(e)
		return
	}
	c, e := compiler.Compile(q)
	if e != nil {
		out.Status = "error"
		out.Error = e
		return
	}
	code, e := r.code(c.Text)
	if e != nil {
		out.Status = "error"
		out.Error = e
		return
	}
	it := code.RunWithContext(ctx, input)
	for {
		if e := ctx.Err(); e != nil {
			out.Error = e
			out.Status = status(e)
			return
		}
		v, ok := it.Next()
		if !ok {
			if out.Count == 0 {
				out.Status = "empty"
			}
			return
		}
		if e, ok := v.(error); ok {
			if h, ok := e.(*gojq.HaltError); ok && h.Value() == nil {
				return
			}
			out.Error = e
			out.Status = status(e)
			if out.Status == "error" {
				out.OperationID = r.locate(ctx, q, input, limits)
			}
			return
		}
		if limits.Results > 0 && out.Count >= limits.Results {
			out.Status = "truncated"
			return
		}
		if limits.Bytes > 0 {
			n, e := data.Size(ctx, v, limits.Bytes-out.Bytes)
			if e != nil {
				out.Error = e
				out.Status = status(e)
				return
			}
			if out.Bytes+n > limits.Bytes {
				out.Status = "truncated"
				return
			}
			out.Bytes += n
		}
		if emit != nil {
			if e := emit(v); e != nil {
				out.Error = e
				out.Status = "io_error"
				return
			}
		} else {
			out.Values = append(out.Values, v)
		}
		out.Count++
	}
}
func status(e error) string {
	if errors.Is(e, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(e, context.Canceled) {
		return "cancelled"
	}
	return "error"
}
func (r *Runtime) locate(ctx context.Context, q query.QueryDocument, input any, limits Limits) string {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	for n := 1; n <= len(q.Operations); n++ {
		c, e := compiler.Compile(query.QueryDocument{Version: q.Version, Operations: q.Operations[:n]})
		if e != nil {
			return q.Operations[n-1].ID
		}
		code, e := r.code(c.Text)
		if e != nil {
			return q.Operations[n-1].ID
		}
		it := code.RunWithContext(ctx, input)
		for i := 0; i < 10000; i++ {
			v, ok := it.Next()
			if !ok {
				break
			}
			if _, ok := v.(error); ok {
				if ctx.Err() == nil {
					return q.Operations[n-1].ID
				}
				return ""
			}
		}
		if ctx.Err() != nil {
			return ""
		}
	}
	return ""
}
func (out ExecutionResult) Summary() string {
	label := fmt.Sprintf("%s · %d result(s) · %s", out.Status, out.Count, out.Duration.Round(time.Millisecond))
	if out.Error != nil {
		label += " · " + out.Error.Error()
	}
	if out.OperationID != "" {
		label += " [" + out.OperationID + "]"
	}
	return label
}
