package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

type lpPairCoreFixture struct {
	h             *lpPairHarness
	a             *RationalStaticMatrix
	b, x, low, up *IndexedPairVector
	costs         *IndexedRationalVector
	basis, nb     []uint32
	heading       []int32
	types         []ColumnType
	settings      *Settings
}

func lpNewPairCoreFixture(t *testing.T, h *lpPairHarness, rows, cols uint32) *lpPairCoreFixture {
	t.Helper()
	s, err := NewSettings()
	h.must(err)
	s.DebugOut, s.MessageOut = nil, nil
	f := &lpPairCoreFixture{h: h, a: NewRationalStaticMatrix(h.m, rows, cols), b: NewIndexedPairVector(h.m, rows), x: NewIndexedPairVector(h.m, cols), low: NewIndexedPairVector(h.m, cols), up: NewIndexedPairVector(h.m, cols), costs: NewIndexedRationalVector(h.m, cols), basis: make([]uint32, rows), nb: []uint32{91}, heading: []int32{92}, types: make([]ColumnType, cols), settings: s}
	for i := range f.basis {
		f.basis[i] = uint32(i)
	}
	return f
}
func (f *lpPairCoreFixture) input() PairCoreInputs {
	return PairCoreInputs{A: f.a, B: &f.b.Data, X: &f.x.Data, LowBounds: &f.low.Data, UpperBounds: &f.up.Data, Costs: &f.costs.Data, Basis: &f.basis, NBasis: &f.nb, Heading: &f.heading, ColumnTypes: &f.types, Settings: f.settings}
}
func (f *lpPairCoreFixture) close() {
	f.h.must(f.costs.Close())
	for _, v := range []*IndexedPairVector{f.up, f.low, f.x, f.b} {
		f.h.must(v.Close())
	}
	f.h.must(f.a.Close())
}

