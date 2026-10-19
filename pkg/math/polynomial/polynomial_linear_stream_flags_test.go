package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	utext "github.com/glycerine/g3/pkg/util/text"
	"strings"
	"testing"
)

// Cold test bridge supplies both io.Writer and the actual receiving numeric
// insertion operations. It contains no numeral conversion or formatting logic.
type linearNumericDisplayWriter struct{ *utext.NumericTextStream }

func (w linearNumericDisplayWriter) Write(p []byte) (int, error) {
	if e := w.AppendCString(string(p)); e != nil {
		return 0, e
	}
	return len(p), nil
}

func TestLinearRationalDisplayReceivingNumericFlags(t *testing.T) {
	for _, f := range []struct {
		name, value, want string
		width             int64
	}{
		{"small", "15/31", "_____0xf/0x1f 0x1f\n", 8},
		{"big-positive", "4294967296/15", "4294967296/0xf 0x1f\n", 5},
		{"big-negative", "-4294967296/15", "____-4294967296/0xf 0x1f\n", 5},
	} {
		t.Run(f.name, func(t *testing.T) {
			q, budget := rationalManager(t, false)
			s, e := NewRationalLinearSolver(q)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.Close()) })
			serviceMust(t, s.Resize(1))
			a, b := unumeral.NewRational(), unumeral.NewRational()
			serviceMust(t, q.SetString(&a, f.value))
			serviceMust(t, q.SetInt64(&b, 31))
			serviceMust(t, s.Add(0, []unumeral.Rational{a}, &b))
			serviceMust(t, q.Del(&a))
			serviceMust(t, q.Del(&b))
			out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			serviceMust(t, out.SetIntegerBase(16))
			serviceMust(t, out.SetShowBase(true))
			serviceMust(t, out.SetFill('_'))
			serviceMust(t, out.SetWidth(f.width))
			serviceMust(t, s.Display(linearNumericDisplayWriter{out}))
			text, e := out.String()
			serviceMust(t, e)
			if text != f.want {
				t.Fatal("source SMALL numeric/BIG string insertions", text, f.want)
			}
		})
	}
}
func TestLinearModularDisplayReceivingNumericFlags(t *testing.T) {
	for _, f := range []struct {
		name, value, want string
		width             int64
	}{
		{"small", "15", "_____0xf 0x1f\n", 8},
		{"big-positive", "4294967296", "4294967296 0x1f\n", 5},
		{"big-negative", "-4294967296", "____-4294967296 0x1f\n", 5},
	} {
		t.Run(f.name, func(t *testing.T) {
			q, budget := rationalManager(t, false)
			m, e := unumeral.NewIntegerModularManager(q.Integers())
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.Close()) })
			// Both SMALL and BIG inputs lie inside this actual field's normalized range.
			serviceMust(t, m.SetZPUint64(2305843009213693951))
			s, e := NewModularLinearSolver(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.Close()) })
			serviceMust(t, s.Resize(1))
			var a unumeral.Int
			serviceMust(t, q.Integers().SetString(&a, f.value))
			b := unumeral.NewInt(31)
			serviceMust(t, s.Add(0, []unumeral.Int{a}, &b))
			serviceMust(t, q.Integers().Del(&a))
			out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			serviceMust(t, out.SetIntegerBase(16))
			serviceMust(t, out.SetShowBase(true))
			serviceMust(t, out.SetFill('_'))
			serviceMust(t, out.SetWidth(f.width))
			serviceMust(t, s.Display(linearNumericDisplayWriter{out}))
			text, e := out.String()
			serviceMust(t, e)
			if text != f.want {
				t.Fatal("underlying integer numeric/string insertions", text, f.want)
			}
		})
	}
}

type linearNumericProbeWriter struct {
	linearNumericDisplayWriter
	probe  func()
	called bool
}

func (w *linearNumericProbeWriter) AppendInt32(v int32) error {
	if !w.called {
		w.called = true
		w.probe()
	}
	return w.NumericTextStream.AppendInt32(v)
}
func TestLinearNumericInsertionCallbackBorrowsActualOwners(t *testing.T) {
	q, budget := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Close()) })
	serviceMust(t, s.Resize(1))
	a, b := unumeral.NewRational(), unumeral.NewRational()
	serviceMust(t, q.SetFraction(&a, 15, 31))
	serviceMust(t, q.SetInt64(&b, 31))
	serviceMust(t, s.Add(0, []unumeral.Rational{a}, &b))
	serviceMust(t, q.Del(&a))
	serviceMust(t, q.Del(&b))
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	serviceMust(t, out.SetIntegerBase(16))
	serviceMust(t, out.SetShowBase(true))
	w := linearNumericProbeWriter{linearNumericDisplayWriter: linearNumericDisplayWriter{out}}
	w.probe = func() {
		before, aa, bb := budget.Used(), s.A[0][0], s.b[0]
		for _, e := range []error{s.Reset(), s.Close(), s.SourceFinalize()} {
			if !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("numeric output acquired solver", e)
			}
		}
		for _, e := range []error{q.CloseChecked(), q.Integers().CloseChecked()} {
			if !errors.Is(e, unumeral.ErrNumericPending) {
				t.Fatal("numeric output acquired borrowed manager", e)
			}
		}
		if !s.running || s.closed || s.A[0][0] != aa || s.b[0] != bb || budget.Used() != before {
			t.Fatal("callback changed actual owner/input state")
		}
	}
	serviceMust(t, s.Display(&w))
	text, e := out.String()
	serviceMust(t, e)
	if !w.called || s.running || text != "0xf/0x1f 0x1f\n" {
		t.Fatal("numeric continuation/borrow lifetime", text, w.called)
	}
}

