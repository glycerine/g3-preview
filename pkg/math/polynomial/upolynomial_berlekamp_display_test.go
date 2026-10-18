package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strings"
	"testing"
)

type berlekampDisplaySink struct {
	text    strings.Builder
	flushes int
}

func (w *berlekampDisplaySink) Write(p []byte) (int, error) { return w.text.Write(p) }
func (w *berlekampDisplaySink) Flush() error                { w.flushes++; return nil }

func TestUPolynomialBerlekampDisplayActualMatrixRows(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	q := upolyBerlekampMatrix{m: m}
	t.Cleanup(q.close)
	serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
	if q.size != 2 || q.values.size != 5 {
		t.Fatal("actual source matrix dimensions", q.size, q.values.size)
	}
	w := berlekampDisplaySink{}
	before := m.config.Integer.Budget().Used()
	m.config.Integer.Budget().FailAfter(0)
	e := q.display(&w)
	m.config.Integer.Budget().FailAfter(-1)
	serviceMust(t, e)
	if w.text.String() != "0\t0\t\n0\t1\t\n" || w.flushes != 2 || m.config.Integer.Budget().Used() != before {
		t.Fatal("actual matrix row text/endl/storage", w.text.String(), w.flushes)
	}
	rank, e := q.diagonalize()
	serviceMust(t, e)
	if rank != 1 {
		t.Fatal("actual null rank", rank)
	}
	w = berlekampDisplaySink{}
	serviceMust(t, q.display(&w))
	if w.text.String() != "0\t0\t\n0\t-1\t\n" || w.flushes != 2 {
		t.Fatal("actual diagonalized matrix text", w.text.String())
	}
}

type berlekampTraceProbe struct {
	t       *testing.T
	q       *upolyBerlekampMatrix
	f       UPolynomialView
	text    strings.Builder
	checks  map[bool]bool
	flushes int
}

func (w *berlekampTraceProbe) Write(p []byte) (int, error) {
	q := w.q
	if q.displaying {
		stage := q.column[1] >= 0
		if !w.checks[stage] {
			w.checks[stage] = true
			values, size, nullRow := q.values, q.size, q.nullRow
			column, row := append([]int32(nil), q.column...), append([]int32(nil), q.row...)
			q.close()
			var out upolyVectorState
			_, de := q.diagonalize()
			_, ne := q.next(&out)
			for _, e := range []error{q.init(q.m, w.f), de, ne, q.traceMatrix(142), q.m.Close(), q.m.SourceFinalize()} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					w.t.Fatal("matrix callback acquired active owner", e)
				}
			}
			if q.values != values || q.size != size || q.nullRow != nullRow || !reflect.DeepEqual(q.column, column) || !reflect.DeepEqual(q.row, row) || out.backing != 0 {
				w.t.Fatal("callback mutated borrowed matrix")
			}
		}
	}
	return w.text.Write(p)
}
func (w *berlekampTraceProbe) Flush() error { w.flushes++; return nil }
func TestUPolynomialBerlekampWholeMatrixTracesAndBorrowedOwner(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	q := upolyBerlekampMatrix{m: m}
	t.Cleanup(q.close)
	w := berlekampTraceProbe{t: t, q: &q, f: mustUPolyView(t, m, f), checks: map[bool]bool{}}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
	rank, e := q.diagonalize()
	serviceMust(t, e)
	if rank != 1 {
		t.Fatal("matrix trace changed null rank")
	}
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		for _, part := range []string{":97 ", "polynomial::berlekamp_matrix(x^2 + 1, 3)\n", ":142 ", "polynomial::berlekamp_matrix():\n0\t0\t\n0\t1\t\n\n", ":198 ", "polynomial::diagonalize():\n0\t0\t\n0\t-1\t\n\n"} {
			if !strings.Contains(w.text.String(), part) {
				t.Fatal("whole matrix source record", part, w.text.String())
			}
		}
		if len(w.checks) != 2 || w.flushes < 10 {
			t.Fatal("matrix record callback/endl", w.checks, w.flushes)
		}
	} else if w.text.Len() != 0 || len(w.checks) != 0 {
		t.Fatal("erased matrix trace")
	}
	if q.displaying || q.traceBorrow.executing {
		t.Fatal("matrix trace borrow retained")
	}
}
func TestUPolynomialBerlekampMatrixTraceStreamFailure(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	q := upolyBerlekampMatrix{m: m}
	t.Cleanup(q.close)
	w := factorDegreeRefusedTrace{cause: errors.New("refused matrix trace")}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
	rank, e := q.diagonalize()
	serviceMust(t, e)
	if rank != 1 || q.displaying || q.traceBorrow.executing {
		t.Fatal("stream refusal changed matrix/source borrow")
	}
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		if w.calls != 1 || w.flushes != 3 {
			t.Fatal("sticky output or final record flushes", w.calls, w.flushes)
		}
	} else if w.calls != 0 {
		t.Fatal("erased refused trace")
	}
	out := berlekampDisplaySink{}
	serviceMust(t, q.display(&out))
	if out.text.String() != "0\t0\t\n0\t-1\t\n" {
		t.Fatal("stream refusal changed actual matrix", out.text.String())
	}
}

func TestUPolynomialBerlekampMatrixTraceRetainedFactorizationConsumer(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	ok, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	serviceMust(t, e)
	if ok {
		t.Fatal("irreducible source case reported a nontrivial split")
	}
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		if !strings.Contains(text.String(), ":142 ") || !strings.Contains(text.String(), ":198 ") {
			t.Fatal("retained consumer omitted matrix records", text.String())
		}
	} else if text.Len() != 0 {
		t.Fatal("erased retained matrix records")
	}
	n, e := fs.DistinctFactors()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("irreducible factor count", n)
	}
	product := upolySourceNew(t, m, nil)
	serviceMust(t, fs.Multiply(product))
	same, e := m.Equal(f, product)
	serviceMust(t, e)
	if !same {
		t.Fatal("matrix traces changed factorization product")
	}
}
