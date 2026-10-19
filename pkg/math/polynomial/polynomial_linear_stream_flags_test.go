package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	utext "github.com/glycerine/g3/pkg/util/text"
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
