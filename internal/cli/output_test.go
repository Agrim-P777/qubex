package cli

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestResultPrinterColorsJSONTokens(t *testing.T) {
	var plain, colored bytes.Buffer
	input := map[string]any{"text": "a\\\"b", "number": -2.5, "null": nil, "true": true}
	if e := newResultPrinter(&plain).Print(input); e != nil {
		t.Fatal(e)
	}
	if e := colorizeJSON(&colored, plain.Bytes()); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(colored.String(), colorKey+`"text"`+colorReset) || !strings.Contains(colored.String(), colorString+`"a\\\"b"`+colorReset) || !strings.Contains(colored.String(), colorNull+"null"+colorReset) {
		t.Fatalf("jq-style colors missing: %q", colored.String())
	}
	stripANSI := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	if got := stripANSI.ReplaceAllString(colored.String(), ""); got != plain.String() {
		t.Fatalf("color output changed JSON: %q != %q", got, plain.String())
	}
}
