package history

import (
	"github.com/Agrim-P777/qubex/internal/query"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentSaveAndRoundTrip(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := s.Save("name", query.New(), "stdin"); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	rs, e := s.List()
	if e != nil || len(rs) != 20 {
		t.Fatal(len(rs), e)
	}
	for i, r := range rs {
		if r.ID != i+1 || r.Query.Version != 1 {
			t.Fatal(r)
		}
	}
	if e := s.Touch(1); e != nil {
		t.Fatal(e)
	}
	r, e := s.Get(1)
	if e != nil || r.LastUsedAt == nil {
		t.Fatal(r, e)
	}
	if e = s.Delete(1); e != nil {
		t.Fatal(e)
	}
	r, e = s.Save("next", query.New(), "stdin")
	if e != nil || r.ID != 21 {
		t.Fatal(r, e)
	}
	stat, _ := os.Stat(filepath.Join(s.Dir, "history.json"))
	if stat.Mode().Perm() != 0600 {
		t.Fatal(stat.Mode())
	}
}
func TestPreserveCorruptHistory(t *testing.T) {
	for _, input := range []string{"{broken", `{"version":99}`, `{"version":1,"records":[{"id":1},{"id":1}]}`} {
		s := Store{Dir: t.TempDir()}
		p := filepath.Join(s.Dir, "history.json")
		if e := os.WriteFile(p, []byte(input), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := s.Save("x", query.New(), "stdin"); e == nil {
			t.Fatal("overwrote corrupt history")
		}
		b, _ := os.ReadFile(p)
		if string(b) != input {
			t.Fatal("modified file")
		}
	}
}
