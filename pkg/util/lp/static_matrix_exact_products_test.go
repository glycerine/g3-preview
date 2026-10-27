package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLPExactStaticProductsIndependentRationalOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	r := rand.New(rand.NewPCG(4191, 798))
	for world := 0; world < 160; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("0", "0")
			out := h.pair("0", "0")
			m := NewRationalStaticMatrix(h.m, 4, 5)
			t.Cleanup(func() { h.must(m.Close()) })
			v := NewIndexedRationalVector(h.m, 5)
			pv := NewIndexedPairVector(h.m, 5)
			dst := NewIndexedRationalVector(h.m, 6)
			t.Cleanup(func() { h.must(dst.Close()); h.must(pv.Close()); h.must(v.Close()) })
			sample := func() *big.Rat {
				n := new(big.Int).Lsh(big.NewInt(int64(r.IntN(41)-20)), uint(r.IntN(140)))
				return new(big.Rat).SetFrac(n, big.NewInt(int64(1+r.IntN(39))))
			}
			want := make(map[IndexPair]*big.Rat)
			for _, k := range r.Perm(20) {
				if r.IntN(4) == 0 {
					continue
				}
				x := sample()
				i, j := uint32(k/5), uint32(k%5)
				h.must(h.m.SetString(&p.X, x.RatString()))
				h.must(m.AddNewElement(i, j, &p.X))
				want[IndexPair{i, j}] = x
			}
			vr, xr, yr := make([]*big.Rat, 5), make([]*big.Rat, 5), make([]*big.Rat, 5)
			for i := range vr {
				vr[i], xr[i], yr[i] = sample(), sample(), sample()
				h.must(h.m.SetString(&v.Data[i], vr[i].RatString()))
				h.must(h.m.SetString(&pv.Data[i].X, xr[i].RatString()))
				h.must(h.m.SetString(&pv.Data[i].Y, yr[i].RatString()))
			}
			for i := uint32(0); i < 4; i++ {
				sum, sx, sy := new(big.Rat), new(big.Rat), new(big.Rat)
				var lo, hi *big.Rat
				for j := uint32(0); j < 5; j++ {
					if x := want[IndexPair{i, j}]; x != nil {
						sum.Add(sum, new(big.Rat).Mul(x, vr[j]))
						sx.Add(sx, new(big.Rat).Mul(x, xr[j]))
						sy.Add(sy, new(big.Rat).Mul(x, yr[j]))
						a := new(big.Rat).Abs(x)
						if lo == nil || a.Cmp(lo) < 0 {
							lo = new(big.Rat).Set(a)
						}
						if hi == nil || a.Cmp(hi) > 0 {
							hi = new(big.Rat).Set(a)
						}
					}
				}
				if lo == nil {
					lo = new(big.Rat)
					hi = new(big.Rat)
				}
				h.must(m.DotRow(i, v.Data, &out.X))
				lpSparseRationalCheck(t, h, &out.X, sum)
				h.must(m.DotPairRow(i, pv.Data, out))
				h.check(out, sx, sy)
				h.must(m.MinAbsRow(i, &out.X))
				lpSparseRationalCheck(t, h, &out.X, lo)
				h.must(m.MaxAbsRow(i, &out.X))
				lpSparseRationalCheck(t, h, &out.X, hi)
			}
			a := sample()
			h.must(h.m.SetString(&p.Y, a.RatString()))
			for j := uint32(0); j < 5; j++ {
				sum := new(big.Rat)
				var lo, hi *big.Rat
				for i := uint32(0); i < 4; i++ {
					if x := want[IndexPair{i, j}]; x != nil {
						sum.Add(sum, new(big.Rat).Mul(x, vr[i]))
						a := new(big.Rat).Abs(x)
						if lo == nil || a.Cmp(lo) < 0 {
							lo = new(big.Rat).Set(a)
						}
						if hi == nil || a.Cmp(hi) > 0 {
							hi = new(big.Rat).Set(a)
						}
					}
				}
				if lo == nil {
					lo = new(big.Rat)
					hi = new(big.Rat)
				}
				h.must(m.DotColumn(v.Data, j, &out.X))
				lpSparseRationalCheck(t, h, &out.X, sum)
				h.must(m.MinAbsColumn(j, &out.X))
				lpSparseRationalCheck(t, h, &out.X, lo)
				h.must(m.MaxAbsColumn(j, &out.X))
				lpSparseRationalCheck(t, h, &out.X, hi)
				// Dense extraction resizes, preserving old entries wherever a column
				// has no nonzero term; the source does not clear the destination.
				h.must(dst.resizeData(6))
				dst.Index = []uint32{0}
				for i := range dst.Data {
					h.must(h.m.SetInt64(&dst.Data[i], int64(11+i)))
				}
				h.must(m.CopyColumnToVector(j, dst))
				if dst.DataSize() != 4 || !reflect.DeepEqual(dst.Index, []uint32{0}) {
					t.Fatal("dense shape/index")
				}
				for i := uint32(0); i < 4; i++ {
					x := want[IndexPair{i, j}]
					if x == nil || x.Sign() == 0 {
						x = big.NewRat(int64(11+i), 1)
					}
					lpSparseRationalCheck(t, h, &dst.Data[i], x)
				}
				h.must(dst.ClearAll())
				dst.Index = nil
				h.must(m.CopyColumnToIndexedVector(j, dst))
				var indices []uint32
				for _, c := range m.Columns[j] {
					x := want[IndexPair{c.Row, j}]
					if x.Sign() != 0 {
						indices = append(indices, c.Row)
						lpSparseRationalCheck(t, h, &dst.Data[c.Row], x)
					}
				}
				if !reflect.DeepEqual(dst.Index, indices) {
					t.Fatal("sparse order", dst.Index, indices)
				}
				h.must(dst.ClearAll())
				dst.Index = nil
				for i := range dst.Data {
					h.must(h.m.Copy(&dst.Data[i], &v.Data[i]))
				}
				h.must(m.AddColumnToVector(&p.Y, j, dst.Data))
				for i := uint32(0); i < 4; i++ {
					x := new(big.Rat).Set(vr[i])
					if c := want[IndexPair{i, j}]; c != nil {
						x.Add(x, new(big.Rat).Mul(a, c))
					}
					lpSparseRationalCheck(t, h, &dst.Data[i], x)
				}
			}
			lpExactStaticCheck(t, h, m, want)
			h.must(m.Close())
			h.must(dst.Close())
			h.must(pv.Close())
			h.must(v.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactStaticProductsLiteralAliasesAndCopies(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("2", "7")
	out := h.pair("99", "100")
	m := NewRationalStaticMatrix(h.m, 3, 3)
	t.Cleanup(func() { h.must(m.Close()) })
	v := NewIndexedRationalVector(h.m, 3)
	pv := NewIndexedPairVector(h.m, 3)
	t.Cleanup(func() { h.must(v.Close()); h.must(pv.Close()) })
	h.must(m.Set(0, 0, &p.X))
	h.must(m.Set(1, 0, &p.Y))
	h.must(h.m.SetInt64(&v.Data[0], 3))
	h.must(h.m.SetInt64(&v.Data[1], 5))
	h.must(m.AddColumnToVector(&v.Data[0], 0, v.Data))
	lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(9, 1))
	lpSparseRationalCheck(t, h, &v.Data[1], big.NewRat(68, 1))
	// A returned sum must not overwrite an aliased input until all terms have
	// been consumed. This row has two terms, both referring to that same slot.
	h.must(m.Set(0, 0, &p.Y))
	h.must(m.DotRow(0, v.Data, &v.Data[0]))
	lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(81, 1))
	h.must(pv.Data[0].Copy(h.m, p))
	h.must(m.DotPairRow(0, pv.Data, &pv.Data[0]))
	h.check(&pv.Data[0], big.NewRat(18, 1), big.NewRat(63, 1))
	// Duplicate extraction writes the second value before the debug assertion.
	h.must(v.ClearAll())
	v.Index = nil
	err := m.CopyColumnToIndexedVector(0, v)
	if sourceDebugTests {
		if err != ErrAssertion {
			t.Fatal(err)
		}
	} else {
		h.must(err)
	}
	lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(7, 1))
	if sourceDebugTests {
		if !reflect.DeepEqual(v.Index, []uint32{0, 1}) {
			t.Fatal(v.Index)
		}
	} else if !reflect.DeepEqual(v.Index, []uint32{0, 1, 0}) {
		t.Fatal(v.Index)
	}
	if m.ready() != nil {
		t.Fatal("matrix adopted assertion stop")
	}
	h.must(v.ClearAll())
	v.Index = nil
	h.must(m.CopyColumnToVector(0, v))
	lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(7, 1))
	h.must(m.DotRow(2, nil, &out.X))
	h.must(m.DotPairRow(2, nil, out))
	h.check(out, new(big.Rat), new(big.Rat))
	h.must(m.DotColumn(nil, 2, &out.X))
	lpSparseRationalCheck(t, h, &out.X, new(big.Rat))
	h.must(m.MinAbsRow(2, &out.X))
	h.must(m.MaxAbsColumn(2, &out.Y))
	h.check(out, new(big.Rat), new(big.Rat))
	h.must(m.Close())
	h.must(v.Close())
	h.must(pv.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactStaticProductsAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"dense", "indexed", "add", "dot-row", "dot-column", "dot-pair", "min-row", "max-row", "min-column", "max-column"} {
		failures, successes, comparisons, publishedX := 0, 0, 0, 0
		for ordinal := int64(0); ordinal < 300; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
				out := h.pair("99", "100")
				m := NewRationalStaticMatrix(h.m, 2, 2)
				v := NewIndexedRationalVector(h.m, 2)
				dst := NewIndexedRationalVector(h.m, 1)
				pv := NewIndexedPairVector(h.m, 2)
				t.Cleanup(func() {
					h.budget.FailAfter(-1)
					h.must(m.Close())
					h.must(dst.Close())
					h.must(v.Close())
					h.must(pv.Close())
				})
				h.must(m.Set(0, 0, &p.X))
				h.must(m.Set(0, 1, &p.Y))
				h.must(m.Set(1, 0, &p.Y))
				h.must(h.m.Copy(&v.Data[0], &p.Y))
				h.must(h.m.Copy(&v.Data[1], &p.X))
				h.must(pv.Data[0].Copy(h.m, p))
				h.must(pv.Data[1].Copy(h.m, p))
				x, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
				y, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 100) + "/13")
				h.budget.FailAfter(ordinal)
				var err error
				switch op {
				case "dense":
					err = m.CopyColumnToVector(0, dst)
				case "indexed":
					h.must(dst.resizeData(2))
					err = m.CopyColumnToIndexedVector(0, dst)
				case "add":
					err = m.AddColumnToVector(&p.X, 0, v.Data)
				case "dot-row":
					err = m.DotRow(0, v.Data, &out.X)
				case "dot-column":
					err = m.DotColumn(v.Data, 0, &out.X)
				case "dot-pair":
					err = m.DotPairRow(0, pv.Data, out)
				case "min-row":
					err = m.MinAbsRow(0, &out.X)
				case "max-row":
					err = m.MaxAbsRow(0, &out.X)
				case "min-column":
					err = m.MinAbsColumn(0, &out.X)
				case "max-column":
					err = m.MaxAbsColumn(0, &out.X)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					successes++
					done = true
					switch op {
					case "dense", "indexed":
						lpSparseRationalCheck(t, h, &dst.Data[0], x)
						lpSparseRationalCheck(t, h, &dst.Data[1], y)
					case "add":
						lpSparseRationalCheck(t, h, &v.Data[0], new(big.Rat).Add(y, new(big.Rat).Mul(x, x)))
						lpSparseRationalCheck(t, h, &v.Data[1], new(big.Rat).Add(x, new(big.Rat).Mul(x, y)))
					case "dot-row", "dot-column":
						lpSparseRationalCheck(t, h, &out.X, new(big.Rat).Mul(big.NewRat(2, 1), new(big.Rat).Mul(x, y)))
					case "dot-pair":
						h.check(out, new(big.Rat).Mul(x, new(big.Rat).Add(x, y)), new(big.Rat).Mul(y, new(big.Rat).Add(x, y)))
					case "min-row", "min-column":
						lpSparseRationalCheck(t, h, &out.X, x)
					case "max-row", "max-column":
						lpSparseRationalCheck(t, h, &out.X, y)
					}
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
					if op == "dot-pair" {
						text, e := h.m.String(&out.X)
						if e == nil {
							got, ok := new(big.Rat).SetString(text)
							want := new(big.Rat).Mul(x, new(big.Rat).Add(x, y))
							if ok && got.Cmp(want) == 0 {
								publishedX++
							}
						}
					}
					if m.calls.comparison != 0 {
						comparisons++
					}
					if op == "dense" || op == "indexed" {
						if m.ready() != nil || m.calls.pending() {
							t.Fatal("adopted vector failure")
						}
						if dst.calls.pending() {
							if e := m.CopyColumnToVector(0, dst); e != ErrExactVectorPending {
								t.Fatal(e)
							}
							if e := m.CopyColumnToIndexedVector(0, dst); e != ErrExactVectorPending {
								t.Fatal(e)
							}
						}
						// Matrix cleanup cannot consume the destination's roots or frame.
						before := dst.calls
						h.must(m.Close())
						if !reflect.DeepEqual(before, dst.calls) {
							t.Fatal("consumed destination receipts")
						}
					} else {
						if m.ready() != ErrExactVectorPending {
							t.Fatal("lost lexical work")
						}
						if e := m.DotRow(0, v.Data, &out.X); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := m.DotColumn(v.Data, 0, &out.X); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := m.DotPairRow(0, pv.Data, out); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := m.AddColumnToVector(&p.X, 0, v.Data); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := m.MinAbsRow(0, &out.X); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := m.MinAbsColumn(0, &out.X); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := m.CopyColumnToVector(0, dst); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e := m.CopyColumnToIndexedVector(0, dst); e != ErrExactVectorPending {
							t.Fatal(e)
						}
					}
				}
				h.must(m.Close())
				h.must(dst.Close())
				h.must(v.Close())
				h.must(pv.Close())
				lpExactVectorNamedCensus(t, h, nil)
				if h.m.ComparisonInvocationState().Active || h.m.DivisionInvocationState().Active || h.m.RawInvocationState().Active != [3]bool{} {
					t.Fatal("orphan frame")
				}
			})
			if done {
				break
			}
		}
		if op == "dot-pair" && publishedX == 0 {
			t.Fatal("no final X-before-Y output-copy refusal")
		}
		if strings.HasPrefix(op, "min-") || strings.HasPrefix(op, "max-") {
			if comparisons == 0 {
				t.Fatal("no real comparison receipt", op)
			}
		}
		if failures == 0 || successes != 1 {
			t.Fatal("incomplete campaign", op, failures, successes)
		}
	}
}

