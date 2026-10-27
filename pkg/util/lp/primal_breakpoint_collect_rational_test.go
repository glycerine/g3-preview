package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func lpScalarBreakpointOracle(x, bound, d *big.Rat, typ BreakpointType, direction int32) (*big.Rat, bool) {
	diff := new(big.Rat).Sub(x, bound)
	same := func(v *big.Rat) bool { return v.Sign() > 0 && direction > 0 || v.Sign() < 0 && direction < 0 }
	if diff.Sign() == 0 {
		return diff, typ == BreakpointFixed || (typ == BreakpointLow && same(d)) || (typ == BreakpointUpper && !same(d)) || typ > BreakpointFixed
	}
	delta := new(big.Rat).Quo(diff, d)
	return delta, same(delta)
}

func TestLPRationalPrimalBreakpointCollectIndependentDelta(t *testing.T) {
	for world := 0; world < 600; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 0, 0)
		h.must(m.BreakpointQueue.Resize(1))
		m.SignOfEnteringDelta = int32(world%3 - 1)
		typ := BreakpointType(world / 3 % 3)
		x := big.NewRat(int64(world%17-8), 7)
		bound := big.NewRat(int64(world%23-11), 13)
		d := big.NewRat(int64(world%11+1), 19)
		if world%2 == 0 {
			d.Neg(d)
		}
		if world%7 == 0 {
			x.Set(bound)
			if world%5 == 0 {
				d.SetInt64(0)
			}
		}
		delta, add := lpScalarBreakpointOracle(x, bound, d, typ, m.SignOfEnteringDelta)
		xv, bv, dv := lpScalarBoundValue(h, x.RatString()), lpScalarBoundValue(h, bound.RatString()), lpScalarBoundValue(h, d.RatString())
		h.must(m.TryAddBreakpoint(7, xv, dv, typ, bv))
		count := 0
		if add {
			count = 1
		}
		if len(m.Breakpoints) != count || int(m.BreakpointQueue.Size()) != count {
			t.Fatal(world, add)
		}
		if add {
			b := m.Breakpoints[0]
			if b.Column != 7 || b.Type != typ {
				t.Fatal("metadata")
			}
			lpSparseRationalCheck(t, h, b.Delta, delta)
		}
		lpSparseRationalCheck(t, h, xv, x)
		lpSparseRationalCheck(t, h, bv, bound)
		lpSparseRationalCheck(t, h, dv, d)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalBreakpointCollectIndependentModel(t *testing.T) {
	for world := 0; world < 300; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 2, 3)
		h.must(m.BreakpointQueue.Resize(8))
		m.SignOfEnteringDelta = int32(world%3 - 1)
		type rec struct {
			j   uint32
			typ BreakpointType
			v   *big.Rat
		}
		var want []rec
		order := []uint32{0, 1}
		if world%2 == 0 {
			order = []uint32{1, 0}
		}
		for j := range f.types {
			f.types[j] = ColumnType((world + j) % 5)
			h.must(h.m.SetString(&f.low.Data[j], "-3/7"))
			h.must(h.m.SetString(&f.up.Data[j], "5/11"))
			h.must(h.m.SetString(&f.x.Data[j], big.NewRat(int64(world%13-6+j), 17).RatString()))
		}
		for _, i := range order {
			d := big.NewRat(int64((world+int(i))%9-4), 19)
			h.must(m.Ed.SetValue(lpScalarBoundValue(h, d.RatString()), i))
			if d.Sign() == 0 {
				continue
			}
			x := big.NewRat(int64(world%13-6+int(i)), 17)
			add := func(typ BreakpointType, b *big.Rat) {
				v, ok := lpScalarBreakpointOracle(x, b, d, typ, m.SignOfEnteringDelta)
				if ok {
					want = append(want, rec{i, typ, v})
				}
			}
			switch f.types[i] {
			case ColumnFixed:
				add(BreakpointFixed, big.NewRat(-3, 7))
			case ColumnBoxed:
				add(BreakpointLow, big.NewRat(-3, 7))
				add(BreakpointUpper, big.NewRat(5, 11))
			case ColumnLowBound:
				add(BreakpointLow, big.NewRat(-3, 7))
			case ColumnUpperBound:
				add(BreakpointUpper, big.NewRat(5, 11))
			}
		}
		if f.types[2] == ColumnBoxed {
			v := new(big.Rat).Sub(big.NewRat(5, 11), big.NewRat(-3, 7))
			typ := BreakpointUpper
			if m.SignOfEnteringDelta < 0 {
				v.Neg(v)
				typ = BreakpointLow
			}
			want = append(want, rec{2, typ, v})
		}
		// Clear must precede visiting the actual sparse index, including explicit zeros.
		h.must(m.AddBreakpoint(99, lpScalarBoundValue(h, "17"), BreakpointLow))
		h.must(m.FillBreakpointsArray(2))
		if len(m.Breakpoints) != len(want) || int(m.BreakpointQueue.Size()) != len(want) {
			t.Fatal(world, len(m.Breakpoints), want)
		}
		for i, w := range want {
			b := m.Breakpoints[i]
			if b.Column != w.j || b.Type != w.typ {
				t.Fatal(world, i)
			}
			lpSparseRationalCheck(t, h, b.Delta, w.v)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalBreakpointCollectPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"delta", "row", "fill-positive", "fill-negative"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 900; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalPricingFixture(t, h, 1, 2)
					h.must(m.BreakpointQueue.Resize(8))
					huge := strings.Repeat("1234567890", 12)
					x, b, d := lpScalarBoundValue(h, huge+"1/7"), lpScalarBoundValue(h, "3/11"), lpScalarBoundValue(h, huge+"5/13")
					m.SignOfEnteringDelta = 1
					h.must(m.Ed.SetValue(d, 0))
					f.types[0] = ColumnBoxed
					f.types[1] = ColumnBoxed
					h.must(h.m.Copy(&f.x.Data[0], x))
					h.must(h.m.Copy(&f.low.Data[0], b))
					h.must(h.m.SetString(&f.up.Data[0], huge+"7/17"))
					h.must(h.m.SetString(&f.low.Data[1], "-2/19"))
					h.must(h.m.SetString(&f.up.Data[1], huge+"9/23"))
					if op == "fill-negative" {
						m.SignOfEnteringDelta = -1
					}
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "delta":
						err = m.TryAddBreakpoint(0, x, d, BreakpointLow, b)
					case "row":
						err = m.TryAddBreakpointInRow(0)
					default:
						err = m.FillBreakpointsArray(1)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if e := m.TryAddBreakpointInRow(0); e != ErrExactVectorPending {
							t.Fatal("lost stop", e)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalPrimalBreakpointCollectGuardsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"nil-x", "nil-d", "nil-bound", "row-index", "unprepared-ed", "basis-extent", "column-index", "type-extent", "low-extent", "upper-extent", "unknown", "zero-row", "entering-index", "entering-low", "entering-upper"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 1, 2)
			h.must(m.BreakpointQueue.Resize(4))
			m.SignOfEnteringDelta = 1
			d, x, b := lpScalarBoundValue(h, "2/3"), lpScalarBoundValue(h, "1"), lpScalarBoundValue(h, "0")
			h.must(m.Ed.SetValue(d, 0))
			f.types[0] = ColumnLowBound
			h.must(h.m.Copy(&f.x.Data[0], x))
			want := ErrBounds
			var err error
			switch phase {
			case "nil-x":
				err = m.TryAddBreakpoint(0, nil, d, BreakpointLow, b)
			case "nil-d":
				err = m.TryAddBreakpoint(0, x, nil, BreakpointLow, b)
			case "nil-bound":
				err = m.TryAddBreakpoint(0, x, d, BreakpointLow, nil)
			case "row-index":
				err = m.TryAddBreakpointInRow(99)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "unprepared-ed":
				h.must(m.Ed.Resize(0))
				err = m.TryAddBreakpointInRow(0)
			case "basis-extent":
				f.basis = f.basis[:0]
				err = m.TryAddBreakpointInRow(0)
			case "column-index":
				f.basis[0] = 99
				err = m.TryAddBreakpointInRow(0)
			case "type-extent":
				f.types = f.types[:0]
				err = m.TryAddBreakpointInRow(0)
			case "low-extent":
				old := f.low.Data
				f.low.Data = nil
				err = m.TryAddBreakpointInRow(0)
				f.low.Data = old
			case "upper-extent":
				f.types[0] = ColumnBoxed
				old := f.up.Data
				f.up.Data = nil
				err = m.TryAddBreakpointInRow(0)
				f.up.Data = old
				if len(m.Breakpoints) != 1 {
					t.Fatal("lost boxed low prefix")
				}
			case "unknown", "zero-row":
				f.types[0] = ColumnType(99)
				want = nil
				if phase == "zero-row" {
					h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
					f.basis = nil
				} else if sourceDebugEnabled {
					want = ErrAssertion
				}
				err = m.TryAddBreakpointInRow(0)
			case "entering-index":
				h.must(m.AddBreakpoint(99, x, BreakpointFixed))
				err = m.FillBreakpointsArray(99)
				if len(m.Breakpoints) != 1 || m.Breakpoints[0].Column != 0 {
					t.Fatal("lost row prefix")
				}
			case "entering-low", "entering-upper":
				f.types[1] = ColumnBoxed
				if phase == "entering-low" {
					old := f.low.Data
					f.low.Data = f.low.Data[:1]
					err = m.FillBreakpointsArray(1)
					f.low.Data = old
				} else {
					old := f.up.Data
					f.up.Data = f.up.Data[:1]
					err = m.FillBreakpointsArray(1)
					f.up.Data = old
				}
				if len(m.Breakpoints) != 1 {
					t.Fatal("lost entering prefix")
				}
			}
			if err != want {
				t.Fatal(err, want)
			}
			if err != nil {
				if e := m.TryAddBreakpointInRow(0); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				if e := m.TryAddBreakpoint(0, x, d, BreakpointLow, b); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				if e := m.FillBreakpointsArray(1); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalBreakpointTraversalPreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"ratio", "row", "fill", "advance"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpRationalPricingFixture(t, h, 1, 2)
				h.must(m.BreakpointQueue.Resize(4))
				m.SignOfEnteringDelta = -1
				f.types[0] = ColumnBoxed
				f.types[1] = ColumnBoxed
				h.must(h.m.SetInt32(&f.x.Data[0], -1))
				h.must(h.m.SetInt32(&f.up.Data[0], 1))
				h.must(h.m.SetInt32(&f.up.Data[1], 3))
				h.must(m.Ed.SetValue(lpScalarBoundValue(h, "3/7"), 0))
				h.must(h.m.SetInt32(&m.D.Data[1], 2))
				h.must(m.AddBreakpoint(0, lpScalarBoundValue(h, "-1/3"), BreakpointLow))
				h.must(m.AddBreakpoint(0, lpScalarBoundValue(h, "-2/3"), BreakpointUpper))
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
				h.budget.FailAfter(ordinal)
				_, err := h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				older := h.m.ComparisonInvocationState()
				if older.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					switch op {
					case "ratio":
						err = m.TryAddBreakpoint(0, &f.x.Data[0], &m.Ed.Data[0], BreakpointLow, &f.up.Data[0])
					case "row":
						err = m.TryAddBreakpointInRow(0)
					case "fill":
						err = m.FillBreakpointsArray(1)
					case "advance":
						_, err = m.AdvanceOnSortedBreakpoints(1, lpScalarBoundValue(h, "97"))
					}
					if m.calls.comparison != 0 || m.BreakpointQueue.calls.comparison != 0 {
						t.Fatal("adopted caller comparison", err)
					}
				}
				h.must(m.Close())
				f.close()
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("destroyed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual comparison refusal", op)
		}
	}
}