func TestLPPairCoreIndependentBasisAndExactLUSolves(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(82719, 417))
	value := h.pair("0", "0")
	for world := 0; world < 100; world++ {
		rows := 1 + rng.IntN(5)
		cols := rows + 2
		f := lpNewPairCoreFixture(t, h, uint32(rows), uint32(cols))
		permutation := rng.Perm(cols)
		for i := range f.basis {
			f.basis[i] = uint32(permutation[i])
		}
		b := make([][]*big.Rat, rows)
		d, y := make([]*big.Rat, rows), make([]*big.Rat, rows)
		for i := 0; i < rows; i++ {
			b[i] = make([]*big.Rat, rows)
			d[i] = big.NewRat(int64(rng.IntN(31)-15), int64(1+rng.IntN(19)))
			y[i] = big.NewRat(int64(rng.IntN(31)-15), int64(1+rng.IntN(19)))
			for j := 0; j < rows; j++ {
				x := big.NewRat(0, 1)
				if j <= i {
					x.SetFrac64(int64(rng.IntN(15)-7), int64(1+rng.IntN(11)))
				}
				if i == j {
					x.SetFrac64(int64(i+2), int64(1+rng.IntN(7)))
				}
				b[i][j] = x
				h.must(h.m.SetString(&value.X, x.RatString()))
				h.must(f.a.Set(uint32(i), f.basis[j], &value.X))
			}
		}
		enter := uint32(permutation[rows])
		for i := 0; i < rows; i++ {
			a := new(big.Rat)
			for j := 0; j < rows; j++ {
				a.Add(a, new(big.Rat).Mul(b[i][j], d[j]))
			}
			h.must(h.m.SetString(&value.X, a.RatString()))
			h.must(f.a.Set(uint32(i), enter, &value.X))
		}
		for j := 0; j < rows; j++ {
			cost := new(big.Rat)
			for i := 0; i < rows; i++ {
				cost.Add(cost, new(big.Rat).Mul(y[i], b[i][j]))
			}
			h.must(h.m.SetString(&f.costs.Data[f.basis[j]], cost.RatString()))
		}
		if world%2 == 0 {
			f.settings.SimplexStrategy = StrategyLU
		}
		core, err := NewPairPrimalCoreSolver(f.input())
		h.must(err)
		if core.Status != StatusUnknown || core.TotalIterations != 0 || core.IterationsNoCostGrowth != 0 || core.UsingInfeasCosts || core.Tracing || core.PivotedRows != nil || core.LookForFeasibleSolutionOnly || core.BlandModeThreshold != 1000 {
			t.Fatal("source constructor defaults")
		}
		lpSparseRationalCheck(t, h, &core.EpsilonOfReducedCost, big.NewRat(1, 10000000))
		lpSparseRationalCheck(t, h, &core.ConvertedHarrisEpsilon, big.NewRat(0, 1))
		if len(core.Y.Data) != rows || len(core.CopyXB.Data) != rows || len(core.D.Data) != cols || len(core.PivotRow.Data) != cols || len(core.Beta.Data) != rows {
			t.Fatal("source scratch extents")
		}
		wantHeading := make([]int32, cols)
		var wantNB []uint32
		for j := cols - 1; j >= 0; j-- {
			pos := -1
			for i, c := range f.basis {
				if c == uint32(j) {
					pos = i
				}
			}
			if pos >= 0 {
				wantHeading[j] = int32(pos)
			} else {
				wantHeading[j] = -int32(len(wantNB)) - 1
				wantNB = append(wantNB, uint32(j))
			}
		}
		if !reflect.DeepEqual(wantHeading, f.heading) || !reflect.DeepEqual(wantNB, f.nb) {
			t.Fatal("inverse basis metadata", f.heading, wantHeading, f.nb, wantNB)
		}
		ok, err := core.BasisHeadingIsCorrect()
		h.must(err)
		if !ok {
			t.Fatal("basis metadata rejected")
		}
		if f.settings.UseLU() {
			out := NewIndexedRationalVector(h.m, uint32(rows))
			h.must(core.SolveYB(out))
			for i, want := range y {
				lpSparseRationalCheck(t, h, &out.Data[i], want)
			}
			core.ColumnsNZ = make([]uint32, cols)
			h.must(core.SolveBd(enter))
			nonzeros := uint32(0)
			for i, want := range d {
				lpSparseRationalCheck(t, h, &core.Ed.Data[i], want)
				if want.Sign() != 0 {
					nonzeros++
				}
			}
			if core.ColumnsNZ[enter] != nonzeros {
				t.Fatal("source nonzero estimate")
			}
			h.must(core.SolveBdIndexed(enter, out))
			for i, want := range d {
				lpSparseRationalCheck(t, h, &out.Data[i], want)
			}
			h.must(out.Close())
		} else if core.Factorization != nil {
			t.Fatal("tableau constructed LU")
		}
		// Header borrows, rather than snapshots, stay current after owner growth.
		h.must(f.costs.resizeData(uint32(cols + 1)))
		if len(*core.Costs) != cols+1 {
			t.Fatal("cost slice header was copied")
		}
		h.must(core.Close())
		f.close()
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairCoreIndependentLexicographicFeasibility(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 2, 5)
	f.basis = []uint32{3, 1}
	one := h.pair("1", "0")
	h.must(f.a.Set(0, 3, &one.X))
	h.must(f.a.Set(1, 1, &one.X))
	core, err := NewPairPrimalCoreSolver(f.input())
	h.must(err)
	rng := rand.New(rand.NewPCG(31827, 19))
	lex := func(x, y [2]*big.Rat) int {
		if c := x[0].Cmp(y[0]); c != 0 {
			return c
		}
		return x[1].Cmp(y[1])
	}
	for world := 0; world < 320; world++ {
		wantInf := map[uint32]bool{}
		for j := 0; j < 5; j++ {
			typ := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}[(world+j)%5]
			f.types[j] = typ
			lo := [2]*big.Rat{big.NewRat(int64(rng.IntN(11)-5), int64(1+rng.IntN(7))), big.NewRat(int64(rng.IntN(7)-3), int64(1+rng.IntN(5)))}
			up := [2]*big.Rat{new(big.Rat).Add(lo[0], big.NewRat(int64(rng.IntN(3)), 3)), big.NewRat(int64(rng.IntN(7)-3), int64(1+rng.IntN(5)))}
			if typ == ColumnFixed {
				up = [2]*big.Rat{new(big.Rat).Set(lo[0]), new(big.Rat).Set(lo[1])}
			}
			if lex(lo, up) > 0 {
				lo, up = up, lo
			}
			x := [2]*big.Rat{new(big.Rat).Set(lo[0]), big.NewRat(int64(rng.IntN(9)-4), int64(1+rng.IntN(5)))}
			if world%3 == 0 {
				x[0].Add(x[0], big.NewRat(int64(rng.IntN(5)-2), 7))
			}
			for k, slots := range [][]*numeral.Rational{{&f.low.Data[j].X, &f.low.Data[j].Y}, {&f.up.Data[j].X, &f.up.Data[j].Y}, {&f.x.Data[j].X, &f.x.Data[j].Y}} {
				values := [][2]*big.Rat{lo, up, x}[k]
				for i, slot := range slots {
					h.must(h.m.SetString(slot, values[i].RatString()))
				}
			}
			below, above := lex(x, lo) < 0, lex(x, up) > 0
			feasible := !(below && (typ == ColumnLowBound || typ == ColumnBoxed || typ == ColumnFixed) || above && (typ == ColumnUpperBound || typ == ColumnBoxed || typ == ColumnFixed))
			got, err := core.ColumnIsFeasible(uint32(j))
			h.must(err)
			if got != feasible {
				t.Fatal(world, j, typ, "feasibility")
			}
			sign := int32(0)
			if below && (typ == ColumnLowBound || typ == ColumnBoxed || typ == ColumnFixed) {
				sign = -1
			} else if above && (typ == ColumnUpperBound || typ == ColumnBoxed || typ == ColumnFixed) {
				sign = 1
			}
			gotSign, err := core.GetColumnOutOfBoundsDeltaSign(uint32(j))
			h.must(err)
			if gotSign != sign {
				t.Fatal(world, j, "direction", gotSign, sign)
			}
			at, err := core.AtBound(&f.x.Data[j], &f.low.Data[j])
			h.must(err)
			if at != (lex(x, lo) == 0) {
				t.Fatal("exact bound equality")
			}
			if !feasible && (j == 1 || j == 3) {
				wantInf[uint32(j)] = true
			}
		}
		ok, err := core.BoundsForBoxedAreSetCorrectly()
		h.must(err)
		if !ok {
			t.Fatal("ordered exact interval rejected")
		}
		h.must(core.InitInfSet())
		if core.CurrentXIsFeasible() != (len(wantInf) == 0) || core.CurrentXIsInfeasible() != (len(wantInf) > 0) {
			t.Fatal("infeasibility flags")
		}
		for j := uint32(0); j < 5; j++ {
			if core.InfSet.Contains(j) != wantInf[j] {
				t.Fatal(world, j, "basic infeasibility membership")
			}
		}
	}
	h.must(core.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairCorePhysicalFailurePrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"construct", "fill", "compare", "solve-indexed", "solve-internal", "solve-y", "init-inf", "delta"} {
			succeeded, failures := false, 0
			for ordinal := int64(0); ordinal < 800; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 2, 3)
					f.settings.SimplexStrategy = StrategyLU
					small := h.pair("100000000000000000000000000000001/7", "200000000000000000000000000000003/13")
					h.must(f.a.Set(0, 0, &small.X))
					h.must(f.a.Set(0, 1, &small.Y))
					h.must(f.a.Set(1, 1, &small.X))
					h.must(f.a.Set(0, 2, &small.Y))
					h.must(f.a.Set(1, 2, &small.X))
					large := strings.Repeat("1234567890", 100)
					v := h.pair(large+"/17", large+"/19")
					comparison := h.pair(large+"/19", "0")
					h.must(h.m.Copy(&f.costs.Data[0], &v.X))
					h.must(h.m.Copy(&f.costs.Data[1], &v.Y))
					out := NewIndexedRationalVector(h.m, 2)
					var core *PairPrimalCoreSolver
					var err error
					if op == "construct" {
						h.budget.FailAfter(ordinal)
						core, err = NewPairPrimalCoreSolver(f.input())
					} else {
						core, err = NewPairPrimalCoreSolver(f.input())
						h.must(err)
						core.ColumnsNZ = make([]uint32, 3)
						if op == "init-inf" || op == "delta" {
							f.types[0], f.types[1] = ColumnLowBound, ColumnLowBound
							h.must(f.low.Data[0].Copy(h.m, v))
							h.must(f.x.Data[0].Copy(h.m, comparison))
							h.must(f.low.Data[1].Copy(h.m, comparison))
							h.must(f.x.Data[1].Copy(h.m, v))
						}
						h.budget.FailAfter(ordinal)
						switch op {
						case "fill":
							err = core.FillCB(out)
						case "compare":
							_, err = core.BelowBound(v, comparison)
						case "solve-indexed":
							err = core.SolveBdIndexed(2, out)
						case "solve-internal":
							err = core.SolveBd(2)
						case "solve-y":
							err = core.SolveYB(out)
						case "init-inf":
							err = core.InitInfSet()
						case "delta":
							_, err = core.GetColumnOutOfBoundsDeltaSign(0)
						}
					}
					h.budget.FailAfter(-1)
					if err != nil {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if core.stopped || core.PairCoreSolverBase.calls.pending() {
							if !errors.Is(core.InitInfSet(), ErrExactVectorPending) {
								t.Fatal("stopped core accepted mutation")
							}
						}
					} else {
						done = true
					}
					if core.Factorization != nil && core.PairCoreSolverBase.calls.pending() && op != "compare" && op != "fill" && op != "solve-y" && op != "init-inf" && op != "delta" {
						t.Fatal("core acquired separately owned LU call")
					}
					h.must(core.Close())
					h.must(out.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					succeeded = true
					break
				}
			}
			if !succeeded || failures == 0 {
				t.Fatal(direct, op, succeeded, failures)
			}
		}
	}
}

