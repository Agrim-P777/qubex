package cli

import (
	"bytes"
	"github.com/Agrim-P777/qubex/internal/history"
	"github.com/Agrim-P777/qubex/internal/query"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIHistoryAndRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := history.Store{Dir: history.DefaultDir()}
	r, e := s.Save("names", query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "i", Kind: "iterate"}, {ID: "s", Kind: "select", Path: query.Path{query.Field("name")}}}}, "original.json")
	if e != nil || r.ID != 1 {
		t.Fatal(r, e)
	}
	p := filepath.Join(t.TempDir(), "input.json")
	os.WriteFile(p, []byte(`[{"name":"api"},{"name":"worker"}]`), 0600)
	var out, err bytes.Buffer
	code := Run([]string{"history", "run", "1", p}, os.Stdin, &out, &err, "test")
	if code != 0 || out.String() != "\"api\"\n\"worker\"\n" {
		t.Fatal(code, out.String(), err.String())
	}
	out.Reset()
	if code := Run([]string{"history", "export", "1", "--format", "json"}, os.Stdin, &out, &err, "test"); code != 0 || !strings.Contains(out.String(), `"version": 1`) {
		t.Fatal(code, out.String())
	}
	out.Reset()
	if code := Run([]string{"history", "--json"}, os.Stdin, &out, &err, "test"); code != 0 || !strings.Contains(out.String(), "names") {
		t.Fatal(code, out.String())
	}
}
func TestCLIExitCodes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, err bytes.Buffer
	for _, args := range [][]string{{"--unknown"}, {"history", "show", "no"}, {"--input-limit", "0"}} {
		if c := Run(args, os.Stdin, &out, &err, "test"); c != 2 {
			t.Fatal(args, c)
		}
	}
	if c := Run([]string{"history", "show", "999"}, os.Stdin, &out, &err, "test"); c != 4 {
		t.Fatal(c)
	}
	if c := Run([]string{"history", "run", "missing"}, os.Stdin, &out, &err, "test"); c != 4 {
		t.Fatal(c)
	}
	if c := Run([]string{"--help"}, os.Stdin, &out, &err, "test"); c != 0 {
		t.Fatal(c)
	}
}

func TestParseRunLimits(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	o, e := parse([]string{"history", "run", "1"})
	if e != nil || o.runLimits.Timeout <= 0 {
		t.Fatalf("headless run must have a positive default timeout: %+v %v", o.runLimits, e)
	}
	o, e = parse([]string{"--timeout", "5s", "--max-results", "3", "--max-bytes", "1MiB", "history", "run", "1"})
	if e != nil || o.runLimits.Timeout != 5*time.Second || o.runLimits.Results != 3 || o.runLimits.Bytes != 1<<20 {
		t.Fatalf("run-limit flags not applied: %+v %v", o.runLimits, e)
	}
	for _, bad := range [][]string{{"--timeout", "0"}, {"--timeout", "x"}, {"--max-results", "0"}, {"--max-bytes", "-1"}} {
		if _, e := parse(bad); e == nil {
			t.Fatalf("expected error for %v", bad)
		}
	}
}

func TestSanitizeStripsControlBytes(t *testing.T) {
	got := sanitize("a\x1b[31mred\x07\nb")
	if strings.ContainsAny(got, "\x1b\x07\n") {
		t.Fatalf("control bytes not stripped: %q", got)
	}
}

func TestHistoryRunByNamePrintsPrettyJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := history.Store{Dir: history.DefaultDir()}
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{
		{ID: "i", Kind: "iterate", Path: query.Path{query.Field("services")}},
		{ID: "f", Kind: "filter", Predicate: &query.Predicate{Kind: "eq", Path: query.Path{query.Field("language")}, Value: "python"}},
	}}
	if _, e := s.Save("filter python", q, "services.json"); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(t.TempDir(), "services.json")
	if e := os.WriteFile(p, []byte(`{"services":[{"name":"api","language":"python"},{"name":"web","language":"go"}]}`), 0600); e != nil {
		t.Fatal(e)
	}
	var out, stderr bytes.Buffer
	if code := Run([]string{"history", "run", "filter python", p}, os.Stdin, &out, &stderr, "test"); code != 0 {
		t.Fatalf("run by name: exit %d: %s", code, stderr.String())
	}
	want := "{\n  \"language\": \"python\",\n  \"name\": \"api\"\n}\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
	if _, e := s.Save("filter python", q, "services.json"); e != nil {
		t.Fatal(e)
	}
	out.Reset()
	stderr.Reset()
	if code := Run([]string{"history", "run", "filter python", p}, os.Stdin, &out, &stderr, "test"); code != 4 || !strings.Contains(stderr.String(), "matches IDs 1, 2; use an ID") || out.Len() != 0 {
		t.Fatalf("ambiguous name: exit %d, stdout %q, stderr %q", code, out.String(), stderr.String())
	}
}
