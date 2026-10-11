package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

type lpRationalCoreFixture struct {
	h                    *lpPairHarness
	a                    *RationalStaticMatrix
	b, x, low, up, costs *IndexedRationalVector
	basis, nb            []uint32
	heading              []int32
	types                []ColumnType
	settings             *Settings
}

func lpNewRationalCoreFixture(t *testing.T, h *lpPairHarness, rows, cols uint32) *lpRationalCoreFixture {
	t.Helper()
	s, e := NewSettings()
	h.must(e)
	s.DebugOut = nil
	s.MessageOut = nil
	f := &lpRationalCoreFixture{h: h, a: NewRationalStaticMatrix(h.m, rows, cols), b: NewIndexedRationalVector(h.m, rows), x: NewIndexedRationalVector(h.m, cols), low: NewIndexedRationalVector(h.m, cols), up: NewIndexedRationalVector(h.m, cols), costs: NewIndexedRationalVector(h.m, cols), basis: make([]uint32, rows), nb: []uint32{91}, heading: []int32{92}, types: make([]ColumnType, cols), settings: s}
	for i := range f.basis {
		f.basis[i] = uint32(i)
	}
	return f
}
func (f *lpRationalCoreFixture) input() RationalCoreInputs {
	return RationalCoreInputs{A: f.a, B: &f.b.Data, X: &f.x.Data, LowBounds: &f.low.Data, UpperBounds: &f.up.Data, Costs: &f.costs.Data, Basis: &f.basis, NBasis: &f.nb, Heading: &f.heading, ColumnTypes: &f.types, Settings: f.settings}
}
func (f *lpRationalCoreFixture) close() {
	for _, v := range []*IndexedRationalVector{f.costs, f.up, f.low, f.x, f.b} {
		f.h.must(v.Close())
	}
	f.h.must(f.a.Close())
}
func TestLPRationalCoreIndependentLUSolves(t *testing.T) {
	rng := rand.New(rand.NewSource(82719417))
	for world := 0; world < 80; world++ {
		h := newLPPairHarness(t)
		rows := 1 + world%4
		cols := rows + 2
		f := lpNewRationalCoreFixture(t, h, uint32(rows), uint32(cols))
		perm := rng.Perm(cols)
		for i := range f.basis {
			f.basis[i] = uint32(perm[i])
		}
		b := make([][]*big.Rat, rows)
		d, y := make([]*big.Rat, rows), make([]*big.Rat, rows)
		for i := 0; i < rows; i++ {
			b[i] = make([]*big.Rat, rows)
			d[i] = big.NewRat(int64(rng.Intn(31)-15), int64(1+rng.Intn(19)))
			y[i] = big.NewRat(int64(rng.Intn(31)-15), int64(1+rng.Intn(19)))
			for j := 0; j < rows; j++ {
				v := new(big.Rat)
				if j < i {
					n := new(big.Int).Lsh(big.NewInt(int64(i+j+1)), uint(81+world))
					if world%2 == 0 {
						n.Neg(n)
					}
					v.SetFrac(n, big.NewInt(23))
				}
				if i == j {
					v.SetFrac64(int64(i+2), int64(j+3))
				}
				b[i][j] = v
				h.must(f.a.Set(uint32(i), f.basis[j], &h.pair(v.RatString(), "0").X))
			}
		}
		enter := uint32(perm[rows])
		for i := 0; i < rows; i++ {
			v := new(big.Rat)
			for j := 0; j < rows; j++ {
				v.Add(v, new(big.Rat).Mul(b[i][j], d[j]))
			}
			h.must(f.a.Set(uint32(i), enter, &h.pair(v.RatString(), "0").X))
		}
		for j := 0; j < rows; j++ {
			v := new(big.Rat)
			for i := 0; i < rows; i++ {
				v.Add(v, new(big.Rat).Mul(y[i], b[i][j]))
			}
			h.must(f.costs.copySlot(&f.costs.Data[f.basis[j]], &h.pair(v.RatString(), "0").X))
		}
		f.settings.SimplexStrategy = StrategyLU
		if world%2 == 1 {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		core, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		if core.Status != StatusFeasible || core.TotalIterations != 0 || core.IterationsNoCostGrowth != 0 || core.UsingInfeasCosts || core.Tracing || core.PivotedRows != nil || core.LookForFeasibleSolutionOnly || core.BasisSortCounter != 0 || len(core.ColumnsNZ) != 0 || len(core.RowsNZ) != 0 {
			t.Fatal("constructor defaults")
		}
		if len(core.Y.Data) != rows || len(core.CopyXB.Data) != rows || len(core.D.Data) != cols || len(core.PivotRow.Data) != cols || len(core.PivotRowB1.Data) != rows || len(core.SteepestEdge.Data) != cols {
			t.Fatal("scratch extents")
		}
		wantH := make([]int32, cols)
		var wantNB []uint32
		for j := cols - 1; j >= 0; j-- {
			at := -1
			for i, k := range f.basis {
				if k == uint32(j) {
					at = i
				}
			}
			if at >= 0 {
				wantH[j] = int32(at)
			} else {
				wantNB = append(wantNB, uint32(j))
				wantH[j] = -int32(len(wantNB))
			}
		}
		if !reflect.DeepEqual(f.heading, wantH) || !reflect.DeepEqual(f.nb, wantNB) {
			t.Fatal("source descending non-basis")
		}
		ok, e := core.BasisHeadingIsCorrect()
		h.must(e)
		if !ok || !core.CurrentXIsFeasible() || core.CurrentXIsInfeasible() {
			t.Fatal("initial basis/inf set")
		}
		h.must(core.InfSet.Insert(enter))
		if core.CurrentXIsFeasible() || !core.CurrentXIsInfeasible() {
			t.Fatal("inf set query")
		}
		core.InfSet.Erase(enter)
		column := NewIndexedRationalVector(h.m, uint32(rows))
		if world%2 == 1 {
			if core.Factorization != nil {
				t.Fatal("tableau constructor factored")
			}
			f.settings.SimplexStrategy = StrategyLU
		}
		h.must(core.SolveBdIndexed(enter, column))
		if core.Factorization == nil || core.Factorization.GetStatus() != LUOK {
			t.Fatal("actual rational factorization")
		}
		for i := 0; i < rows; i++ {
			lpSparseRationalCheck(t, h, &column.Data[i], d[i])
		}
		core.ColumnsNZ = make([]uint32, cols)
		h.must(core.SolveBd(enter))
		for i := 0; i < rows; i++ {
			lpSparseRationalCheck(t, h, &core.Ed.Data[i], d[i])
		}
		if core.ColumnsNZ[enter] != core.Ed.Size() {
			t.Fatal("nonzero estimate")
		}
		h.must(core.FillCB(core.Y))
		for i, j := range f.basis {
			lpSparseRationalCheck(t, h, &core.Y.Data[i], func() *big.Rat {
				s, e := h.m.String(&f.costs.Data[j])
				h.must(e)
				r, _ := new(big.Rat).SetString(s)
				return r
			}())
		}
		h.must(core.SolveYB(core.Y))
		for i := 0; i < rows; i++ {
			lpSparseRationalCheck(t, h, &core.Y.Data[i], y[i])
		}
		old := core.Factorization
		h.must(core.Init())
		if old == core.Factorization || !old.closed {
			t.Fatal("refactor did not replace old owner")
		}
		if err := old.SolveBdFaster(enter, column); err != ErrLUClosed {
			t.Fatal("refactored LU accepted a stale borrow", err)
		}
		h.must(core.Close())
		h.must(core.Close())
		h.must(column.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalCoreExactBoundProfiles(t *testing.T) {
	for typ := ColumnFree; typ <= ColumnFixed; typ++ {
		for _, tolerance := range []float64{0, -1, 1e99, math.Inf(1), math.NaN()} {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 0, 1)
			f.settings.PrimalFeasibilityTolerance = tolerance
			f.types[0] = typ
			lo, up := big.NewRat(-7, 13), big.NewRat(11, 17)
			h.must(f.low.copySlot(&f.low.Data[0], &h.pair(lo.RatString(), "0").X))
			h.must(f.up.copySlot(&f.up.Data[0], &h.pair(up.RatString(), "0").X))
			c, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			c.Settings = nil
			tiny, _ := new(big.Rat).SetString("1/1" + strings.Repeat("0", 100))
			for _, v := range []*big.Rat{new(big.Rat).Sub(lo, tiny), lo, new(big.Rat), up, new(big.Rat).Add(up, tiny)} {
				h.must(f.x.copySlot(&f.x.Data[0], &h.pair(v.RatString(), "0").X))
				want := true
				switch typ {
				case ColumnFixed, ColumnBoxed:
					want = v.Cmp(lo) >= 0 && v.Cmp(up) <= 0
				case ColumnLowBound:
					want = v.Cmp(lo) >= 0
				case ColumnUpperBound:
					want = v.Cmp(up) <= 0
				}
				ok, e := c.ColumnIsFeasible(0)
				h.must(e)
				if ok != want {
					t.Fatal(typ, v, ok, want)
				}
				below, e := c.BelowBound(&f.x.Data[0], &f.low.Data[0])
				h.must(e)
				above, e := c.AboveBound(&f.x.Data[0], &f.low.Data[0])
				h.must(e)
				at, e := c.AtBound(&f.x.Data[0], &f.low.Data[0])
				h.must(e)
				if below != (v.Cmp(lo) < 0) || above != (v.Cmp(lo) > 0) || at != (v.Cmp(lo) == 0) {
					t.Fatal("exact comparison")
				}
			}
			ok, e := c.BoundsForBoxedAreSetCorrectly()
			h.must(e)
			if !ok {
				t.Fatal("valid bounds")
			}
			if typ == ColumnBoxed {
				h.must(f.low.copySlot(&f.low.Data[0], &h.pair("99", "0").X))
				ok, e = c.BoundsForBoxedAreSetCorrectly()
				h.must(e)
				if ok {
					t.Fatal("reversed box accepted")
				}
			}
			h.must(c.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalCoreBasisTraceAndTranspose(t *testing.T) {
	rng := rand.New(rand.NewSource(511017))
	for world := 0; world < 100; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 5)
		f.settings.SimplexStrategy = StrategyTableauRows
		p := rng.Perm(5)
		a, b, c3, d := uint32(p[0]), uint32(p[1]), uint32(p[3]), uint32(p[4])
		f.basis[0], f.basis[1] = a, b
		first, second := big.NewRat(int64(7+world), 13), big.NewRat(int64(11+world), 17)
		h.must(f.a.Set(0, a, &h.pair(first.RatString(), "0").X))
		h.must(f.a.Set(1, b, &h.pair(second.RatString(), "0").X))
		c, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		c.StartTracingBasisChanges()
		h.must(c.ChangeBasis(d, a))
		if !reflect.DeepEqual(c.Trace, []uint32{d, a}) {
			t.Fatal("trace")
		}
		h.must(c.ChangeBasisUnconditionally(a, d))
		if len(c.Trace) != 0 {
			t.Fatal("inverse trace not cancelled")
		}
		c.TraceBasisChange(c3, b)
		c.TraceBasisChange(d, a)
		c.TraceBasisChange(a, d)
		if !reflect.DeepEqual(c.Trace, []uint32{c3, b}) {
			t.Fatal("trace suffix")
		}
		c.StopTracingBasisChanges()
		h.must(c.ChangeBasis(c3, b))
		if len(c.Trace) != 2 {
			t.Fatal("disabled trace changed")
		}
		h.must(c.TransposeRowsTableau(0, 1))
		if !reflect.DeepEqual(f.basis, []uint32{c3, a}) || f.heading[c3] != 0 || f.heading[a] != 1 {
			t.Fatal("basis row swap")
		}
		if len(f.a.Rows[0]) != 1 || len(f.a.Rows[1]) != 1 {
			t.Fatal("row swap extent")
		}
		lpSparseRationalCheck(t, h, &f.a.Rows[0][0].Value, second)
		lpSparseRationalCheck(t, h, &f.a.Rows[1][0].Value, first)
		ok, e := c.BasisHeadingIsCorrect()
		h.must(e)
		if !ok {
			t.Fatal("basis invariant")
		}
		c.StartTracingBasisChanges()
		if !c.Tracing || len(c.Trace) != 0 {
			t.Fatal("trace reset")
		}
		h.must(c.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalCorePhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, phase := range []string{"construct", "refactor", "fill", "yB", "Bd", "indexed", "lazy", "transpose"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 700 && !done; ordinal++ {
				if !t.Run(fmt.Sprintf("direct%t/%s/%d", direct, phase, ordinal), func(t *testing.T) {
					h := newLPPairHarness(t)
					if direct {
						h = lpNewDirectSparseHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 2, 3)
					f.settings.SimplexStrategy = StrategyLU
					large := h.pair(strings.Repeat("1234567890", 20)+"7/13", "0")
					other := h.pair(strings.Repeat("9876543210", 20)+"1/17", "0")
					h.must(f.a.Set(0, 0, &large.X))
					h.must(f.a.Set(1, 0, &other.X))
					h.must(f.a.Set(1, 1, &large.X))
					h.must(f.a.Set(0, 2, &other.X))
					h.must(f.a.Set(1, 2, &other.X))
					for j := range f.costs.Data {
						h.must(f.costs.copySlot(&f.costs.Data[j], &large.X))
					}
					if phase == "lazy" || phase == "transpose" {
						f.settings.SimplexStrategy = StrategyTableauRows
					}
					var c *RationalCoreSolverBase
					var err error
					if phase != "construct" {
						c, err = NewRationalCoreSolverBase(f.input())
						h.must(err)
						c.ColumnsNZ = make([]uint32, 3)
					}
					out := NewIndexedRationalVector(h.m, 2)
					h.budget.FailAfter(ordinal)
					switch phase {
					case "construct":
						c, err = NewRationalCoreSolverBase(f.input())
					case "refactor":
						err = c.Init()
					case "fill":
						err = c.FillCB(c.Y)
					case "yB":
						err = c.SolveYB(c.Y)
					case "Bd":
						err = c.SolveBd(2)
					case "indexed":
						err = c.SolveBdIndexed(2, out)
					case "lazy":
						f.settings.SimplexStrategy = StrategyLU
						err = c.SolveBdIndexed(2, out)
					case "transpose":
						err = c.TransposeRowsTableau(0, 1)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if c != nil && c.Init() != ErrExactVectorPending {
							t.Fatal("stopped core accepted retry")
						}
						if phase == "transpose" && !reflect.DeepEqual(f.basis, []uint32{1, 0}) {
							t.Fatal("matrix failure preceded source basis swap")
						}
					}
					if c != nil {
						h.must(c.Close())
					}
					h.must(out.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				}) {
					t.FailNow()
				}
			}
			if !done || direct && failures == 0 {
				t.Fatal("incomplete physical core campaign", direct, phase, failures)
			}
		}
	}
}

func TestLPRationalCoreConstructorAndDomains(t *testing.T) {
	for field := 0; field < 11; field++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 0, 0)
		in := f.input()
		switch field {
		case 0:
			in.A = nil
		case 1:
			in.B = nil
		case 2:
			in.X = nil
		case 3:
			in.Costs = nil
		case 4:
			in.Basis = nil
		case 5:
			in.NBasis = nil
		case 6:
			in.Heading = nil
		case 7:
			in.ColumnTypes = nil
		case 8:
			in.LowBounds = nil
		case 9:
			in.UpperBounds = nil
		case 10:
			in.Settings = nil
		}
		if c, e := NewRationalCoreSolverBase(in); c != nil || e != ErrBounds {
			t.Fatal(field, c, e)
		}
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	for _, kind := range []string{"bad-box", "bad-basis", "duplicate-basis", "short-heading", "duplicate-nbasis", "wrong-basis-heading", "wrong-nbasis-heading", "column-index", "unknown-type", "upper-read", "lower-read", "free-no-bounds", "nil-factor-yB", "nil-factor-Bd", "late-column-count", "short-output", "short-costs", "nil-output", "nil-settings", "foreign-output", "unconditional-missing-slot", "basis-publication", "transpose-publication", "tableau-indexed"} {
		t.Run(kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 4)
			f.settings.SimplexStrategy = StrategyTableauRows
			one := h.pair("1", "0")
			h.must(f.a.Set(0, 0, &one.X))
			h.must(f.a.Set(1, 1, &one.X))
			h.must(f.a.Set(0, 2, &one.X))
			var c *RationalCoreSolverBase
			var err error
			if kind == "bad-box" {
				f.types[0] = ColumnBoxed
				h.must(f.low.copySlot(&f.low.Data[0], &one.X))
			}
			if kind == "bad-basis" {
				f.basis[1] = 99
			}
			if kind == "duplicate-basis" {
				f.basis[1] = 0
			}
			c, err = NewRationalCoreSolverBase(f.input())
			if kind == "bad-box" || kind == "duplicate-basis" {
				want := error(nil)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				if err != want {
					t.Fatal(err, want)
				}
				h.must(c.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
				return
			}
			if kind == "bad-basis" {
				if err != ErrBounds || !reflect.DeepEqual(f.heading, []int32{0, -1, -1, -1}) || !reflect.DeepEqual(f.nb, []uint32{91}) {
					t.Fatal("constructor prefix", err, f.heading, f.nb)
				}
				h.must(c.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
				return
			}
			h.must(err)
			var want error
			ok := true
			switch kind {
			case "short-heading":
				f.heading = f.heading[:1]
				ok, err = c.BasisHeadingIsCorrect()
				want = ErrBounds
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "duplicate-nbasis":
				f.nb = []uint32{2, 2}
				ok, err = c.BasisHeadingIsCorrect()
				if ok {
					t.Fatal("duplicate accepted")
				}
			case "wrong-basis-heading":
				f.heading[0] = 1
				ok, err = c.BasisHeadingIsCorrect()
				if ok {
					t.Fatal("wrong heading accepted")
				}
			case "wrong-nbasis-heading":
				f.heading[3] = -99
				ok, err = c.BasisHeadingIsCorrect()
				if ok {
					t.Fatal("wrong nonbasis accepted")
				}
			case "column-index":
				ok, err = c.ColumnIsFeasible(99)
				want = ErrBounds
			case "unknown-type":
				f.types[0] = ColumnType(99)
				ok, err = c.ColumnIsFeasible(0)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				if ok {
					t.Fatal("unknown type feasible")
				}
			case "upper-read":
				f.types[0] = ColumnUpperBound
				owner := f.up.Data
				f.up.Data = f.up.Data[:0]
				ok, err = c.ColumnIsFeasible(0)
				f.up.Data = owner
				want = ErrBounds
			case "lower-read":
				f.types[0] = ColumnLowBound
				owner := f.low.Data
				f.low.Data = f.low.Data[:0]
				ok, err = c.ColumnIsFeasible(0)
				f.low.Data = owner
				want = ErrBounds
			case "free-no-bounds":
				f.types[0] = ColumnFree
				lo, up := f.low.Data, f.up.Data
				f.low.Data = nil
				f.up.Data = nil
				ok, err = c.ColumnIsFeasible(0)
				f.low.Data = lo
				f.up.Data = up
				if !ok {
					t.Fatal("free read unused bounds")
				}
			case "nil-factor-yB":
				err = c.SolveYB(c.Y)
				want = ErrBounds
			case "nil-factor-Bd":
				err = c.SolveBd(2)
				want = ErrBounds
			case "late-column-count":
				f.settings.SimplexStrategy = StrategyLU
				h.must(c.Init())
				err = c.SolveBd(2)
				want = ErrBounds
				lpSparseRationalCheck(t, h, &c.Ed.Data[0], big.NewRat(1, 1))
			case "short-output":
				out := NewIndexedRationalVector(h.m, 1)
				err = c.FillCB(out)
				want = ErrBounds
				h.must(out.Close())
			case "short-costs":
				owner := f.costs.Data
				f.costs.Data = f.costs.Data[:0]
				err = c.FillCB(c.Y)
				f.costs.Data = owner
				want = ErrBounds
			case "nil-output":
				err = c.FillCB(nil)
				want = ErrBounds
			case "nil-settings":
				c.Settings = nil
				err = c.Init()
				want = ErrBounds
			case "foreign-output":
				foreign := newLPPairHarness(t)
				out := NewIndexedRationalVector(foreign.m, 2)
				err = c.FillCB(out)
				want = ErrExactVectorOwner
				h.must(out.Close())
			case "unconditional-missing-slot":
				f.heading[2] = -99
				err = c.ChangeBasisUnconditionally(2, 0)
				if err == nil {
					if f.basis[0] != 2 || f.nb[len(f.nb)-1] != 0 {
						t.Fatal("missing slot not appended")
					}
				}
			case "basis-publication":
				f.heading[0] = 99
				err = c.ChangeBasis(2, 0)
				want = ErrBounds
				if f.heading[2] != 99 || f.basis[0] != 0 {
					t.Fatal("early basis exchange prefix")
				}
			case "transpose-publication":
				f.heading = f.heading[:1]
				err = c.TransposeBasis(0, 1)
				want = ErrBounds
				if !reflect.DeepEqual(f.basis, []uint32{1, 0}) {
					t.Fatal("row swap prefix")
				}
			case "tableau-indexed":
				out := NewIndexedRationalVector(h.m, 2)
				err = c.SolveBdIndexed(2, out)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				h.must(out.Close())
			}
			if err != want {
				t.Fatal(kind, err, want)
			}
			h.must(c.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if c.Init() != ErrExactVectorPending {
				t.Fatal("closed init")
			}
			if _, e := c.ColumnIsFeasible(0); e != ErrExactVectorPending {
				t.Fatal("closed bound check")
			}
		})
	}
	// An ownerless matrix cannot supply scalar arithmetic.
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 0, 0)
	f.a.manager = nil
	if c, err := NewRationalCoreSolverBase(f.input()); c != nil || err != ErrExactVectorOwner {
		t.Fatal("ownerless matrix", c, err)
	}
	f.a.manager = h.m
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalCorePreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 0, 1)
		c, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
		var caller exactEtaCalls
		b, e := caller.begin(h.m)
		h.must(e)
		h.budget.FailAfter(ordinal)
		_, e = h.m.Compare(&v.X, &v.Y)
		h.budget.FailAfter(-1)
		e = caller.after(h.m, b, e)
		older := h.m.ComparisonInvocationState()
		if older.Active {
			matched = true
			if !lpExactPriorityPhysicalRefusal(e) {
				t.Fatal(e)
			}
			_, _ = c.AtBound(&f.x.Data[0], &f.low.Data[0])
			if c.calls.comparison != 0 {
				t.Fatal("adopted caller comparison")
			}
		}
		h.must(c.Close())
		if older.Active && h.m.ComparisonInvocationState() != older {
			t.Fatal("consumed caller comparison")
		}
		h.must(caller.cleanupRaw(h.m))
		h.must(caller.cleanupRoots(h.m))
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	if !matched {
		t.Fatal("no actual older comparison")
	}
}

func TestLPRationalCoreComparisonRefusalAndForeignBorrow(t *testing.T) {
	for _, direct := range []bool{false, true} {
		done, failures := false, 0
		for ordinal := int64(0); ordinal < 100 && !done; ordinal++ {
			if !t.Run(fmt.Sprintf("direct%t/%d", direct, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				if direct {
					h = lpNewDirectSparseHarness(t)
				}
				f := lpNewRationalCoreFixture(t, h, 0, 1)
				c, e := NewRationalCoreSolverBase(f.input())
				h.must(e)
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				h.budget.FailAfter(ordinal)
				_, e = c.AtBound(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				if e == nil {
					done = true
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(e) {
						t.Fatal(e)
					}
					if _, e = c.BelowBound(&v.X, &v.Y); e != ErrExactVectorPending {
						t.Fatal("stopped comparison retry")
					}
				}
				h.must(c.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}) {
				t.FailNow()
			}
		}
		if !done || direct && failures == 0 {
			t.Fatal("incomplete physical comparison campaign", direct, failures)
		}
	}
	h := newLPPairHarness(t)
	foreign := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 0, 1)
	c, e := NewRationalCoreSolverBase(f.input())
	h.must(e)
	v := foreign.pair(strings.Repeat("1234567890", 20)+"7/13", "0")
	// The source sign-only comparison does not read foreign backing limbs.
	if below, err := c.BelowBound(&f.x.Data[0], &v.X); err != nil || !below {
		t.Fatal("sign-only prefix", below, err)
	}
	h.must(f.x.copySlot(&f.x.Data[0], &h.pair("1/2", "0").X))
	if _, e = c.BelowBound(&f.x.Data[0], &v.X); e == nil {
		t.Fatal("foreign backing read accepted")
	}
	h.must(c.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
	want, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 20) + "7/13")
	lpSparseRationalCheck(t, foreign, &v.X, want)
	lpExactVectorNamedCensus(t, foreign, nil)
}

func TestLPRationalCoreReachedMetadataGuards(t *testing.T) {
	for _, kind := range []string{"basis-short", "basis-duplicate", "basis-outside", "nb-outside", "debug-reverse", "boxed-extent", "unknown-bounds", "enter-outside", "leave-outside", "enter-basic", "signed-overflow", "late-nbasis", "transpose-outside", "transpose-matrix", "restore-none", "restore-actual", "restore-outside", "nil-indexed", "foreign-indexed", "indexed-settings"} {
		t.Run(kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 4)
			f.settings.SimplexStrategy = StrategyTableauRows
			c, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			var err, want error
			ok := true
			switch kind {
			case "basis-short":
				f.basis = f.basis[:1]
				ok, err = c.BasisHeadingIsCorrect()
				want = ErrBounds
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "basis-duplicate":
				f.basis = []uint32{0, 0}
				ok, err = c.BasisHeadingIsCorrect()
				if ok {
					t.Fatal("duplicate basis")
				}
			case "basis-outside":
				f.basis[1] = 99
				ok, err = c.BasisHeadingIsCorrect()
				want = ErrBounds
			case "nb-outside":
				f.nb[0] = 99
				ok, err = c.BasisHeadingIsCorrect()
				want = ErrBounds
			case "debug-reverse":
				f.nb = nil
				f.heading[3] = 99
				ok, err = c.BasisHeadingIsCorrect()
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "boxed-extent":
				f.types[0] = ColumnBoxed
				owner := f.low.Data
				f.low.Data = nil
				ok, err = c.BoundsForBoxedAreSetCorrectly()
				f.low.Data = owner
				want = ErrBounds
			case "unknown-bounds":
				f.types[0] = ColumnType(99)
				ok, err = c.BoundsForBoxedAreSetCorrectly()
				if !ok {
					t.Fatal("unused type reached")
				}
			case "enter-outside":
				err = c.ChangeBasis(99, 0)
				want = ErrBounds
			case "leave-outside":
				err = c.ChangeBasisUnconditionally(2, 99)
				want = ErrBounds
			case "enter-basic":
				err = c.ChangeBasis(0, 1)
				want = ErrBounds
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "signed-overflow":
				f.heading[2] = -1 << 31
				err = c.ChangeBasis(2, 0)
				want = ErrBounds
			case "late-nbasis":
				f.heading[2] = -99
				err = c.ChangeBasis(2, 0)
				want = ErrBounds
				if f.basis[0] != 2 || f.heading[0] != -99 {
					t.Fatal("late nonbasis prefix")
				}
			case "transpose-outside":
				err = c.TransposeBasis(2, 0)
				want = ErrBounds
			case "transpose-matrix":
				f.a.Rows = nil
				err = c.TransposeRowsTableau(0, 1)
				want = ErrBounds
				if !reflect.DeepEqual(f.basis, []uint32{1, 0}) {
					t.Fatal("matrix extent prefix")
				}
			case "restore-none":
				err = c.RestoreBasisChange(2, 0)
			case "restore-actual":
				c.StartTracingBasisChanges()
				h.must(c.ChangeBasis(2, 0))
				err = c.RestoreBasisChange(2, 0)
				if !reflect.DeepEqual(f.basis, []uint32{0, 1}) || len(c.Trace) != 0 {
					t.Fatal("basis restore")
				}
			case "restore-outside":
				err = c.RestoreBasisChange(99, 0)
				want = ErrBounds
			case "nil-indexed":
				err = c.SolveBdIndexed(2, nil)
				want = ErrBounds
			case "foreign-indexed":
				foreign := newLPPairHarness(t)
				out := NewIndexedRationalVector(foreign.m, 2)
				err = c.SolveBdIndexed(2, out)
				want = ErrExactVectorOwner
				h.must(out.Close())
			case "indexed-settings":
				c.Settings = nil
				err = c.SolveBdIndexed(2, c.Ed)
				want = ErrBounds
			}
			if err != want {
				t.Fatal(kind, ok, err, want)
			}
			h.must(c.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			// Every exported numerical/metadata entry rejects a closed core.
			for _, err := range []error{c.InitBasisHeadingAndNonBasicColumns(), c.AllocateBasisHeading(), c.FillCB(c.Y), c.SolveYB(c.Y), c.SolveBd(0), c.SolveBdIndexed(0, c.Ed), c.ChangeBasis(0, 1), c.TransposeBasis(0, 1), c.TransposeRowsTableau(0, 1), c.RestoreBasisChange(0, 1)} {
				if err != ErrExactVectorPending {
					t.Fatal("closed entry", err)
				}
			}
			if _, e := c.BasisHeadingIsCorrect(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if _, e := c.BoundsForBoxedAreSetCorrectly(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
		})
	}
}
