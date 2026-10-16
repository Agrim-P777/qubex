// The benchmark uses a repeating reader, avoiding a second full input buffer.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/data"
	"github.com/Agrim-P777/qubex/internal/tree"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
)

type repeated struct {
	record string
	left   int
	offset int
}

func (r *repeated) Read(p []byte) (int, error) {
	if r.left == 0 {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && r.left > 0 {
		k := copy(p[n:], r.record[r.offset:])
		n += k
		r.offset += k
		if r.offset == len(r.record) {
			r.offset = 0
			r.left--
		}
	}
	return n, nil
}
func main() {
	size := flag.Int("mib", 5, "input size in MiB")
	flag.Parse()
	record := `{"id":123456789,"name":"` + strings.Repeat("record", 160) + `","active":true},`
	count := ((*size << 20) - 2) / len(record)
	input := io.MultiReader(strings.NewReader("["), &repeated{record: record, left: count}, strings.NewReader(`null]`))
	start := time.Now()
	doc, e := data.Read(context.Background(), input, "synthetic", int64(*size+1)<<20, nil)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	tr := tree.New([]any{doc.Value}, false)
	tr.Rows(40)
	load := time.Since(start)
	times := make([]int64, 1000)
	for i := range times {
		start := time.Now()
		tr.Move(1)
		tr.Rows(40)
		times[i] = time.Since(start).Nanoseconds()
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	var ru syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	rss := ru.Maxrss
	if runtime.GOOS == "linux" {
		rss *= 1024
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "go": runtime.Version(), "cpus": runtime.NumCPU(), "input_bytes": doc.Bytes, "records": count + 1, "load_and_first_rows_ms": float64(load) / float64(time.Millisecond), "navigation_p95_ms": float64(times[949]) / 1e6, "peak_rss_mib": float64(rss) / (1 << 20)})
	runtime.KeepAlive(doc)
}
