package ast

import (
	"fmt"
	"strings"
	"testing"
)

func TestLegacySMTPrinterStringLiteralByteEncoding(t *testing.T) {
	m := astTestManager(t)
	p, err := NewLegacySMTPrinter(m)
	astTestMust(t, err)
	for i := 0; i < 256; i++ {
		value, err := m.MkString(string([]byte{byte(i)}))
		astTestMust(t, err)
		astTestPin(t, m, value)
		var encoded string
		switch {
		case i == 10:
			encoded = `\n`
		case i == 11:
			encoded = `\v`
		case i == 12:
			encoded = `\f`
		case i == 13:
			encoded = `\r`
		case i < 32:
			encoded = fmt.Sprintf(`\x%02X`, i)
		case i == 34:
			encoded = `""`
		case i == 92:
			encoded = `\\`
		case i >= 128:
			encoded = fmt.Sprintf(`\x%x`, i)
		default:
			encoded = string([]byte{byte(i)})
		}
		var out strings.Builder
		astTestMust(t, p.DisplayExpr(&out, value, nil))
		if out.String() != `"`+encoded+`"` {
			t.Fatalf("legacy byte %02x: %q", i, out.String())
		}
	}
	for _, raw := range []string{"", `a"b\c`, string([]byte{0, 10, 34, 92, 127, 128, 255})} {
		value, err := m.MkString(raw)
		astTestMust(t, err)
		astTestPin(t, m, value)
		var out strings.Builder
		astTestMust(t, p.DisplayExpr(&out, value, nil))
		if !strings.HasPrefix(out.String(), `"`) || !strings.HasSuffix(out.String(), `"`) || strings.Contains(out.String(), "\x00") {
			t.Fatal("unquoted or raw NUL literal", out.String())
		}
	}
}
