package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/charmbracelet/x/term"
)

const (
	colorReset  = "\x1b[0m"
	colorSyntax = "\x1b[1;39m"
	colorKey    = "\x1b[1;34m"
	colorString = "\x1b[0;32m"
	colorNumber = "\x1b[0;39m"
	colorNull   = "\x1b[0;90m"
)

type resultPrinter struct {
	encoder *json.Encoder
	buffer  bytes.Buffer
	output  *bufio.Writer
	color   bool
}

func newResultPrinter(out io.Writer) *resultPrinter {
	p := &resultPrinter{}
	if f, ok := out.(*os.File); ok && term.IsTerminal(f.Fd()) && os.Getenv("TERM") != "dumb" {
		_, noColor := os.LookupEnv("NO_COLOR")
		p.color = !noColor
	}
	if p.color {
		p.encoder = json.NewEncoder(&p.buffer)
		p.output = bufio.NewWriter(out)
	} else {
		p.encoder = json.NewEncoder(out)
	}
	p.encoder.SetEscapeHTML(false)
	p.encoder.SetIndent("", "  ")
	return p
}

func (p *resultPrinter) Print(v any) error {
	if !p.color {
		return p.encoder.Encode(v)
	}
	p.buffer.Reset()
	if e := p.encoder.Encode(v); e != nil {
		return e
	}
	if e := colorizeJSON(p.output, p.buffer.Bytes()); e != nil {
		return e
	}
	return p.output.Flush()
}

// colorizeJSON receives only JSON produced by encoding/json. Its palette follows
// jq's defaults: blue object keys, green strings, gray null, and white syntax.
func colorizeJSON(out io.Writer, b []byte) error {
	write := func(s []byte) error {
		n, e := out.Write(s)
		if e == nil && n != len(s) {
			return io.ErrShortWrite
		}
		return e
	}
	colored := func(code string, token []byte) error {
		if e := write([]byte(code)); e != nil {
			return e
		}
		if e := write(token); e != nil {
			return e
		}
		return write([]byte(colorReset))
	}
	for i := 0; i < len(b); {
		switch b[i] {
		case '"':
			end := i + 1
			for end < len(b) {
				if b[end] == '\\' {
					end += 2
				} else if b[end] == '"' {
					end++
					break
				} else {
					end++
				}
			}
			code := colorString
			for next := end; next < len(b); next++ {
				if b[next] == ' ' || b[next] == '\t' {
					continue
				}
				if b[next] == ':' {
					code = colorKey
				}
				break
			}
			if e := colored(code, b[i:end]); e != nil {
				return e
			}
			i = end
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			end := i + 1
			for end < len(b) && (b[end] == '.' || b[end] == '-' || b[end] == '+' || b[end] == 'e' || b[end] == 'E' || b[end] >= '0' && b[end] <= '9') {
				end++
			}
			if e := colored(colorNumber, b[i:end]); e != nil {
				return e
			}
			i = end
		case 't', 'f', 'n':
			end := i + 1
			for end < len(b) && b[end] >= 'a' && b[end] <= 'z' {
				end++
			}
			code := colorNumber
			if b[i] == 'n' {
				code = colorNull
			}
			if e := colored(code, b[i:end]); e != nil {
				return e
			}
			i = end
		case '{', '}', '[', ']', ':', ',':
			end := i + 1
			if b[i] == ':' && end < len(b) && b[end] == ' ' {
				end++
			}
			if e := colored(colorSyntax, b[i:end]); e != nil {
				return e
			}
			i = end
		default:
			end := i + 1
			for end < len(b) && (b[end] == ' ' || b[end] == '\t' || b[end] == '\n' || b[end] == '\r') {
				end++
			}
			if e := write(b[i:end]); e != nil {
				return e
			}
			i = end
		}
	}
	return nil
}