func TestLPExactStaticProductsBoundsAndPrefixes(t *testing.T) {
	for _, op := range []string{"dense-column", "indexed-column", "add-column", "dot-row", "dot-column", "dot-pair", "min-row", "min-column", "dense-link", "indexed-link", "add-link", "dot-link", "extrema-link", "row-short", "pair-short", "column-short", "add-short", "dense-grow", "zero-skip"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("2", "3")
			out := h.pair("99", "100")
			m := NewRationalStaticMatrix(h.m, 2, 2)
			v := NewIndexedRationalVector(h.m, 1)
			pv := NewIndexedPairVector(h.m, 1)
			t.Cleanup(func() { h.must(m.Close()); h.must(v.Close()); h.must(pv.Close()) })
			h.must(m.Set(0, 0, &p.X))
			h.must(m.Set(0, 1, &p.Y))
			h.must(m.Set(1, 0, &p.Y))
			h.must(h.m.SetInt64(&v.Data[0], 5))
			h.must(pv.Data[0].Copy(h.m, p))
			var err error
			want := ErrBounds
			switch op {
			case "dense-column":
				err = m.CopyColumnToVector(7, v)
				if v.DataSize() != 2 {
					t.Fatal("resize before bounds lost")
				}
				lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(5, 1))
				lpSparseRationalCheck(t, h, &v.Data[1], new(big.Rat))
			case "indexed-column":
				err = m.CopyColumnToIndexedVector(7, v)
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "add-column":
				err = m.AddColumnToVector(&p.X, 7, v.Data)
			case "dot-row":
				err = m.DotRow(7, v.Data, &out.X)
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "dot-column":
				err = m.DotColumn(v.Data, 7, &out.X)
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "dot-pair":
				err = m.DotPairRow(7, pv.Data, out)
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "min-row":
				err = m.MinAbsRow(7, &out.X)
			case "min-column":
				err = m.MinAbsColumn(7, &out.X)
			case "dense-link", "indexed-link", "add-link", "dot-link", "extrema-link":
				m.Columns[0][1].Offset = 7
				switch op {
				case "dense-link":
					err = m.CopyColumnToVector(0, v)
					lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(2, 1))
				case "indexed-link":
					h.must(v.resizeData(2))
					err = m.CopyColumnToIndexedVector(0, v)
					lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(2, 1))
					if !reflect.DeepEqual(v.Index, []uint32{0}) {
						t.Fatal(v.Index)
					}
				case "add-link":
					h.must(v.resizeData(2))
					err = m.AddColumnToVector(&p.X, 0, v.Data)
					lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(9, 1))
				case "dot-link":
					h.must(v.resizeData(2))
					err = m.DotColumn(v.Data, 0, &out.X)
				case "extrema-link":
					err = m.MinAbsColumn(0, &out.X)
				}
			case "row-short":
				err = m.DotRow(0, v.Data, &out.X)
			case "pair-short":
				err = m.DotPairRow(0, pv.Data, out)
			case "column-short":
				err = m.DotColumn(v.Data, 0, &out.X)
			case "add-short":
				err = m.AddColumnToVector(&p.X, 0, v.Data)
				lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(9, 1))
			case "dense-grow":
				err = m.CopyColumnToVector(1, v)
				want = nil
				lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(3, 1))
				lpSparseRationalCheck(t, h, &v.Data[1], new(big.Rat))
			case "zero-skip":
				h.must(h.m.SetInt64(&m.Rows[0][0].Value, 0))
				err = m.CopyColumnToVector(0, v)
				want = nil
				lpSparseRationalCheck(t, h, &v.Data[0], big.NewRat(5, 1))
				lpSparseRationalCheck(t, h, &v.Data[1], big.NewRat(3, 1))
			}
			if err != want {
				t.Fatal(err, want)
			}
			// Local sum/extremum returns have not reached result publication on a
			// bounds stop, even if earlier terms have accumulated successfully.
			h.check(out, big.NewRat(99, 1), big.NewRat(100, 1))
			h.must(m.Close())
			h.must(v.Close())
			h.must(pv.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	t.Run("foreign destination", func(t *testing.T) {
		h := newLPPairHarness(t)
		other := newLPPairHarness(t)
		m := NewRationalStaticMatrix(h.m, 0, 0)
		v := NewIndexedRationalVector(other.m, 0)
		if e := m.CopyColumnToVector(0, v); e != ErrExactVectorOwner {
			t.Fatal(e)
		}
		if e := m.CopyColumnToIndexedVector(0, v); e != ErrExactVectorOwner {
			t.Fatal(e)
		}
		h.must(m.Close())
		other.must(v.Close())
	})
}

func TestLPExactStaticProductsOlderComparisonIdentity(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			z := h.pair("2", "3")
			out := h.pair("0", "0")
			m := NewRationalStaticMatrix(h.m, 1, 2)
			t.Cleanup(func() { h.must(m.Close()) })
			h.must(m.Set(0, 0, &z.X))
			h.must(m.Set(0, 1, &z.Y))
			var caller exactPriorityCalls
			b, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&p.X, &p.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, b, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				matched = true
				// Inline integer extrema complete without entering rational comparison.
				h.must(m.MaxAbsRow(0, &out.X))
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(3, 1))
				h.must(m.MinAbsColumn(0, &out.X))
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(2, 1))
				h.must(m.Close())
				if h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed older comparison")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real stopped comparison")
	}
}
