package cli

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/compiler"
	"github.com/Agrim-P777/qubex/internal/data"
	"github.com/Agrim-P777/qubex/internal/execute"
	"github.com/Agrim-P777/qubex/internal/history"
	"github.com/Agrim-P777/qubex/internal/query"
	"github.com/Agrim-P777/qubex/internal/tree"
	"github.com/Agrim-P777/qubex/internal/tui"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
)

const Usage = `Qubex — interactive structured-data query builder

Usage:
  qbx [file]                          Build a query; defaults to piped stdin
  qbx history [--json]                List saved queries
  qbx history search TEXT [--json]    Fuzzy-search saved queries
  qbx history show ID                 Inspect a saved definition
  qbx history open ID [file]          Restore into the TUI on new input
  qbx history run ID|NAME [file]      Execute fully, writing pretty JSON to stdout
  qbx history copy ID                 Request terminal clipboard copy
  qbx history delete ID               Delete a saved definition
  qbx history export ID --format jq|json|text

Options:
  --input-limit BYTES       Maximum input (default 524288000; accepts MiB/GiB)
  --preview-timeout TIME    Preview timeout (default 2s)
  --preview-results N       Displayed results (default 10000)
  --preview-bytes BYTES     Display budget (default 8388608)
  --timeout TIME            history run timeout (default 60s)
  --max-results N           history run result cap (default unlimited)
  --max-bytes BYTES         history run output cap (default unlimited)
  --help, -h               Show this help
  --version               Show build version

TUI: ? actions · f filter · m transform · o sort · v source/stage
     s save · c copy · u undo · Ctrl+P pipeline · Ctrl+H history · q quit

One JSON document per invocation. No jq executable is required.
`

type config struct {
	Keys           map[string]string `json:"keys"`
	InputLimit     int64             `json:"input_limit"`
	PreviewTimeout string            `json:"preview_timeout"`
	PreviewResults int               `json:"preview_results"`
	PreviewBytes   int64             `json:"preview_bytes"`
}
type options struct {
	args          []string
	json          bool
	format        string
	inputLimit    int64
	limits        execute.Limits
	runLimits     execute.Limits
	keys          map[string]string
	help, version bool
}

