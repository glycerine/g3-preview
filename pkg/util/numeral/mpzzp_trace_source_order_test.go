//go:build g3trace

package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type modularTraceRefusal struct {
	output        bytes.Buffer
	calls, failAt int
}

func (w *modularTraceRefusal) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= w.failAt {
		return 0, errors.New("trace refused")
	}
	return w.output.Write(p)
}
func (w *modularTraceRefusal) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestIntegerModularTraceSourceInsertionPrefixes(t *testing.T) {
	z, _ := integerManager(t)
	m, err := NewIntegerModularManager(z)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	if err := m.SetZPUint64(7); err != nil {
		t.Fatal(err)
	}
	a := NewInt(3)
	if err := m.Inv(&a); err != nil {
		t.Fatal(err)
	}
	sites := []struct {
		tag, header string
		parts       []string
		call        func() error
	}{
		{"mpzzp", "-------- [mpzzp] setup_p src/util/mpzzp.h:55 ---------\n", []string{"lower: ", "-3", ", upper: ", "3", "\n"}, m.traceSetupP},
		{"mpzp_inv_bug", "-------- [mpzp_inv_bug] inv src/util/mpzzp.h:175 ---------\n", []string{"a: ", "-2", ", p: ", "7", "\n"}, func() error { return m.traceInvStart(&a) }},
		{"mpzp_inv_bug", "-------- [mpzp_inv_bug] inv src/util/mpzzp.h:177 ---------\n", []string{"after normalization a: ", "-2", "\n"}, func() error { return m.traceInvNormalized(&a) }},
		{"mpzp_inv_bug", "-------- [mpzp_inv_bug] inv src/util/mpzzp.h:179 ---------\n", []string{"tmp1: ", "3", "\ntmp2: ", "1", "\ntmp3: ", "1", "\n"}, m.traceInvGCD},
	}
	for _, site := range sites {
		for refused := range site.parts {
			w := &modularTraceRefusal{failAt: 10 + refused}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable(site.tag)
			if err := m.BindTrace(tags); err != nil {
				t.Fatal(err)
			}
			if err := site.call(); err != nil {
				t.Fatal(site.tag, refused, err)
			}
			want := site.header
			for _, part := range site.parts[:refused] {
				want += part
			}
			if got := w.output.String(); got != want {
				t.Fatalf("%s refusal %d: got %q, want %q", site.tag, refused, got, want)
			}
		}
	}
}

func TestIntegerModularTraceNumberFailureAfterSourcePrefix(t *testing.T) {
	z, _ := integerManager(t)
	m, err := NewIntegerModularManager(z)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpzp_inv_bug")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := m.traceInvStart(nil); err == nil {
		t.Fatal("missing numeral conversion error")
	}
	const want = "-------- [mpzp_inv_bug] inv src/util/mpzzp.h:175 ---------\na: "
	if got := output.String(); got != want {
		t.Fatalf("conversion prefix: got %q, want %q", got, want)
	}
	// A failed ostream insertion sets failbit but does not skip evaluating the
	// following to_string operand in the source << expression.
	w := &modularTraceRefusal{failAt: 10}
	refusing := diagnostic.NewTraceTagsWithWriter(w)
	refusing.Enable("mpzp_inv_bug")
	if err := m.BindTrace(refusing); err != nil {
		t.Fatal(err)
	}
	if err := m.traceInvStart(nil); err == nil {
		t.Fatal("write refusal skipped source numeral conversion")
	}
	const header = "-------- [mpzp_inv_bug] inv src/util/mpzzp.h:175 ---------\n"
	if got := w.output.String(); got != header {
		t.Fatalf("refused prefix: got %q, want %q", got, header)
	}
}
