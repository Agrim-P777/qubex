package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/query"
	"io"
	"math/big"
	"strconv"
)

const DefaultInputLimit int64 = 500 << 20

type Document struct {
	Value  any
	Source string
	Bytes  int64
	Shape  *query.Shape
}
type reader struct {
	ctx      context.Context
	r        io.Reader
	n, max   int64
	progress func(int64)
}

func (r *reader) Read(b []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	if r.n > r.max {
		return 0, fmt.Errorf("input exceeds %d bytes; increase --input-limit", r.max)
	}
	if int64(len(b)) > r.max-r.n+1 {
		b = b[:r.max-r.n+1]
	}
	n, e := r.r.Read(b)
	r.n += int64(n)
	if r.progress != nil {
		r.progress(r.n)
	}
	if r.n > r.max {
		return n, fmt.Errorf("input exceeds %d bytes; increase --input-limit", r.max)
	}
	return n, e
}
func Read(ctx context.Context, r io.Reader, source string, limit int64, progress func(int64)) (*Document, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("input limit must be positive")
	}
	cr := &reader{ctx: ctx, r: r, max: limit, progress: progress}
	d := json.NewDecoder(cr)
	d.UseNumber()
	var v any
	if e := d.Decode(&v); e != nil {
		return nil, fmt.Errorf("parse %s at byte %d: %w", source, d.InputOffset(), e)
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		if e == nil {
			return nil, errors.New("multiple JSON documents are not supported yet; provide one document")
		}
		return nil, fmt.Errorf("trailing input: %w", e)
	}
	v, e := normalize(ctx, v)
	if e != nil {
		return nil, e
	}
	return &Document{v, source, cr.n, query.Infer(v)}, nil
}
func normalize(ctx context.Context, v any) (any, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	switch x := v.(type) {
	case json.Number:
		if i, e := strconv.Atoi(string(x)); e == nil {
			return i, nil
		}
		if b, ok := new(big.Int).SetString(string(x), 10); ok {
			return b, nil
		}
		f, e := strconv.ParseFloat(string(x), 64)
		if e != nil {
			return nil, fmt.Errorf("unsupported JSON number %s: %w", x, e)
		}
		return f, nil
	case []any:
		for i, v := range x {
			n, e := normalize(ctx, v)
			if e != nil {
				return nil, e
			}
			x[i] = n
		}
	case map[string]any:
		for k, v := range x {
			n, e := normalize(ctx, v)
			if e != nil {
				return nil, e
			}
			x[k] = n
		}
	}
	return v, nil
}

// Size stops once a compact JSON value exceeds the display budget. It does not
// allocate a second serialization of a potentially 500 MiB document.
func Size(ctx context.Context, v any, budget int64) (int64, error) {
	left := budget
	var walk func(any) error
	walk = func(v any) error {
		if left < 0 {
			return nil
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		switch x := v.(type) {
		case []any:
			left -= 2
			for i, v := range x {
				if i > 0 {
					left--
				}
				if e := walk(v); e != nil {
					return e
				}
				if left < 0 {
					break
				}
			}
		case map[string]any:
			left -= 2
			i := 0
			for k, v := range x {
				left--
				if i > 0 {
					left--
				}
				i++
				if e := walk(k); e != nil {
					return e
				}
				if e := walk(v); e != nil {
					return e
				}
				if left < 0 {
					break
				}
			}
		case string:
			if int64(len(x)) > left {
				left = -1
				return nil
			}
			b, _ := json.Marshal(x)
			left -= int64(len(b))
		default:
			b, e := json.Marshal(v)
			if e != nil {
				return e
			}
			left -= int64(len(b))
		}
		return nil
	}
	e := walk(v)
	return budget - left, e
}
