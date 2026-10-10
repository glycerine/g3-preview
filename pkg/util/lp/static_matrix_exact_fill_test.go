package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLPExactStaticFillIndependentEliminationOracle(t *testing.T) {
	r := rand.New(rand.NewPCG(79113, 59))
	for world := 0; world < 240; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("0", "17/19")
			m := NewRationalStaticMatrix(h.m, 3, 8)
			term := NewLARTerm(h.m)
			var it *LARTermIterator
			t.Cleanup(func() {
				if it != nil {
					h.must(it.Close())
				}
				h.must(m.Close())
				h.must(term.Close())
			})
			h.must(term.SetValue(&p.Y))
			want := make(map[IndexPair]*big.Rat)
			sample := func() *big.Rat {
				n := int64(1 + r.IntN(19))
				if r.IntN(2) == 0 {
					n = -n
				}
				return new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(n), uint(r.IntN(130))), big.NewInt(int64(1+r.IntN(29))))
			}
			basis := [2]map[uint32]*big.Rat{make(map[uint32]*big.Rat), make(map[uint32]*big.Rat)}
			for i := uint32(0); i < 2; i++ {
				h.must(h.m.SetInt64(&p.X, 1))
				h.must(m.Set(i, i, &p.X))
				want[IndexPair{i, i}] = big.NewRat(1, 1)
				for _, jj := range r.Perm(5) {
					j := uint32(jj + 2)
					if r.IntN(3) == 0 {
						continue
					}
					x := sample()
					h.must(h.m.SetString(&p.X, x.RatString()))
					h.must(m.Set(i, j, &p.X))
					want[IndexPair{i, j}] = x
					basis[i][j] = x
				}
			}
			coeff := make(map[uint32]*big.Rat)
			for _, jj := range r.Perm(7) {
				j := uint32(jj)
				if r.IntN(4) == 0 {
					continue
				}
				x := sample()
				if j == 4 && world%3 == 0 {
					x = new(big.Rat)
					for i := uint32(0); i < 2; i++ {
						if c := coeff[i]; c != nil {
							if b := basis[i][j]; b != nil {
								x.Add(x, new(big.Rat).Mul(c, b))
							}
						}
					}
					if x.Sign() == 0 {
						continue
					}
				}
				h.must(h.m.SetString(&p.X, x.RatString()))
				h.must(term.AddCoefficient(j, &p.X))
				coeff[j] = x
			}
			// Independent dense expression: -term + its normalized basis rows.
			for j := uint32(2); j < 7; j++ {
				x := new(big.Rat)
				if c := coeff[j]; c != nil {
					x.Neg(c)
				}
				for i := uint32(0); i < 2; i++ {
					if c := coeff[i]; c != nil {
						if b := basis[i][j]; b != nil {
							x.Add(x, new(big.Rat).Mul(c, b))
						}
					}
				}
				if x.Sign() != 0 {
					want[IndexPair{2, j}] = x
				}
			}
			want[IndexPair{2, 7}] = big.NewRat(1, 1)
			var e error
			it, e = NewLARTermIterator(term, 7)
			h.must(e)
			heading := []int32{0, 1, -1, -1, -1, -1, -1, 2}
			h.must(m.FillLastRowWithPivoting(it, heading))
			lpExactStaticCheck(t, h, m, want)
			for j := uint32(0); j < 8; j++ {
				x := want[IndexPair{2, j}]
				if x == nil || j == 7 {
					x = new(big.Rat)
				}
				lpSparseRationalCheck(t, h, &m.WorkVector.Data[j], x)
			}
			if m.work != nil {
				t.Fatal("retained completed lexical values")
			}
			lpSparseRationalCheck(t, h, &term.Value, big.NewRat(17, 19))
			h.must(it.Close())
			h.must(m.Close())
			h.must(term.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactStaticFillLiteralWorkspaceAndPublication(t *testing.T) {
	for _, mode := range []string{"zero-fill", "zero-index-duplicate", "negative-heading", "stale-workspace", "repeat-and-clear"} {
		t.Run(mode, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("2", "3")
			m := NewRationalStaticMatrix(h.m, 2, 3)
			term := NewLARTerm(h.m)
			h.must(m.Set(0, 0, &p.Y))
			h.must(m.Set(0, 1, &p.Y))
			h.must(term.AddCoefficient(0, &p.X))
			heading := []int32{0, -1, 1}
			if mode == "zero-fill" {
				h.must(h.m.SetInt64(&m.Rows[0][1].Value, 0))
			}
			if mode == "zero-index-duplicate" {
				h.must(h.m.SetInt64(&p.Y, 0))
				h.must(term.AddCoefficient(1, &p.Y))
			}
			if mode == "negative-heading" {
				heading[2] = -1
			}
			if mode == "stale-workspace" {
				h.must(m.WorkVector.resizeData(3))
				h.must(h.m.SetInt64(&m.WorkVector.Data[1], 5))
			}
			it, e := NewLARTermIterator(term, 2)
			h.must(e)
			t.Cleanup(func() { h.must(it.Close()); h.must(m.Close()); h.must(term.Close()) })
			h.must(m.FillLastRowWithPivoting(it, heading))
			// No normalization/division of the basis row is invented: its diagonal
			// three is simply skipped and the nonbasic coefficient contributes 2*3.
			switch mode {
			case "zero-fill":
				if !reflect.DeepEqual(m.WorkVector.Index, []uint32{1}) || len(m.Rows[1]) != 1 {
					t.Fatal("stored-zero workspace", m.WorkVector.Index, len(m.Rows[1]))
				}
				lpSparseRationalCheck(t, h, &m.WorkVector.Data[1], new(big.Rat))
			case "zero-index-duplicate":
				if !reflect.DeepEqual(m.WorkVector.Index, []uint32{1, 1}) || len(m.Rows[1]) != 3 {
					t.Fatal("literal duplicate", m.WorkVector.Index, len(m.Rows[1]))
				}
			case "negative-heading":
				if len(m.Rows[1]) != 3 || m.Rows[1][0].Column != 2 || m.Rows[1][2].Column != 2 {
					t.Fatal("repeated unit publication")
				}
			case "stale-workspace":
				lpSparseRationalCheck(t, h, &m.WorkVector.Data[1], big.NewRat(11, 1))
			case "repeat-and-clear":
				h.must(it.Close())
				h.must(m.AddColumn())
				h.must(m.AddRow())
				it, e = NewLARTermIterator(term, 3)
				h.must(e)
				h.must(m.FillLastRowWithPivoting(it, []int32{0, -1, 1, 2}))
				lpSparseRationalCheck(t, h, &m.Rows[2][0].Value, big.NewRat(6, 1))
				size := m.WorkVector.Size()
				h.must(m.Clear())
				if m.WorkVector.Size() != size || m.WorkVector.DataSize() != 4 {
					t.Fatal("Clear erased persistent workspace")
				}
			}
			h.must(it.Close())
			h.must(m.Close())
			h.must(term.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactStaticFillAllocationFailureCensus(t *testing.T) {
	for _, mode := range []string{"eliminate", "publish", "cancel"} {
		failures, successes, iteratorStops, workspaceStops, matrixStops, linkStops := 0, 0, 0, 0, 0, 0
		for ordinal := int64(0); ordinal < 280; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				xs := strings.Repeat("1234567890", 100) + "/7"
				ys := strings.Repeat("9876543210", 100) + "/13"
				p := h.pair(xs, ys)
				m := NewRationalStaticMatrix(h.m, 2, 3)
				term := NewLARTerm(h.m)
				one := h.pair("1", "0")
				h.must(m.Set(0, 0, &one.X))
				h.must(m.Set(0, 1, &p.Y))
				key := uint32(0)
				if mode == "publish" {
					key = 1
				}
				h.must(term.AddCoefficient(key, &p.X))
				x, _ := new(big.Rat).SetString(xs)
				y, _ := new(big.Rat).SetString(ys)
				expected := new(big.Rat).Mul(x, y)
				if mode == "publish" {
					expected.Neg(x)
				}
				if mode == "cancel" {
					product := h.pair(expected.RatString(), "0")
					h.must(term.AddCoefficient(1, &product.X))
					expected = new(big.Rat)
				}
				it, e := NewLARTermIterator(term, 2)
				h.must(e)
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(it.Close()); h.must(m.Close()); h.must(term.Close()) })
				h.budget.FailAfter(ordinal)
				err := m.FillLastRowWithPivoting(it, []int32{0, -1, 1})
				h.budget.FailAfter(-1)
				if err == nil {
					successes++
					done = true
					out := h.pair("0", "0")
					h.must(m.Element(1, 1, &out.X))
					lpSparseRationalCheck(t, h, &out.X, expected)
					h.must(m.Element(1, 2, &out.X))
					lpSparseRationalCheck(t, h, &out.X, big.NewRat(1, 1))
					if expected.Sign() == 0 && len(m.Rows[1]) != 1 {
						t.Fatal("cancellation publication")
					}
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
					if it.calls.pending() {
						iteratorStops++
					} else if m.work != nil && m.work.Data[1].Num.Val == 0 && !m.calls.pending() && !m.WorkVector.calls.pending() {
						basisPresent := false
						processed := uint32(0)
						for _, j := range m.WorkVector.Index {
							if j == 2 {
								basisPresent = true
							}
							for _, k := range it.keys {
								if j == k {
									processed++
								}
							}
						}
						if basisPresent && it.offset == processed {
							iteratorStops++
						} // Copy stopped before advancing the term cursor.
					}
					if m.WorkVector.calls.pending() {
						workspaceStops++
					} else if m.work != nil && m.work.Data[1].Num.Val < 0 && !it.calls.pending() && !m.calls.pending() {
						hasIndex := false
						for _, j := range m.WorkVector.Index {
							if j == key {
								hasIndex = true
							}
						}
						if !hasIndex {
							workspaceStops++
						} // Negation completed; SetValue has not published its index.
					}
					if m.calls.pending() {
						matrixStops++
					}
					if len(m.Columns[1]) > 0 && m.Columns[1][len(m.Columns[1])-1].Row == 1 && len(m.Rows[1]) == 0 {
						linkStops++
						if !m.calls.pending() {
							matrixStops++
						} // The matrix's row-value copy failed before acquiring a pending receipt.
					}
					if m.ready() != ErrExactVectorPending {
						t.Fatal("lost stopped fill")
					}
					if e := m.FillLastRowWithPivoting(it, nil); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.Set(0, 0, &p.X); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.Clear(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					before := it.calls
					h.must(m.Close())
					if !reflect.DeepEqual(before, it.calls) {
						t.Fatal("matrix consumed borrowed iterator receipt")
					}
				}
				h.must(it.Close())
				h.must(m.Close())
				h.must(term.Close())
				lpExactVectorNamedCensus(t, h, nil)
				if h.m.RawInvocationState().Active != [3]bool{} || h.m.DivisionInvocationState().Active {
					t.Fatal("orphan numeral frame")
				}
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 {
			t.Fatal("incomplete fill campaign", mode, failures, successes)
		}
		if iteratorStops == 0 || workspaceStops == 0 || matrixStops == 0 {
			t.Fatal("missing actual numeric owner", mode, iteratorStops, workspaceStops, matrixStops)
		}
		if mode != "cancel" && linkStops == 0 {
			t.Fatal("no source column-before-row publication stop", mode)
		}
	}
}

func TestLPExactStaticFillBoundsAndLiteralStops(t *testing.T) {
	for _, mode := range []string{"empty-rows", "short-heading", "bad-basis-row", "bad-row-column", "empty-columns", "zero-pivot", "duplicate-basis", "lazy-workspace", "bad-workspace-index"} {
		t.Run(mode, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("2", "0")
			m := NewRationalStaticMatrix(h.m, 2, 3)
			term := NewLARTerm(h.m)
			h.must(m.Set(0, 0, &p.X))
			h.must(m.Set(0, 1, &p.X))
			h.must(term.AddCoefficient(0, &p.X))
			heading := []int32{0, -1, 1}
			basis := uint32(2)
			want := ErrBounds
			switch mode {
			case "empty-rows":
				h.must(m.Clear())
				h.must(m.InitRowColumns(0, 3))
				heading = []int32{-1, -1, -1}
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "short-heading":
				heading = nil
			case "bad-basis-row":
				heading[2] = 7
			case "bad-row-column":
				m.Rows[0][1].Column = 7
			case "empty-columns":
				h.must(m.Clear())
				h.must(m.InitRowColumns(1, 0))
				basis = 0
			case "zero-pivot":
				h.must(term.AddCoefficient(0, &p.X))
				h.must(h.m.SetInt64(&p.X, -4))
				h.must(term.AddCoefficient(0, &p.X))
				h.must(term.AddCoefficient(0, &p.Y))
				want = nil
			case "duplicate-basis":
				h.must(term.AddCoefficient(2, &p.X))
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "bad-workspace-index":
				h.must(m.WorkVector.resizeData(1))
				m.WorkVector.Index = []uint32{7}
			case "lazy-workspace":
				h.must(m.WorkVector.Close())
				m.WorkVector = nil
				want = nil
			}
			it, e := NewLARTermIterator(term, basis)
			h.must(e)
			t.Cleanup(func() { h.must(it.Close()); h.must(m.Close()); h.must(term.Close()) })
			if e = m.FillLastRowWithPivoting(it, heading); e != want {
				t.Fatal(e, want)
			}
			if mode == "zero-pivot" && !reflect.DeepEqual(m.WorkVector.Index, []uint32{0}) {
				t.Fatal("zero alpha skip", m.WorkVector.Index)
			}
			h.must(it.Close())
			h.must(m.Close())
			h.must(term.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	t.Run("foreign term owner", func(t *testing.T) {
		h := newLPPairHarness(t)
		other := newLPPairHarness(t)
		term := NewLARTerm(other.m)
		m := NewRationalStaticMatrix(h.m, 1, 1)
		it, e := NewLARTermIterator(term, 0)
		other.must(e)
		if e = m.FillLastRowWithPivoting(it, []int32{0}); e != ErrExactVectorOwner {
			t.Fatal(e)
		}
		h.must(m.Close())
		other.must(it.Close())
		other.must(term.Close())
	})
}

func TestLPExactStaticFillAndTermLeaveOlderComparisonOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			z := h.pair("1", "3")
			m := NewRationalStaticMatrix(h.m, 2, 3)
			term := NewLARTerm(h.m)
			v := NewIndexedRationalVector(h.m, 3)
			h.must(m.Set(0, 0, &z.X))
			h.must(m.Set(0, 1, &z.Y))
			h.must(term.AddCoefficient(0, &z.X))
			h.must(term.AddCoefficient(2, &z.Y))
			h.must(v.SetValue(&z.X, 1))
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
				h.must(term.Substitute(2, v)) // Leaves coefficient1=-3 and removes2.
				it, e := NewLARTermIterator(term, 2)
				h.must(e)
				h.must(m.FillLastRowWithPivoting(it, []int32{0, -1, 1}))
				out := h.pair("0", "0")
				h.must(m.Element(1, 1, &out.X))
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(6, 1))
				h.must(it.Close())
				h.must(m.Close())
				h.must(term.Close())
				h.must(v.Close())
				if h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed older comparison")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			h.must(m.Close())
			h.must(term.Close())
			h.must(v.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual stopped comparison")
	}
}