func parse(args []string) (options, error) {
	o := options{inputLimit: data.DefaultInputLimit, limits: execute.PreviewLimits(), runLimits: execute.RunLimits(), format: "jq"}
	p := filepath.Join(history.DefaultDir(), "config.json")
	if b, e := os.ReadFile(p); e == nil {
		var c config
		if e = json.Unmarshal(b, &c); e != nil {
			return o, fmt.Errorf("config %s: %w", p, e)
		}
		o.keys = c.Keys
		if c.InputLimit > 0 {
			o.inputLimit = c.InputLimit
		}
		if c.PreviewResults > 0 {
			o.limits.Results = c.PreviewResults
		}
		if c.PreviewBytes > 0 {
			o.limits.Bytes = c.PreviewBytes
		}
		if c.PreviewTimeout != "" {
			d, e := time.ParseDuration(c.PreviewTimeout)
			if e != nil || d <= 0 {
				return o, fmt.Errorf("invalid preview_timeout")
			}
			o.limits.Timeout = d
		}
	} else if !os.IsNotExist(e) {
		return o, e
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--":
			o.args = append(o.args, args[i+1:]...)
			return o, nil
		case "--help", "-h":
			o.help = true
		case "--version":
			o.version = true
		case "--json":
			o.json = true
		case "--format", "--input-limit", "--preview-timeout", "--preview-results", "--preview-bytes", "--timeout", "--max-results", "--max-bytes":
			if i+1 == len(args) {
				return o, fmt.Errorf("%s requires a value", a)
			}
			i++
			v := args[i]
			switch a {
			case "--format":
				if v != "jq" && v != "json" && v != "text" {
					return o, fmt.Errorf("format must be jq, json or text")
				}
				o.format = v
			case "--input-limit":
				n, e := parseBytes(v)
				if e != nil {
					return o, e
				}
				o.inputLimit = n
			case "--preview-bytes":
				n, e := parseBytes(v)
				if e != nil {
					return o, e
				}
				o.limits.Bytes = n
			case "--preview-results":
				n, e := strconv.Atoi(v)
				if e != nil || n <= 0 {
					return o, fmt.Errorf("preview-results must be positive")
				}
				o.limits.Results = n
			case "--preview-timeout":
				d, e := time.ParseDuration(v)
				if e != nil || d <= 0 {
					return o, fmt.Errorf("preview-timeout must be positive")
				}
				o.limits.Timeout = d
			case "--timeout":
				d, e := time.ParseDuration(v)
				if e != nil || d <= 0 {
					return o, fmt.Errorf("timeout must be positive")
				}
				o.runLimits.Timeout = d
			case "--max-results":
				n, e := strconv.Atoi(v)
				if e != nil || n <= 0 {
					return o, fmt.Errorf("max-results must be positive")
				}
				o.runLimits.Results = n
			case "--max-bytes":
				n, e := parseBytes(v)
				if e != nil {
					return o, e
				}
				o.runLimits.Bytes = n
			}
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				return o, fmt.Errorf("unknown option %s", a)
			}
			o.args = append(o.args, a)
		}
	}
	return o, nil
}
func parseBytes(s string) (int64, error) {
	mult := int64(1)
	for suffix, n := range map[string]int64{"MiB": 1 << 20, "GiB": 1 << 30, "KiB": 1 << 10} {
		if strings.HasSuffix(s, suffix) {
			mult = n
			s = strings.TrimSuffix(s, suffix)
			break
		}
	}
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n <= 0 || n > int64(^uint64(0)>>1)/mult {
		return 0, fmt.Errorf("invalid positive byte limit")
	}
	return n * mult, nil
}
func Run(args []string, stdin *os.File, stdout, stderr io.Writer, version string) int {
	o, e := parse(args)
	fail := func(code int, e error) int { fmt.Fprintln(stderr, "qbx:", e); return code }
	if e != nil {
		return fail(2, e)
	}
	if o.help {
		fmt.Fprint(stdout, Usage)
		return 0
	}
	if o.version {
		fmt.Fprintln(stdout, "Qubex", version)
		return 0
	}
	store := history.Store{Dir: history.DefaultDir()}
	q := query.New()
	run := false
	var usedID int
	if len(o.args) > 0 && o.args[0] == "history" {
		a := o.args[1:]
		if len(a) == 0 || (len(a) > 0 && a[0] == "search") {
			pattern := ""
			if len(a) > 0 {
				if len(a) != 2 {
					return fail(2, fmt.Errorf("history search requires TEXT"))
				}
				pattern = a[1]
			}
			records, e := store.List()
			if e != nil {
				return fail(4, e)
			}
			found := []history.Record{}
			for _, r := range records {
				if tree.Match(r.Name+" "+r.JQ, pattern, false, false) {
					found = append(found, r)
				}
			}
			if o.json {
				if e := json.NewEncoder(stdout).Encode(found); e != nil {
					return fail(4, e)
				}
			} else {
				for _, r := range found {
					if _, e := fmt.Fprintf(stdout, "%d\t%s\t%s\n", r.ID, safe(r.Name), sanitize(r.JQ)); e != nil {
						return fail(4, e)
					}
				}
			}
			return 0
		}
		if len(a) < 2 {
			return fail(2, fmt.Errorf("history command requires an ID or name"))
		}
		id, idError := strconv.Atoi(a[1])
		var r history.Record
		if a[0] == "run" && idError != nil {
			r, e = store.GetByName(a[1])
			id = r.ID
		} else {
			if idError != nil || id < 1 {
				return fail(2, fmt.Errorf("invalid history ID"))
			}
			r, e = store.Get(id)
		}
		if e != nil {
			return fail(4, e)
		}
		if a[0] != "open" && a[0] != "run" && len(a) != 2 {
			return fail(2, fmt.Errorf("unexpected history arguments"))
		}
		switch a[0] {
		case "show":
			b, _ := json.MarshalIndent(r, "", "  ")
			if _, e := fmt.Fprintln(stdout, string(b)); e != nil {
				return fail(4, e)
			}
			return 0
		case "export":
			var text string
			switch o.format {
			case "jq":
				text = r.JQ
			case "json":
				b, e := json.MarshalIndent(r.Query, "", "  ")
				if e != nil {
					return fail(4, e)
				}
				text = string(b)
			case "text":
				text = r.Name + "\n"
				for i, op := range r.Query.Operations {
					text += fmt.Sprintf("%d. %s %s\n", i+1, op.Kind, op.Path.String())
				}
			}
			if _, e := fmt.Fprintln(stdout, text); e != nil {
				return fail(4, e)
			}
			return 0
		case "delete":
			if e := store.Delete(id); e != nil {
				return fail(4, e)
			}
			return 0
		case "copy":
			f, e := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
			if e != nil {
				return fail(4, fmt.Errorf("clipboard needs a terminal; use history export %d --format jq", id))
			}
			defer f.Close()
			_, e = fmt.Fprintf(f, "\x1b]52;c;%s\x07", base64.StdEncoding.EncodeToString([]byte(r.JQ)))
			if e != nil {
				return fail(4, e)
			}
			fmt.Fprintln(stderr, "Clipboard write requested; terminal support is required.")
			return 0
		case "open", "run":
			if len(a) > 3 {
				return fail(2, fmt.Errorf("provide at most one input file"))
			}
			q = r.Query
			usedID = id
			run = a[0] == "run"
			o.args = a[2:]
		default:
			return fail(2, fmt.Errorf("unknown history command %q", a[0]))
		}
	}
	if len(o.args) > 1 {
		return fail(2, fmt.Errorf("provide one JSON file or piped stdin"))
	}
	var reader io.Reader = stdin
	source := "stdin"
	if len(o.args) == 1 && o.args[0] != "-" {
		f, e := os.Open(o.args[0])
		if e != nil {
			return fail(2, e)
		}
		defer f.Close()
		reader = f
		source = filepath.Base(o.args[0])
	} else {
		stat, e := stdin.Stat()
		if e != nil {
			return fail(2, e)
		}
		if stat.Mode()&os.ModeCharDevice != 0 {
			return fail(2, fmt.Errorf("provide a JSON file or pipe input; see qbx --help"))
		}
	}
	if _, e := compiler.Compile(q); e != nil {
		return fail(3, e)
	}
	if run {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		doc, e := data.Read(ctx, reader, source, o.inputLimit, nil)
		if e != nil {
			if ctx.Err() != nil {
				return 130
			}
			return fail(2, e)
		}
		printer := newResultPrinter(stdout)
		result := execute.New().Run(ctx, q, doc.Value, 1, o.runLimits, printer.Print)
		if result.Error != nil {
			if errors.Is(result.Error, context.Canceled) {
				return 130
			}
			code := 3
			if result.Status == "io_error" {
				code = 4
			}
			return fail(code, result.Error)
		}
		if usedID > 0 {
			if e := store.Touch(usedID); e != nil {
				return fail(4, e)
			}
		}
		return 0
	}
	in, out, e := tea.OpenTTY()
	if e != nil {
		return fail(2, fmt.Errorf("interactive mode needs a controlling terminal; use history run for scripts: %w", e))
	}
	defer in.Close()
	defer out.Close()
	m := tui.New(tui.Options{Reader: reader, Source: source, InputLimit: o.inputLimit, Limits: o.limits, Store: store, Query: q, Keys: o.keys})
	defer m.Close()
	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	program := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out), tea.WithContext(signalContext), tea.WithoutSignalHandler())
	_, e = program.Run()
	if signalContext.Err() != nil || errors.Is(e, tea.ErrInterrupted) {
		return 130
	}
	if e != nil {
		return fail(4, e)
	}
	if m.Interrupted {
		return 130
	}
	if m.InputError {
		return 2
	}
	if usedID > 0 {
		if e := store.Touch(usedID); e != nil {
			return fail(4, e)
		}
	}
	return 0
}
func safe(s string) string { b, _ := json.Marshal(s); return string(b) }

// sanitize strips control characters (incl. ANSI escape introducers and
// newlines) so a saved record's jq cannot inject terminal escape sequences or
// break the one-line-per-record listing when printed to a TTY.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '�'
		}
		return r
	}, s)
}
