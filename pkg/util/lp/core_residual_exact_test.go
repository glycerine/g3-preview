package lp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestLPPairCoreResidualIndependentMatrixOracle(t *testing.T) {
	for world := 0; world < 180; world++ {
		h := newLPPairHarness(t)
		rows := 1 + world%4
		cols := rows + 2
		f := lpNewPairCoreFixture(t, h, uint32(rows), uint32(cols))
		x, y := make([]*big.Rat, cols), make([]*big.Rat, cols)
		for j := 0; j < cols; j++ {
			x[j] = big.NewRat(int64(world%17+j-8), 7)
			y[j] = big.NewRat(int64(world%19-j-9), 11)
			h.must(f.x.Data[j].Copy(h.m, h.pair(x[j].RatString(), y[j].RatString())))
		}
		dots := make([][2]*big.Rat, rows)
		for i := 0; i < rows; i++ {
			dots[i] = [2]*big.Rat{new(big.Rat), new(big.Rat)}
			for j := 0; j < cols; j++ {
				a := big.NewRat(int64((world+i+j)%7-3), 13)
				if j == i {
					a.SetInt64(1)
				}
				if a.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(a.RatString(), "0").X))
				}
				dots[i][0].Add(dots[i][0], new(big.Rat).Mul(a, x[j]))
				dots[i][1].Add(dots[i][1], new(big.Rat).Mul(a, y[j]))
			}
			h.must(f.b.Data[i].Copy(h.m, h.pair(dots[i][0].RatString(), dots[i][1].RatString())))
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		m.TotalIterations = uint32(world + 3)
		mismatch := -1
		dx, dy := big.NewRat(0, 1), big.NewRat(0, 1)
		if world%3 != 0 {
			mismatch = world % rows
			if world%2 == 0 {
				dx.SetFrac64(-3, 17)
			} else {
				dy.SetFrac64(5, 19)
			}
			rhsx := new(big.Rat).Add(dots[mismatch][0], dx)
			rhsy := new(big.Rat).Add(dots[mismatch][1], dy)
			h.must(f.b.Data[mismatch].Copy(h.m, h.pair(rhsx.RatString(), rhsy.RatString())))
		}
		var got bool
		var err error
		output := lpCaptureFeasibleStdout(t, func() { got, err = m.AMultXIsOff() })
		h.must(err)
		if got != (mismatch >= 0) {
			t.Fatal(world, got, mismatch)
		}
		want := ""
		if mismatch >= 0 {
			rhsx := new(big.Rat).Add(dots[mismatch][0], dx)
			rhsy := new(big.Rat).Add(dots[mismatch][1], dy)
			want = fmt.Sprintf("precise x is off (m_b[%d] = (%s, %s) left side = (%s, %s) delta = (%s, %s) iters = %d)\n", mismatch, rhsx.RatString(), rhsy.RatString(), dots[mismatch][0].RatString(), dots[mismatch][1].RatString(), dx.RatString(), dy.RatString(), m.TotalIterations)
		}
		if output != want {
			t.Fatal(world, output, want)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

type lpResidualFailWriter struct {
	bytes.Buffer
	remaining int
}

var lpErrResidualWrite = errors.New("residual write refusal")

func (w *lpResidualFailWriter) Write(p []byte) (int, error) {
	if len(p) > w.remaining {
		n := w.remaining
		w.Buffer.Write(p[:n])
		w.remaining = 0
		return n, lpErrResidualWrite
	}
	w.remaining -= len(p)
	return w.Buffer.Write(p)
}

// Expose only the fallible io.Writer surface in the diagnostic tests.
type lpResidualWriterOnly struct{ writer *lpResidualFailWriter }

func (w lpResidualWriterOnly) Write(p []byte) (int, error) { return w.writer.Write(p) }

type lpResidualMutatingWriter struct {
	bytes.Buffer
	f       *lpPairCoreFixture
	changed bool
}

func (w *lpResidualMutatingWriter) Write(p []byte) (int, error) {
	if string(p) == "left side = " && !w.changed {
		w.changed = true
		if e := w.f.h.m.SetInt32(&w.f.x.Data[0].X, 5); e != nil {
			return 0, e
		}
	}
	return w.Buffer.Write(p)
}

func TestLPPairCoreResidualDiagnosticRecomputeAndWriterPrefixes(t *testing.T) {
	want := "precise x is off (m_b[0] = (7, 0) left side = (3, 0) delta = (4, 0) iters = 9)\n"
	for limit := 0; limit < len(want); limit++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 1, 1)
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.b.Data[0].Copy(h.m, h.pair("7", "0")))
		h.must(f.x.Data[0].Copy(h.m, h.pair("3", "0")))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		m.TotalIterations = 9
		m.residualWork = NewIndexedPairVector(h.m, 2)
		h.must(f.a.DotPairRow(0, f.x.Data, &m.residualWork.Data[0]))
		h.must(m.buildRowResidual(0))
		w := &lpResidualFailWriter{remaining: limit}
		e = m.printRowResidual(0, lpResidualWriterOnly{w})
		if !errors.Is(e, lpErrResidualWrite) || w.String() != want[:limit] {
			t.Fatal(limit, e, w.String(), want[:limit])
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 1, 1)
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.b.Data[0].Copy(h.m, h.pair("7", "0")))
	h.must(f.x.Data[0].Copy(h.m, h.pair("3", "0")))
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	m.residualWork = NewIndexedPairVector(h.m, 2)
	h.must(f.a.DotPairRow(0, f.x.Data, &m.residualWork.Data[0]))
	h.must(m.buildRowResidual(0))
	w := &lpResidualMutatingWriter{f: f}
	h.must(m.printRowResidual(0, w))
	if !w.changed || !strings.Contains(w.String(), "left side = (5, 0) delta = (4, 0)") {
		t.Fatal("did not recompute", w.String())
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairCoreResidualGuardsAndClosedStdout(t *testing.T) {
	for _, op := range []string{"rhs", "x-size", "closed-output", "stopped-matrix"} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 1, 1)
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.b.Data[0].Copy(h.m, h.pair("1", "0")))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		want := ErrBounds
		switch op {
		case "rhs":
			h.must(f.b.resizeData(0))
		case "x-size":
			h.must(f.x.resizeData(0))
			if sourceDebugEnabled {
				want = ErrAssertion
			}
		case "stopped-matrix":
			e = f.a.DotPairRow(99, f.x.Data, h.pair("0", "0"))
			if e == nil {
				t.Fatal("invalid matrix row accepted")
			}
			want = ErrExactVectorPending
		}
		var off bool
		var err error
		if op == "closed-output" {
			closed, e := os.CreateTemp("/mnt/oldrog/tmp", "g3-residual-closed-")
			h.must(e)
			h.must(closed.Close())
			old := os.Stdout
			os.Stdout = closed
			off, err = m.AMultXIsOff()
			os.Stdout = old
			h.must(os.Remove(closed.Name()))
			if err == nil {
				t.Fatal("closed stdout accepted")
			}
		} else {
			off, err = m.AMultXIsOff()
			if err != want {
				t.Fatal(op, off, err, want)
			}
		}
		if off {
			t.Fatal("error reported as completed check")
		}
		if _, e = m.AMultXIsOff(); e != ErrExactVectorPending {
			t.Fatal("lost stopped residual", e)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

var _ io.Writer = lpResidualWriterOnly{}