func TestLinearTraceRetainsReceivingNumericFlags(t *testing.T) {
	q, budget := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Close()); serviceMust(t, q.BindTrace(nil)) })
	serviceMust(t, s.Resize(1))
	a, b := unumeral.NewRational(), unumeral.NewRational()
	serviceMust(t, q.SetFraction(&a, 15, 31))
	serviceMust(t, q.SetInt64(&b, 31))
	serviceMust(t, s.Add(0, []unumeral.Rational{a}, &b))
	serviceMust(t, q.Del(&a))
	serviceMust(t, q.Del(&b))
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	serviceMust(t, out.SetIntegerBase(16))
	serviceMust(t, out.SetShowBase(true))
	tags := diagnostic.NewTraceTagsWithWriter(linearNumericDisplayWriter{out})
	tags.Enable("linear_eq_solver")
	if !tags.ShouldTrace("linear_eq_solver") {
		t.Skip("source TRACE erased")
	}
	serviceMust(t, q.BindTrace(tags))
	result := []unumeral.Rational{unumeral.NewRational()}
	ok, e := s.Solve(result)
	serviceMust(t, e)
	serviceMust(t, q.Del(&result[0]))
	text, e := out.String()
	serviceMust(t, e)
	if !ok || !strings.Contains(text, "linear_eq_solver.h:0x5a ---------\n") || !strings.Contains(text, "iteration 0\n0xf/0x1f 0x1f\n") {
		t.Fatal("trace lost receiving numeric flags", ok, text)
	}
}

type linearFormatterProbe struct {
	want    string
	probe   func()
	refusal error
	called  bool
}

func (w *linearFormatterProbe) Write(p []byte) (int, error) {
	if e := w.AppendCString(string(p)); e != nil {
		return 0, e
	}
	return len(p), nil
}
func (w *linearFormatterProbe) AppendInt32(int32) error { return nil }
func (w *linearFormatterProbe) AppendCString(s string) error {
	if !w.called && s == w.want {
		w.called = true
		w.probe()
		return w.refusal
	}
	return nil
}

func TestLinearRationalDisplayOwnsExactFormatterCallback(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Close()) })
	serviceMust(t, s.Resize(1))
	a, b := unumeral.NewRational(), unumeral.NewRational()
	serviceMust(t, q.SetString(&a, "18446744073709551617"))
	serviceMust(t, q.SetInt64(&b, 1))
	serviceMust(t, s.Add(0, []unumeral.Rational{a}, &b))
	serviceMust(t, q.Del(&a))
	serviceMust(t, q.Del(&b))
	refusal := errors.New("rational magnitude refused")
	w := linearFormatterProbe{want: "18446744073709551617", refusal: refusal}
	w.probe = func() {
		if !s.displayInsertion.Active() || s.displayInsertion.Character == nil {
			t.Fatal("BIG output callback lacks exact formatter receipt")
		}
		for _, e := range []error{s.Reset(), s.Close()} {
			if !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("formatter callback acquired solver", e)
			}
		}
	}
	if body := s.Display(&w); !errors.Is(body, refusal) || !w.called || s.displayInsertion.Active() || s.numericPending() {
		t.Fatal("rational formatter callback cleanup", body, w.called, s.displayInsertion.Active())
	}
}

func TestLinearModularDisplayOwnsExactFormatterCallback(t *testing.T) {
	q, _ := rationalManager(t, false)
	z := q.Integers()
	m, e := unumeral.NewIntegerModularManager(z)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	serviceMust(t, m.SetZPUint64(2305843009213693951))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Close()) })
	serviceMust(t, s.Resize(1))
	var a unumeral.Int
	serviceMust(t, z.SetString(&a, "4294967296"))
	b := unumeral.NewInt(1)
	serviceMust(t, s.Add(0, []unumeral.Int{a}, &b))
	serviceMust(t, z.Del(&a))
	refusal := errors.New("modular magnitude refused")
	w := linearFormatterProbe{want: "4294967296", refusal: refusal}
	w.probe = func() {
		if !s.displayInsertion.Active() || s.displayInsertion.Character == nil {
			t.Fatal("BIG modular output callback lacks exact formatter receipt")
		}
		if e := s.Reset(); !errors.Is(e, ErrPolynomialPending) {
			t.Fatal("formatter callback acquired modular solver", e)
		}
	}
	if body := s.Display(&w); !errors.Is(body, refusal) || !w.called || s.displayInsertion.Active() || s.numericPending() {
		t.Fatal("modular formatter callback cleanup", body, w.called, s.displayInsertion.Active())
	}
}
