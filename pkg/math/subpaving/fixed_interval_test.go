package subpaving

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type fixedDisplayRefusal struct {
	bytes.Buffer
	remaining int
	err       error
}

func (w *fixedDisplayRefusal) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, w.err
	}
	w.remaining--
	return w.Buffer.Write(p)
}

func (w *fixedDisplayRefusal) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestSubpavingFixedMutableIntervalRetainedCore(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	lo, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	hi, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewFixedInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewFixedInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		must(a.Close())
		must(b.Close())
		must(c.Close())
		must(lo.Close())
		must(hi.Close())
		must(m.Close())
	}()
	set := func(v *FixedInterval, lnum, lden, unum, uden int64, lOpen, uOpen bool) {
		t.Helper()
		must(m.SetRatio(lo, lnum, uint64(lden)))
		must(m.SetRatio(hi, unum, uint64(uden)))
		must(v.SetFinite(lo, hi, lOpen, uOpen))
	}
	show := func(v *FixedInterval) string {
		t.Helper()
		var out bytes.Buffer
		must(v.Display(&out))
		return out.String()
	}
	if got := show(a); got != "(-oo, +oo)" {
		t.Fatalf("initial interval = %q", got)
	}
	for _, tc := range []struct {
		at   int
		want string
	}{{1, ""}, {3, "(-oo,"}, {5, "(-oo, +oo"}} {
		w := &shortOnceSubpavingIntervalWriter{at: tc.at}
		if err := a.Display(w); !errors.Is(err, io.ErrShortWrite) || w.String() != tc.want || w.calls != tc.at {
			t.Fatalf("short insertion %d = (%q, %d, %v)", tc.at, w.String(), w.calls, err)
		}
	}
	contains, err := a.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("unbounded interval excludes zero")
	}
	set(a, -3, 2, 2, 1, true, false) // (-3/2,2]
	writeErr := errors.New("fixed upper endpoint refused")
	refused := &fixedDisplayRefusal{remaining: 3, err: writeErr}
	if err := a.Display(refused); !errors.Is(err, writeErr) || refused.String() != "(-3/2, " {
		t.Fatalf("fixed display refusal = (%q, %v)", refused.String(), err)
	}
	set(b, 1, 2, 3, 2, false, true) // [1/2,3/2)
	contains, err = a.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("crossing interval excludes zero")
	}
	contains, err = b.ContainsZero()
	must(err)
	if contains {
		t.Fatal("positive interval contains zero")
	}
	must(c.Add(a, b))
	if got := show(c); got != "(-1, 7/2)" {
		t.Fatalf("sum = %q", got)
	}
	must(c.Sub(a, b))
	if got := show(c); got != "(-3, 3/2]" {
		t.Fatalf("difference = %q", got)
	}
	must(c.SetFrom(a))
	must(c.Sub(c, b))
	if got := show(c); got != "(-3, 3/2]" {
		t.Fatalf("aliased difference = %q", got)
	}
	must(c.Neg(a))
	if got := show(c); got != "[-2, 3/2)" {
		t.Fatalf("negation = %q", got)
	}
	must(c.Neg(c))
	if got := show(c); got != "(-3/2, 2]" {
		t.Fatalf("aliased negation = %q", got)
	}
	must(a.ResetLower())
	must(c.Add(a, b))
	if got := show(c); got != "(-oo, 7/2)" {
		t.Fatalf("one-sided sum = %q", got)
	}
	must(c.Neg(a))
	if got := show(c); got != "[-2, +oo)" {
		t.Fatalf("one-sided negation = %q", got)
	}
	negative, err := c.LowerIsNeg()
	must(err)
	if !negative {
		t.Fatal("negative lower bound classified nonnegative")
	}
	negative, err = b.LowerIsNeg()
	must(err)
	if negative {
		t.Fatal("positive lower bound classified negative")
	}
	set(a, 0, 1, 0, 1, false, false)
	if got := show(a); got != "[0, 0]" {
		t.Fatalf("zero singleton = %q", got)
	}
	contains, err = a.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("closed zero singleton excludes zero")
	}
	if err := a.SetFinite(lo, hi, true, false); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatalf("open zero singleton error = %v", err)
	}
	must(c.Reset())
	if got := show(c); got != "(-oo, +oo)" {
		t.Fatalf("reset = %q", got)
	}
	must(c.CheckInvariant())
}
