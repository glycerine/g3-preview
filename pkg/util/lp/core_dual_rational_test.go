// Source: src/util/lp/lp_core_solver_base.hpp:390-444.
package lp

import (
	"math/big"
	"testing"
)

func TestLPRationalDualBoundsIndependentOracle(t *testing.T) {
	for _, kind := range []ColumnType{ColumnFixed, ColumnBoxed, ColumnLowBound, ColumnUpperBound, ColumnFree} {
		for xi := -2; xi <= 2; xi++ {
			for di := -2; di <= 2; di++ {
				for _, equal := range []bool{false, true} {
					h := newLPPairHarness(t)
					f := lpNewRationalCoreFixture(t, h, 1, 2)
					f.settings.SimplexStrategy = StrategyLU
					h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
					f.types[1] = kind
					lo, up := int64(-1), int64(1)
					if equal {
						up = lo
					}
					h.must(h.m.SetInt32(&f.low.Data[1], int32(lo)))
					h.must(h.m.SetInt32(&f.up.Data[1], int32(up)))
					h.must(h.m.SetInt32(&f.x.Data[1], int32(xi)))
					m, e := NewRationalCoreSolverBase(f.input())
					h.must(e)
					h.must(h.m.Copy(&m.D.Data[1], lpScalarBoundValue(h, big.NewRat(int64(di), 1000000007).RatString())))
					nn, e := m.DIsNotNegative(1)
					h.must(e)
					np, e := m.DIsNotPositive(1)
					h.must(e)
					if nn != (di >= 0) || np != (di <= 0) {
						t.Fatal("scalar threshold", di, nn, np)
					}
					got, e := m.ColumnIsDualFeasible(1)
					want := false
					switch kind {
					case ColumnFixed, ColumnBoxed:
						want = int64(xi) == lo && di >= 0 || int64(xi) == up && di <= 0
					case ColumnLowBound:
						want = int64(xi) == lo && di >= 0
					case ColumnFree, ColumnUpperBound:
						want = di == 0
					}
					if sourceDebugEnabled && kind == ColumnUpperBound {
						if e != ErrAssertion {
							t.Fatal(e)
						}
					} else {
						h.must(e)
						if got != want {
							t.Fatal(kind, xi, di, equal, got, want)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				}
			}
		}
	}
}

func TestLPRationalDualBoundsPlacementAndAssertionPrefix(t *testing.T) {
	for _, kind := range []ColumnType{ColumnFixed, ColumnBoxed, ColumnLowBound, ColumnUpperBound, ColumnFree} {
		for _, sign := range []int32{-1, 0, 1} {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			f.types[1] = kind
			h.must(h.m.SetInt32(&f.x.Data[1], 9))
			h.must(h.m.SetInt32(&f.low.Data[1], -3))
			h.must(h.m.SetInt32(&f.up.Data[1], 4))
			m, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			h.must(h.m.SetInt32(&m.D.Data[1], sign))
			e = m.SetNonBasicXToCorrectBounds()
			want := int64(9)
			switch kind {
			case ColumnBoxed:
				want = -3
				if sign < 0 {
					want = 4
				}
			case ColumnLowBound:
				want = -3
			case ColumnUpperBound:
				want = 4
			}
			if sourceDebugEnabled && (kind == ColumnUpperBound || kind == ColumnLowBound && sign < 0) {
				if e != ErrAssertion {
					t.Fatal(kind, sign, e)
				}
			} else {
				h.must(e)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(want, 1))
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalDualBoundsMetadataAndDiagnostics(t *testing.T) {
	for _, phase := range []string{"d", "types", "x", "low", "upper", "place-x", "place-low", "place-upper", "unknown", "unknown-writer", "upper-writer"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			f.types[1] = ColumnBoxed
			m, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			xs, lo, up, ds, types := f.x.Data, f.low.Data, f.up.Data, m.D.Data, f.types
			switch phase {
			case "d":
				m.D.Data = nil
				_, e = m.DIsNotNegative(1)
				if e != ErrBounds {
					t.Fatal(e)
				}
				_, e = m.DIsNotPositive(1)
			case "types":
				f.types = nil
				_, e = m.ColumnIsDualFeasible(1)
			case "x":
				f.x.Data = nil
				_, e = m.ColumnIsDualFeasible(1)
			case "low":
				f.low.Data = nil
				_, e = m.ColumnIsDualFeasible(1)
			case "upper":
				h.must(h.m.SetInt32(&f.x.Data[1], 1))
				f.up.Data = nil
				_, e = m.ColumnIsDualFeasible(1)
			case "place-x":
				f.x.Data = nil
				e = m.SetNonBasicXToCorrectBounds()
			case "place-low":
				f.low.Data = nil
				e = m.SetNonBasicXToCorrectBounds()
			case "place-upper":
				h.must(h.m.SetInt32(&m.D.Data[1], -1))
				f.up.Data = nil
				e = m.SetNonBasicXToCorrectBounds()
			case "unknown":
				f.types[1] = ColumnType(99)
				var ok bool
				ok, e = m.ColumnIsDualFeasible(1)
				if ok || sourceDebugEnabled && e != ErrAssertion || !sourceDebugEnabled && e != nil {
					t.Fatal("unknown source assertion", ok, e)
				}
			case "unknown-writer":
				f.types[1] = ColumnType(99)
				f.settings.DebugOut = lpRefusingWriter{err: ErrBounds}
				_, e = m.ColumnIsDualFeasible(1)
			case "upper-writer":
				f.types[1] = ColumnUpperBound
				f.settings.DebugOut = lpRefusingWriter{err: ErrBounds}
				_, e = m.ColumnIsDualFeasible(1)
			}
			if e == nil && !(phase == "unknown" && !sourceDebugEnabled) {
				t.Fatal("missing rejection")
			}
			f.x.Data, f.low.Data, f.up.Data, m.D.Data, f.types = xs, lo, up, ds, types
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualBoundsLiveAliasOrder(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
	h.must(f.x.Resize(4))
	can := []bool{false, true, true}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	// The low-bound header borrows shifted model storage. A completed earlier
	// assignment is the next column's bound; no endpoint snapshot is permitted.
	for i, n := range []int32{0, 11, 22, 33} {
		h.must(h.m.SetInt32(&f.x.Data[i], n))
	}
	f.types[1], f.types[2] = ColumnBoxed, ColumnBoxed
	borrowed := f.x.Data[1:]
	m.LowBounds = &borrowed
	h.must(m.SetNonBasicXToCorrectBounds())
	lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(33, 1))
	lpSparseRationalCheck(t, h, &f.x.Data[2], big.NewRat(33, 1))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
