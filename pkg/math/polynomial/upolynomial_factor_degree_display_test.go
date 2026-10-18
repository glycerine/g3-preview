package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func factorDegreeDisplayOwner(t *testing.T, degrees ...uint32) *UPolynomialDegreeSet {
	t.Helper()
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	for _, degree := range degrees {
		coefficients := make([]int32, int(degree)+1)
		coefficients[0], coefficients[degree] = 1, 1
		p := upolySourceNew(t, m, coefficients)
		serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), 1))
	}
	d, e := NewUPolynomialDegreeSet(fs)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.Close()) })
	return d
}

func TestUPolynomialFactorDegreeDisplayLogicalBitsAndSourceZeroPrefix(t *testing.T) {
	for _, test := range []struct {
		name       string
		degrees    []uint32
		removeZero bool
		want       string
	}{
		{"zero", nil, false, "[0] represented by 1"},
		{"subset", []uint32{2, 2}, false, "[0, 2, 4] represented by 10101"},
		{"removed-zero", []uint32{2, 2}, true, "[0, 2, 4] represented by 10100"},
		{"boundary", []uint32{32, 32}, false, "[0, 32, 64] represented by 1" + strings.Repeat("0", 31) + "1" + strings.Repeat("0", 31) + "1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := factorDegreeDisplayOwner(t, test.degrees...)
			if test.removeZero {
				serviceMust(t, d.Remove(0))
			}
			before := d.budget.Used()
			d.budget.FailAfter(0)
			t.Cleanup(func() { d.budget.FailAfter(-1) })
			var out strings.Builder
			serviceMust(t, d.Display(&out))
			if out.String() != test.want {
				t.Fatal("independent source representation", out.String(), test.want)
			}
			if d.budget.Used() != before || d.Stats().WordBytes != 0 {
				t.Fatal("display acquired numeric/packed-word storage")
			}
		})
	}
}

type factorDegreeDisplayProbe struct {
	t            *testing.T
	owner, other *UPolynomialDegreeSet
	called       bool
}

func (w *factorDegreeDisplayProbe) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		before := w.owner.budget.Used()
		for _, e := range []error{w.owner.Close(), w.owner.Remove(0), w.owner.Intersect(w.other), w.owner.Swap(w.other), w.other.Swap(w.owner), w.owner.Display(io.Discard)} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Error("host callback acquired displayed degree owner", e)
			}
		}
		included, e := w.owner.InSet(0)
		if e != nil || !included || w.owner.budget.Used() != before {
			w.t.Error("callback changed displayed flags/owner", included, e)
		}
	}
	return len(p), nil
}

func TestUPolynomialFactorDegreeDisplayCallbackProtectsNativeOwner(t *testing.T) {
	d, other := factorDegreeDisplayOwner(t, 2), factorDegreeDisplayOwner(t)
	w := factorDegreeDisplayProbe{t: t, owner: d, other: other}
	serviceMust(t, d.Display(&w))
	if !w.called || d.displaying {
		t.Fatal("actual output callback/lifetime")
	}
	serviceMust(t, d.Remove(0))
}

type factorDegreeStoppedWriter struct {
	calls int
	seen  strings.Builder
	cause error
}

func (w *factorDegreeStoppedWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 2 {
		return 0, w.cause
	}
	return w.seen.Write(p)
}

func TestUPolynomialFactorDegreeDisplayOutputRefusalReleasesBorrow(t *testing.T) {
	d := factorDegreeDisplayOwner(t, 2)
	marker := errors.New("degree writer stopped")
	w := factorDegreeStoppedWriter{cause: marker}
	if e := d.Display(&w); !errors.Is(e, marker) || w.calls != 2 || w.seen.String() != "[0" || d.displaying {
		t.Fatal("reached output refusal", e, w.calls, w.seen.String())
	}
	if e := d.Display(linearShortWriter{}); !errors.Is(e, io.ErrShortWrite) || d.displaying {
		t.Fatal("short write borrow", e)
	}
	if e := d.Display(nil); !errors.Is(e, arena.ErrConfig) || d.displaying {
		t.Fatal("nil writer borrow", e)
	}
	var out strings.Builder
	serviceMust(t, d.Display(&out))
	if out.String() != "[0, 2] represented by 101" {
		t.Fatal("later display after host refusal", out.String())
	}
}
