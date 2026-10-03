// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Tests FIRST: actual registered root-isolation inputs and assertion routes.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

type upolyIsolationCounts struct{ calls, roots, intervals, ensure, verify uint32 }
type upolyIsolationRational struct {
	n int64
	d uint64
}

func upolyIsolationBinaryOwner(t *testing.T, q *mpnum.RationalManager, budget *arena.Budget) *mpnum.BinaryRationalManager {
	t.Helper()
	d, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: budget}}}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := d.Close(); e != nil {
			t.Error(e)
		}
	})
	b, e := mpnum.NewBinaryRationalManager(q.Integers(), mpnum.BinaryRationalConfig{Diagnostics: d})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(b.Close)
	return b
}
func upolyIsolationVector(t *testing.T, b *mpnum.BinaryRationalManager) *mpnum.BinaryRationalVector {
	t.Helper()
	v, e := b.NewVector()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := v.Close(); e != nil {
			t.Error(e)
		}
	})
	return v
}
func upolyIsolationSize(t *testing.T, v *mpnum.BinaryRationalVector) uint32 {
	t.Helper()
	n, e := v.Size()
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func upolyIsolationRead(t *testing.T, b *mpnum.BinaryRationalManager, v *mpnum.BinaryRationalVector, i uint32) mpnum.BinaryRational {
	t.Helper()
	a := mpnum.NewBinaryRational()
	if e := v.Read(i, &a); e != nil {
		t.Fatal(e)
	}
	return a
}
func upolyIsolationAssert(t *testing.T, c *upolyIsolationCounts, verify, condition bool) {
	t.Helper()
	if verify {
		c.verify++
	} else {
		c.ensure++
	}
	if !condition {
		t.Fatal("actual source assertion", *c)
	}
}
func upolyIsolationSign(t *testing.T, m *mpnum.UPolynomialManager, p mpnum.UPolynomialView, b *mpnum.BinaryRational) int {
	t.Helper()
	s, e := m.EvalSignBinary(p, b)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func upolyIsolationVariation(t *testing.T, m *mpnum.UPolynomialManager, s *mpnum.UPolynomialSequence, b *mpnum.BinaryRational) uint32 {
	t.Helper()
	n, e := m.SignVariationsBinary(s, b)
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func upolyIsolationCheck(t *testing.T, m *mpnum.UPolynomialManager, b *mpnum.BinaryRationalManager, p mpnum.UPolynomial, roots, lowers, uppers *mpnum.BinaryRationalVector, c *upolyIsolationCounts) {
	t.Helper()
	view := mustUPolyView(t, m, p)
	if e := m.IsolateRoots(view, b, roots, lowers, uppers); e != nil {
		t.Fatal(e)
	}
	sseq, fseq := upolySturmSequence(t, m), upolySturmSequence(t, m)
	if e := m.SturmSequence(view, sseq); e != nil {
		t.Fatal(e)
	}
	sqf := upolySourceNew(t, m, nil)
	if e := m.SquareFree(p, sqf); e != nil {
		t.Fatal(e)
	}
	sv := mustUPolyView(t, m, sqf)
	if e := m.FourierSequence(sv, fseq); e != nil {
		t.Fatal(e)
	}
	r, i := upolyIsolationSize(t, roots), upolyIsolationSize(t, lowers)
	c.calls++
	c.roots += r
	c.intervals += i
	minus, e := m.SignVariationsMinusInfinity(sseq)
	if e != nil {
		t.Fatal(e)
	}
	plus, e := m.SignVariationsPlusInfinity(sseq)
	if e != nil {
		t.Fatal(e)
	}
	upolyIsolationAssert(t, c, false, r+i == minus-plus) // source104
	for j := uint32(0); j < r; j++ {
		a := upolyIsolationRead(t, b, roots, j)
		upolyIsolationAssert(t, c, false, upolyIsolationSign(t, m, view, &a) == 0)
		if e = b.Del(&a); e != nil {
			t.Fatal(e)
		}
	}
	for j := uint32(0); j < i; j++ {
		a, z := upolyIsolationRead(t, b, lowers, j), upolyIsolationRead(t, b, uppers, j)
		// Keep source endpoint OR evaluation short-circuit order.
		upolyIsolationAssert(t, c, false, upolyIsolationSign(t, m, view, &a) == 0 || upolyIsolationSign(t, m, view, &z) == 0 || upolyIsolationVariation(t, m, sseq, &a)-upolyIsolationVariation(t, m, sseq, &z) == 1)
		fl, fu := upolyIsolationVariation(t, m, fseq, &a), upolyIsolationVariation(t, m, fseq, &z)
		upolyIsolationAssert(t, c, true, upolyIsolationSign(t, m, view, &a) == 0 || upolyIsolationSign(t, m, view, &z) == 0 || (fl-fu >= 1 && (fl-fu-1)%2 == 0))
		bound, e := m.DescartesBoundInterval(sv, b, &a, &z)
		if e != nil {
			t.Fatal(e)
		}
		upolyIsolationAssert(t, c, true, bound == 1)
		if e = b.Del(&a); e != nil {
			t.Fatal(e)
		}
		if e = b.Del(&z); e != nil {
			t.Fatal(e)
		}
	}
}
func upolyIsolationMatch(t *testing.T, q *mpnum.RationalManager, b *mpnum.BinaryRationalManager, roots, lowers, uppers *mpnum.BinaryRationalVector, expected []upolyIsolationRational, c *upolyIsolationCounts) {
	t.Helper()
	r, i := upolyIsolationSize(t, roots), upolyIsolationSize(t, lowers)
	upolyIsolationAssert(t, c, false, uint32(len(expected)) == r+i)
	visited := make([]bool, len(expected))
	for _, want := range expected {
		ex := mpnum.NewRational()
		if e := q.SetFraction(&ex, want.n, want.d); e != nil {
			t.Fatal(e)
		}
		found := false
		for j := uint32(0); j < r; j++ {
			value := upolyIsolationRead(t, b, roots, j)
			rat := mpnum.NewRational()
			if e := b.ToRational(q, &value, &rat); e != nil {
				t.Fatal(e)
			}
			same, e := q.Equal(&rat, &ex)
			if e != nil {
				t.Fatal(e)
			}
			if same {
				upolyIsolationAssert(t, c, false, !visited[j])
				upolyIsolationAssert(t, c, true, !found)
				found = true
				visited[j] = true
			}
			if e = q.Del(&rat); e != nil {
				t.Fatal(e)
			}
			if e = b.Del(&value); e != nil {
				t.Fatal(e)
			}
		}
		for j := uint32(0); j < i; j++ {
			lower, upper := upolyIsolationRead(t, b, lowers, j), upolyIsolationRead(t, b, uppers, j)
			a, z := mpnum.NewRational(), mpnum.NewRational()
			if e := b.ToRational(q, &lower, &a); e != nil {
				t.Fatal(e)
			}
			if e := b.ToRational(q, &upper, &z); e != nil {
				t.Fatal(e)
			}
			left, e := q.Compare(&a, &ex)
			if e != nil {
				t.Fatal(e)
			}
			right := 0
			if left < 0 {
				right, e = q.Compare(&ex, &z)
				if e != nil {
					t.Fatal(e)
				}
			}
			if left < 0 && right < 0 {
				upolyIsolationAssert(t, c, true, !found)
				upolyIsolationAssert(t, c, false, !visited[j+r])
				found = true
				visited[j+r] = true
			}
			for _, v := range []*mpnum.Rational{&a, &z} {
				if e = q.Del(v); e != nil {
					t.Fatal(e)
				}
			}
			if e = b.Del(&lower); e != nil {
				t.Fatal(e)
			}
			if e = b.Del(&upper); e != nil {
				t.Fatal(e)
			}
		}
		upolyIsolationAssert(t, c, false, found)
		if e := q.Del(&ex); e != nil {
			t.Fatal(e)
		}
	}
}
func upolyIsolationProduct(t *testing.T, m *mpnum.UPolynomialManager, factors [][]int32) mpnum.UPolynomial {
	t.Helper()
	p := upolySourceNew(t, m, []int32{1})
	for _, f := range factors {
		factor := upolySourceNew(t, m, f)
		if e := m.Mul(p, factor, p); e != nil {
			t.Fatal(e)
		}
	}
	return p
}
func TestUPolynomialIsolationRegisteredTwentyFourDatasets(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	p17 := make([]int32, 18)
	for _, x := range []struct {
		i int
		v int32
	}{{17, 1}, {16, 5}, {15, 3}, {13, 10}, {10, 13}, {9, 1}, {5, 8}, {2, 3}, {0, 7}} {
		p17[x.i] = x.v
	}
	a5, z5 := []int32{-1, -1, 0, 0, 0, 1}, []int32{-1, 1, 0, 0, 0, 1}
	x3 := []int32{-2, 0, 0, 1}
	recipes := []struct {
		f  [][]int32
		ex []upolyIsolationRational
	}{
		{[][]int32{{-1, 1}, {-2, 1}}, []upolyIsolationRational{{1, 1}, {2, 1}}},
		{[][]int32{{-1, 1}, {-1, 1}, {0, 0, 0, 1}}, []upolyIsolationRational{{1, 1}, {0, 1}}},
		{[][]int32{a5}, []upolyIsolationRational{{11673039, 10000000}}},
		{[][]int32{{-1, 1}, {1, 1}, {2, 1}, {3, 1}, {-3, 1}, {-3, 1}}, []upolyIsolationRational{{1, 1}, {-1, 1}, {-2, 1}, {-3, 1}, {3, 1}}},
		{[][]int32{{-31, 10000}, {-32, 10000}}, []upolyIsolationRational{{31, 10000}, {32, 10000}}},
		{[][]int32{{-31, 10000}, {-32, 10000}, {-33, 10000}}, []upolyIsolationRational{{31, 10000}, {32, 10000}, {33, 10000}}},
		{[][]int32{a5, z5, {-1167, 1000}}, []upolyIsolationRational{{11673039, 10000000}, {75487766, 100000000}, {1167, 1000}}},
		{[][]int32{{-2, 1}, {-4, 1}, {-8, 1}, {-16, 1}, {-32, 1}, {-64, 1}, {-1, 2}, {-1, 4}, {-1, 8}, {-1, 16}, {-1, 32}}, []upolyIsolationRational{{2, 1}, {4, 1}, {8, 1}, {16, 1}, {32, 1}, {64, 1}, {1, 2}, {1, 4}, {1, 8}, {1, 16}, {1, 32}}},
		{[][]int32{a5, z5, {-1167, 1000}, a5, z5, {-1167, 1000}}, []upolyIsolationRational{{11673039, 10000000}, {75487766, 100000000}, {1167, 1000}}},
		{[][]int32{p17}, []upolyIsolationRational{{-413582, 100000}, {-170309, 100000}, {-109968, 100000}}},
		{[][]int32{p17, a5, a5, x3, x3}, []upolyIsolationRational{{-413582, 100000}, {-170309, 100000}, {-109968, 100000}, {11673039, 10000000}, {125992, 100000}}},
		{[][]int32{{-1000000000, 0, 0, 0, 0, 1}, {-1000000000, 0, 0, 0, 0, 1}, {-1000000000, 0, 0, 0, 0, 1}, {-10000000, 3}, {-10000000, 3}, {-632, 10}, {-632, 10}}, []upolyIsolationRational{{630957, 10000}, {10000000, 3}, {632, 10}}},
	}
	counts := upolyIsolationCounts{}
	for repeat := 0; repeat < 2; repeat++ {
		for index, r := range recipes {
			p := upolyIsolationProduct(t, m, r.f)
			upolyIsolationCheck(t, m, b, p, roots, lo, hi, &counts)
			upolyIsolationMatch(t, q, b, roots, lo, hi, r.ex, &counts)
			t.Log("actual source dataset", repeat, index, upolyIsolationSize(t, roots), upolyIsolationSize(t, lo))
		}
	}
	if counts.calls != 24 || counts.roots+counts.intervals != 86 || counts.ensure+counts.verify != 306+counts.roots+3*counts.intervals {
		t.Fatal("actual conditional source counts", counts)
	}
	t.Logf("24 helper calls; R=%d I=%d ENSURE=%d VERIFY=%d", counts.roots, counts.intervals, counts.ensure, counts.verify)
}
func TestUPolynomialIsolationRegisteredDegreeSeventy(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
	coefficients := make([]int32, 71)
	for _, term := range []struct {
		i int
		v int32
	}{{70, 1}, {65, -6}, {60, -1}, {55, 60}, {50, -54}, {45, -230}, {40, 274}, {35, 542}, {30, -615}, {25, -1120}, {20, 1500}, {15, -160}, {10, -395}, {5, 76}, {0, 34}} {
		coefficients[term.i] = term.v
	}
	p := upolySourceNew(t, m, coefficients)
	counts := upolyIsolationCounts{}
	upolyIsolationCheck(t, m, b, p, roots, lo, hi, &counts)
	if counts.calls != 1 || counts.ensure+counts.verify != 1+counts.roots+3*counts.intervals {
		t.Fatal(counts)
	}
	t.Logf("real degree70 R=%d I=%d ENSURE=%d VERIFY=%d", counts.roots, counts.intervals, counts.ensure, counts.verify)
}
