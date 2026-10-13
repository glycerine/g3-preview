//go:build g3trace

package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type fixedDivisionTraceRefusal struct {
	output        bytes.Buffer
	calls, failAt int
}

func (w *fixedDivisionTraceRefusal) Write(p []byte) (int, error) {
	w.calls++
	if w.calls >= w.failAt {
		return 0, errors.New("trace refused")
	}
	return w.output.Write(p)
}
func (w *fixedDivisionTraceRefusal) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestFixedPointDivisionTraceSourceInsertionPrefixes(t *testing.T) {
	m, e := NewFixedPointManager(1, 1)
	if e != nil {
		t.Fatal(e)
	}
	n, e := m.New()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if err := n.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	if e := m.SetRatio(n, -3, 2); e != nil {
		t.Fatal(e)
	}
	sites := []struct {
		parts []string
		call  func()
	}{
		{[]string{"[mpfx] (", "-", "3", ") / (", "2", ") ", ">=", " "}, func() { m.traceFixedIntegerDivision(-3, 2, n, false) }},
		{[]string{"[mpfx] (1) / (", "-", "3", "/2", ") ", ">=", " "}, func() { m.traceFixedInverse(n, false) }},
	}
	for _, site := range sites {
		for refused := range site.parts {
			w := &fixedDivisionTraceRefusal{failAt: 1 + refused}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("mpfx_trace")
			if e := m.BindTrace(tags); e != nil {
				t.Fatal(e)
			}
			site.call()
			want := ""
			for _, part := range site.parts[:refused] {
				want += part
			}
			if got := w.output.String(); got != want || w.calls != w.failAt {
				t.Fatalf("refusal %d: got %q, want %q, calls %d", refused, got, want, w.calls)
			}
		}
	}
}

func TestFixedPointDivisionTraceRefusalKeepsArithmetic(t *testing.T) {
	m, e := NewFixedPointManager(1, 1)
	if e != nil {
		t.Fatal(e)
	}
	n, e := m.New()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if err := n.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	ratio := &fixedDivisionTraceRefusal{failAt: 3}
	tags := diagnostic.NewTraceTagsWithWriter(ratio)
	tags.Enable("mpfx_trace")
	if e := m.BindTrace(tags); e != nil {
		t.Fatal(e)
	}
	if e := m.SetRatio(n, -3, 2); e != nil {
		t.Fatal("ratio after trace failbit", e)
	}
	if got, e := m.String(n); e != nil || got != "-3/2" {
		t.Fatal("ratio result", got, e)
	}
	if got := ratio.output.String(); got != "[mpfx] (-" {
		t.Fatal("ratio refused prefix", got)
	}
	inverse := &fixedDivisionTraceRefusal{failAt: 3}
	tags = diagnostic.NewTraceTagsWithWriter(inverse)
	tags.Enable("mpfx_trace")
	if e := m.BindTrace(tags); e != nil {
		t.Fatal(e)
	}
	if e := m.Inv(n); e != nil {
		t.Fatal("inverse after trace failbit", e)
	}
	if got, e := m.String(n); e != nil || got != "-2863311531/2^32" {
		t.Fatal("inverse result", got, e)
	}
	if got := inverse.output.String(); got != "[mpfx] (1) / (-" {
		t.Fatal("inverse refused prefix", got)
	}
}