func TestLPPairCoreSourceAssertionsAndPublication(t *testing.T) {
	h := newLPPairHarness(t)
	if m, err := NewPairPrimalCoreSolver(PairCoreInputs{}); m != nil || !errors.Is(err, ErrBounds) {
		t.Fatal("missing source references", err)
	}
	empty := lpNewPairCoreFixture(t, h, 0, 0)
	ec, err := NewPairPrimalCoreSolver(empty.input())
	h.must(err)
	h.must(ec.InitInfSet())
	h.must(ec.Close())
	empty.close()
	f := lpNewPairCoreFixture(t, h, 2, 3)
	one := h.pair("1", "0")
	h.must(f.a.Set(0, 0, &one.X))
	h.must(f.a.Set(1, 1, &one.X))
	h.must(f.a.Set(0, 2, &one.X))
	h.must(f.a.Set(1, 2, &one.X))
	core, err := NewPairPrimalCoreSolver(f.input())
	h.must(err)
	out := NewIndexedRationalVector(h.m, 2)
	h.must(h.m.SetInt32(&f.costs.Data[0], 7))
	h.must(h.m.SetInt32(&f.costs.Data[1], 11))
	if err = core.SolveYB(out); !errors.Is(err, ErrBounds) {
		t.Fatal("missing factorization", err)
	}
	lpSparseRationalCheck(t, h, &out.Data[0], big.NewRat(7, 1))
	lpSparseRationalCheck(t, h, &out.Data[1], big.NewRat(11, 1))
	// Indexed solve performs its source tableau assertion before lazy LU creation.
	err = core.SolveBdIndexed(2, out)
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) || core.Factorization != nil {
			t.Fatal(err)
		}
	} else {
		h.must(err)
		lpSparseRationalCheck(t, h, &out.Data[0], big.NewRat(1, 1))
		lpSparseRationalCheck(t, h, &out.Data[1], big.NewRat(1, 1))
	}
	foreign := lpNewDirectSparseHarness(t)
	fo := NewIndexedRationalVector(foreign.m, 2)
	if err = core.FillCB(fo); !errors.Is(err, ErrExactVectorOwner) {
		t.Fatal(err)
	}
	if err = core.SolveBdIndexed(2, fo); !errors.Is(err, ErrExactVectorOwner) {
		t.Fatal(err)
	}
	foreign.must(fo.Close())
	// The first copied basic cost survives a later missing-cost guard.
	f.costs.Data = f.costs.Data[:1]
	if err = core.FillCB(out); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	lpSparseRationalCheck(t, h, &out.Data[0], big.NewRat(7, 1))
	f.costs.Data = f.costs.Data[:3]
	f.basis = []uint32{0, 0}
	ok, err := core.BasisHeadingIsCorrect()
	h.must(err)
	if ok {
		t.Fatal("duplicate basis accepted")
	}
	f.basis = []uint32{0, 1}
	f.nb = []uint32{2, 2}
	ok, err = core.BasisHeadingIsCorrect()
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) {
			t.Fatal(err)
		}
	} else {
		h.must(err)
		if ok {
			t.Fatal("duplicate nonbasis accepted")
		}
	}
	f.nb = []uint32{2}
	f.heading[0] = 1
	ok, err = core.BasisHeadingIsCorrect()
	h.must(err)
	if ok {
		t.Fatal("incorrect basic inverse accepted")
	}
	h.must(core.InitBasisHeadingAndNonBasicColumns())
	f.heading[2] = -7
	ok, err = core.BasisHeadingIsCorrect()
	h.must(err)
	if ok {
		t.Fatal("incorrect nonbasic inverse accepted")
	}
	h.must(core.InitBasisHeadingAndNonBasicColumns())
	// Unknown column kinds retain the source assertion/release false profile.
	f.types[0] = ColumnType(99)
	ok, err = core.ColumnIsFeasible(0)
	if ok || sourceDebugEnabled && !errors.Is(err, ErrAssertion) || !sourceDebugEnabled && err != nil {
		t.Fatal(ok, err)
	}
	_, err = core.GetColumnOutOfBoundsDeltaSign(0)
	if sourceDebugEnabled && !errors.Is(err, ErrAssertion) || !sourceDebugEnabled && err != nil {
		t.Fatal(err)
	}
	f.types[0] = ColumnFree
	h.must(core.Close())
	h.must(out.Close())
	f.close()
	boxed := lpNewPairCoreFixture(t, h, 0, 1)
	boxed.types[0] = ColumnBoxed
	h.must(h.m.SetInt32(&boxed.low.Data[0].X, 2))
	h.must(h.m.SetInt32(&boxed.up.Data[0].X, 1))
	bad, err := NewPairPrimalCoreSolver(boxed.input())
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) || bad.Status != StatusFeasible || bad.Beta != nil {
			t.Fatal("boxed assertion construction prefix", err)
		}
	} else {
		h.must(err)
	}
	h.must(bad.Close())
	boxed.close()
	lpExactVectorNamedCensus(t, h, nil)
	lpExactVectorNamedCensus(t, foreign, nil)
}

func TestLPPairCoreLeavesOlderComparisonOwned(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 0, 0)
			core, err := NewPairPrimalCoreSolver(f.input())
			h.must(err)
			v := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			bound := h.pair("3/17", "5/19")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_, err = core.BelowBound(v, bound)
				if !errors.Is(err, numeral.ErrRationalRawArithmeticPending) || core.PairCoreSolverBase.calls.comparison != 0 {
					t.Fatal("older comparison acquired by core", err)
				}
			} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			h.must(core.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("core consumed older comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual separately owned comparison refusal")
	}
}
