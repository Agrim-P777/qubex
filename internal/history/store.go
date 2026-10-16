// Package history persists definitions only, with process-safe atomic updates.
package history

import (
	"encoding/json"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/compiler"
	"github.com/Agrim-P777/qubex/internal/query"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Record struct {
	ID         int                 `json:"id"`
	Name       string              `json:"name"`
	Query      query.QueryDocument `json:"operations"`
	JQ         string              `json:"jq"`
	Source     string              `json:"source"`
	CreatedAt  time.Time           `json:"created_at"`
	LastUsedAt *time.Time          `json:"last_used_at,omitempty"`
}
type file struct {
	Version int      `json:"version"`
	NextID  int      `json:"next_id"`
	Records []Record `json:"records"`
}
type Store struct{ Dir string }

func DefaultDir() string {
	if p := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(p) {
		return filepath.Join(p, "qubex")
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config", "qubex")
}
func (s Store) transact(write bool, fn func(*file) error) error {
	if e := os.MkdirAll(s.Dir, 0700); e != nil {
		return e
	}
	lock, e := os.OpenFile(filepath.Join(s.Dir, "history.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); e != nil {
		return e
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	f := file{Version: 1, NextID: 1, Records: []Record{}}
	p := filepath.Join(s.Dir, "history.json")
	b, e := os.ReadFile(p)
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	if e == nil {
		d := json.NewDecoder(strings.NewReader(string(b)))
		d.UseNumber()
		if e = d.Decode(&f); e != nil {
			return fmt.Errorf("history is corrupt; back up and repair %s: %w", p, e)
		}
		var extra any
		if e = d.Decode(&extra); e != io.EOF {
			return fmt.Errorf("history has trailing data; preserve and repair %s", p)
		}
		if f.Version != 1 {
			return fmt.Errorf("unsupported history version %d; preserve %s and use a compatible Qubex version", f.Version, p)
		}
		seen := map[int]bool{}
		for _, r := range f.Records {
			if r.ID < 1 || seen[r.ID] {
				return fmt.Errorf("invalid history IDs; preserve and repair %s", p)
			}
			seen[r.ID] = true
			if r.ID >= f.NextID {
				f.NextID = r.ID + 1
			}
		}
	}
	if e = fn(&f); e != nil {
		return e
	}
	if !write {
		return nil
	}
	b, e = json.MarshalIndent(f, "", "  ")
	if e != nil {
		return e
	}
	tmp, e := os.CreateTemp(s.Dir, ".history-*")
	if e != nil {
		return e
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, e = tmp.Write(b); e != nil {
		tmp.Close()
		return e
	}
	if e = tmp.Sync(); e != nil {
		tmp.Close()
		return e
	}
	if e = tmp.Close(); e != nil {
		return e
	}
	if e = os.Rename(name, p); e != nil {
		return e
	}
	dir, e := os.Open(s.Dir)
	if e == nil {
		e = dir.Sync()
		dir.Close()
	}
	return e
}
func (s Store) List() ([]Record, error) {
	var r []Record
	e := s.transact(false, func(f *file) error { r = append([]Record{}, f.Records...); return nil })
	return r, e
}
func (s Store) Get(id int) (Record, error) {
	var r Record
	e := s.transact(false, func(f *file) error {
		for _, v := range f.Records {
			if v.ID == id {
				r = v
				return nil
			}
		}
		return fmt.Errorf("history %d not found", id)
	})
	return r, e
}
func (s Store) GetByName(name string) (Record, error) {
	var matches []Record
	e := s.transact(false, func(f *file) error {
		for _, r := range f.Records {
			if r.Name == name {
				matches = append(matches, r)
			}
		}
		return nil
	})
	if e != nil {
		return Record{}, e
	}
	if len(matches) == 0 {
		return Record{}, fmt.Errorf("history name %q not found", name)
	}
	if len(matches) > 1 {
		ids := make([]string, len(matches))
		for i, r := range matches {
			ids[i] = strconv.Itoa(r.ID)
		}
		return Record{}, fmt.Errorf("history name %q matches IDs %s; use an ID", name, strings.Join(ids, ", "))
	}
	return matches[0], nil
}
func (s Store) Save(name string, q query.QueryDocument, source string) (Record, error) {
	var r Record
	c, e := compiler.Compile(q)
	if e != nil {
		return r, e
	}
	e = s.transact(true, func(f *file) error {
		r = Record{ID: f.NextID, Name: name, Query: q.Clone(), JQ: c.Text, Source: source, CreatedAt: time.Now().UTC()}
		f.NextID++
		f.Records = append(f.Records, r)
		return nil
	})
	return r, e
}
func (s Store) Delete(id int) error {
	return s.transact(true, func(f *file) error {
		for i, r := range f.Records {
			if r.ID == id {
				f.Records = append(f.Records[:i], f.Records[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("history %d not found", id)
	})
}
func (s Store) Touch(id int) error {
	return s.transact(true, func(f *file) error {
		for i, r := range f.Records {
			if r.ID == id {
				t := time.Now().UTC()
				f.Records[i].LastUsedAt = &t
				return nil
			}
		}
		return fmt.Errorf("history %d not found", id)
	})
}
