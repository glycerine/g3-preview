package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLARModelIndependentVariableDirectories(t *testing.T) {
	h := newLPPairHarness(t)
	r := rand.New(rand.NewPCG(74831, 39))
	for _, strategy := range []SimplexStrategy{StrategyUndecided, StrategyTableauRows, StrategyTableauCosts, StrategyLU} {
		s, err := NewLARSolver(h.m)
		h.must(err)
		s.Settings.SimplexStrategy = strategy
		if s.Core.RPrimal.Status != StatusUnknown || s.Core.DPrimal.Status != StatusUnknown || s.Bounds.Status.Value != StatusOptimal || s.Bounds.InfeasibleColumn.Value != -1 || s.Core.InfeasibleSumSign != 0 || !s.Core.IsEmpty() {
			t.Fatal("source constructors")
		}
		want := map[uint32]uint32{}
		var directory []uint32
		p := s.NewBoundPropagator()
		for step := 0; step < 300; step++ {
			ext := uint32(r.IntN(77)*13 + 1)
			index, exists := want[ext]
			if !exists {
				index = uint32(len(directory))
				want[ext] = index
				directory = append(directory, ext)
			}
			got, err := s.AddVar(ext)
			h.must(err)
			if got != index {
				t.Fatal(strategy, step, got, index)
			}
			if !reflect.DeepEqual(s.ExternalToColumns, want) || !reflect.DeepEqual(s.ColumnsToExternal, directory) {
				t.Fatal("external/internal directory")
			}
			n := uint32(len(directory))
			c := s.Core
			if c.RA.ColumnCount() != n || c.RA.RowCount() != 0 || c.RX.DataSize() != n || s.Bounds.Low.Size() != n || s.Bounds.Upper.Size() != n || c.RPrimal.InfSet.DataSize() != n || c.Costs.DataSize() != n || c.RPrimal.D.DataSize() != n || s.Bounds.Witnesses.Size() != n || s.Bounds.ChangedColumns.DataSize() != n {
				t.Fatal("source exact registration extents")
			}
			for j := uint32(0); j < n; j++ {
				if c.RHeading[j] != -int32(j)-1 || c.RNBasis[j] != j || s.Bounds.Types.values[j] != ColumnFree {
					t.Fatal("source append-order nonbasis headings")
				}
			}
			if s.UseLU() {
				if c.DA.ColumnCount() != n || uint32(len(c.DX)) != n || c.DHeading[index] != -int32(index)-1 || c.DNBasis[index] != index {
					t.Fatal("LU floating registration")
				}
			} else if c.DA.ColumnCount() != 0 || len(c.DX) != 0 {
				t.Fatal("non-LU grew floating matrix")
			}
			if c.RPrimal.PivotRow.DataSize() != 0 || c.RPrimal.Ed.DataSize() != 0 || len(c.DCosts) != 0 || c.DPrimal.InfSet.DataSize() != 0 {
				t.Fatal("registration eagerly initialized later solve state")
			}
			if !s.VarIsRegistered(index) || s.VarIsRegistered(n) || s.IsTerm(index) || s.ColumnName(index) != "v"+fmt.Sprint(ext) {
				t.Fatal("source column lookup")
			}
			kind, err := p.ColumnType(index)
			h.must(err)
			if kind != ColumnFree {
				t.Fatal("live propagator directory")
			}
			v, err := s.Value(index)
			h.must(err)
			h.check(v, big.NewRat(0, 1), big.NewRat(0, 1))
			witness, err := s.Bounds.Witnesses.At(index)
			h.must(err)
			if witness != NewLARBoundWitnesses(InvalidLARIndex) {
				t.Fatal("source invalid witnesses")
			}
		}
		s.SetPropagateBoundsOnPivotedRowsMode(true)
		if s.Core.RPrimal.PivotedRows != &s.RowsWithChangedBounds {
			t.Fatal("pivot row set alias")
		}
		s.SetPropagateBoundsOnPivotedRowsMode(false)
		if s.Core.RPrimal.PivotedRows != nil {
			t.Fatal("pivot row set disable")
		}
		if s.ColumnName(LARTermStartIndex+7) != "_t1000007" || s.ColumnName(999) != "_s999" {
			t.Fatal("source names")
		}
		h.must(p.Close())
		h.must(s.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARModelIndependentTermRowsAndBounds(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(84931, 729))
	for world := 0; world < 120; world++ {
		s, err := NewLARSolver(h.m)
		h.must(err)
		s.Settings.SimplexStrategy = []SimplexStrategy{StrategyUndecided, StrategyTableauRows, StrategyLU}[world%3]
		_, err = s.AddVar(91)
		h.must(err)
		_, err = s.AddVar(13)
		h.must(err)
		x0, x1 := [2]*big.Rat{big.NewRat(int64(rng.IntN(31)-15), 7), big.NewRat(int64(rng.IntN(19)-9), 13)}, [2]*big.Rat{big.NewRat(int64(rng.IntN(31)-15), 11), big.NewRat(int64(rng.IntN(19)-9), 17)}
		for i, x := range [][2]*big.Rat{x0, x1} {
			h.must(h.m.SetString(&s.Core.RX.Data[i].X, x[0].RatString()))
			h.must(h.m.SetString(&s.Core.RX.Data[i].Y, x[1].RatString()))
		}
		a, b := big.NewRat(int64(1+rng.IntN(9)), int64(1+rng.IntN(7))), big.NewRat(-int64(1+rng.IntN(9)), int64(1+rng.IntN(7)))
		inputs := NewSparseRationalVector(h.m)
		v := h.pair(a.RatString(), b.RatString())
		constant := h.pair("7/19", "0")
		h.must(inputs.Append(0, &v.X))
		h.must(inputs.Append(1, &v.Y))
		term, err := s.AddTerm(inputs.Data, &constant.X)
		h.must(err)
		if term != LARTermStartIndex || !s.IsTerm(term) || !s.VarIsRegistered(term) || s.IsTerm(term+1) || s.VarIsRegistered(term+1) || len(s.Terms) != 1 || len(s.OriginalTerms) != 1 {
			t.Fatal("term identity")
		}
		h.must(h.m.SetInt32(&constant.X, 100))
		h.must(h.m.SetInt32(&inputs.Data[0].Value, 101))
		lpSparseRationalCheck(t, h, &s.OriginalTerms[0].Value, big.NewRat(7, 19))
		// Mutating the working term must not change the original source term.
		h.must(s.Terms[0].SetValue(&constant.X))
		rhs := h.pair("23/29", "0")
		kind := LConstraintGE
		if world%2 != 0 {
			kind = LConstraintGT
		}
		ci, err := s.AddVarBound(term, kind, &rhs.X)
		h.must(err)
		if ci != 0 || s.NumberOfConstraints() != 1 {
			t.Fatal("constraint index")
		}
		column := s.ExternalToColumns[term]
		if column != 2 || s.Core.RA.RowCount() != 1 || s.Core.RBasis[0] != 2 || s.Core.RHeading[2] != 0 || s.Bounds.Witnesses.values[2].Row != 2 {
			t.Fatal("term row/basis/witness")
		}
		wantX := new(big.Rat).Add(new(big.Rat).Mul(a, x0[0]), new(big.Rat).Mul(b, x1[0]))
		wantY := new(big.Rat).Add(new(big.Rat).Mul(a, x0[1]), new(big.Rat).Mul(b, x1[1]))
		h.check(&s.Core.RX.Data[column], wantX, wantY)
		low, err := s.Bounds.Low.At(column)
		h.must(err)
		shifted := new(big.Rat).Sub(big.NewRat(23, 29), big.NewRat(7, 19))
		epsilon := big.NewRat(0, 1)
		if kind == LConstraintGT {
			epsilon.SetInt64(1)
		}
		h.check(low, shifted, epsilon)
		constraint, err := s.Constraint(ci)
		h.must(err)
		tc, ok := constraint.(*LARTermConstraint)
		if !ok || tc.Term != s.OriginalTerms[0] || tc.Kind != kind || constraint.Base() != &tc.LARBaseConstraint {
			t.Fatal("polymorphic term constraint")
		}
		lpSparseRationalCheck(t, h, &tc.RightSide, big.NewRat(23, 29))
		if s.Bounds.Witnesses.values[column].LowBoundWitness != ci || !s.Bounds.ChangedColumns.Contains(column) {
			t.Fatal("constraint bound evidence")
		}
		if s.UseLU() {
			if s.Core.DA.Rows[0][len(s.Core.DA.Rows[0])-1].Value != -1 || s.Core.DX[column] != 0 {
				t.Fatal("source floating basis sign and deferred X")
			}
		} else if s.Core.DA.ColumnCount() != 0 {
			t.Fatal("non-LU floating growth")
		}
		p := s.NewBoundPropagator()
		h.must(p.TryAddBound(&low.X, column, true, true, 7, kind == LConstraintGT))
		if p.FoundBounds[0].Column != term {
			t.Fatal("term propagation mapping")
		}
		// Source propagation restores the working term's current constant (100),
		// while constraint insertion subtracts the original constant (7/19).
		lpSparseRationalCheck(t, h, &p.FoundBounds[0].Bound, new(big.Rat).Add(shifted, big.NewRat(100, 1)))
		h.must(p.Close())
		h.must(inputs.Close())
		h.must(s.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARModelNestedTableauAndRowPropagation(t *testing.T) {
	h := newLPPairHarness(t)
	s, err := NewLARSolver(h.m)
	h.must(err)
	_, err = s.AddVar(31)
	h.must(err)
	_, err = s.AddVar(67)
	h.must(err)
	inputs := NewSparseRationalVector(h.m)
	v := h.pair("2", "3")
	constant := h.pair("5", "0")
	h.must(inputs.Append(0, &v.X))
	h.must(inputs.Append(1, &v.Y))
	term, err := s.AddTerm(inputs.Data, &constant.X)
	h.must(err)
	endpoints := h.pair("0", "9")
	for j := uint32(0); j < 2; j++ {
		_, err = s.AddVarBound(j, LConstraintGE, &endpoints.X)
		h.must(err)
		kind := LConstraintLE
		if j == 1 {
			kind = LConstraintLT
		}
		_, err = s.AddVarBound(j, kind, &endpoints.Y)
		h.must(err)
	}
	p := s.NewBoundPropagator()
	it, err := NewRationalRowIterator(s.Core.RA, 0)
	h.must(err)
	rhs := h.pair("0", "0")
	a, err := AnalyzeLARRow(it, rhs, 0, p)
	h.must(err)
	if len(p.FoundBounds) != 2 || p.FoundBounds[0].Column != term || !p.FoundBounds[0].IsLowBound || p.FoundBounds[0].Strict || p.FoundBounds[1].IsLowBound || !p.FoundBounds[1].Strict {
		t.Fatal("actual model row propagation evidence")
	}
	lpSparseRationalCheck(t, h, &p.FoundBounds[0].Bound, big.NewRat(5, 1))
	lpSparseRationalCheck(t, h, &p.FoundBounds[1].Bound, big.NewRat(50, 1))
	h.must(a.Close())
	h.must(it.Close())
	h.must(p.Close())
	h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("7/11", "2/13")))
	h.must(s.Core.RX.Data[1].Copy(h.m, h.pair("3/17", "-5/19")))
	// The old basic value must satisfy its existing row before adding the next.
	h.must(s.setBasicValueFromRow(0, 2))
	nested := NewSparseRationalVector(h.m)
	nv := h.pair("-4", "5")
	h.must(nested.Append(2, &nv.X))
	h.must(nested.Append(0, &nv.Y))
	_, err = s.AddTerm(nested.Data, &rhs.X)
	h.must(err)
	want := map[uint32]*big.Rat{0: big.NewRat(3, 1), 1: big.NewRat(12, 1), 3: big.NewRat(1, 1)}
	if len(s.Core.RA.Rows[1]) != 3 {
		t.Fatal("basic coefficient not eliminated")
	}
	for i := range s.Core.RA.Rows[1] {
		cell := &s.Core.RA.Rows[1][i]
		w, ok := want[cell.Column]
		if !ok {
			t.Fatal(cell.Column)
		}
		lpSparseRationalCheck(t, h, &cell.Value, w)
	}
	h.check(&s.Core.RX.Data[3], new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(-3, 1), big.NewRat(7, 11)), new(big.Rat).Mul(big.NewRat(12, 1), big.NewRat(3, 17))), new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(-3, 1), big.NewRat(2, 13)), new(big.Rat).Mul(big.NewRat(12, 1), big.NewRat(-5, 19))))
	h.must(inputs.Close())
	h.must(nested.Close())
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARModelPhysicalFailurePrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	beforeBound, afterBound := 0, 0
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"term-undecided", "term-tableau-zero", "term-tableau-value", "term-bound-mapped", "term-bound-unmapped", "var-bound"} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s, err := NewLARSolver(h.m)
					h.must(err)
					if op == "term-undecided" {
						s.Settings.SimplexStrategy = StrategyUndecided
					}
					if op == "term-bound-unmapped" {
						s.Settings.SimplexStrategy = StrategyLU
					}
					_, err = s.AddVar(31)
					h.must(err)
					_, err = s.AddVar(37)
					h.must(err)
					inputs := NewSparseRationalVector(h.m)
					large := strings.Repeat("1234567890", 100)
					v := h.pair(large+"/7", "-"+large+"/13")
					constant := h.pair(strings.Repeat("9876543210", 100)+"/17", "0")
					h.must(inputs.Append(0, &v.X))
					h.must(inputs.Append(1, &v.Y))
					if op == "term-tableau-value" {
						h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("123456789012345678901/19", "1/23")))
					}
					var term uint32
					if strings.HasPrefix(op, "term-bound") {
						term, err = s.AddTerm(inputs.Data, &constant.X)
						h.must(err)
					}
					h.budget.FailAfter(ordinal)
					if op == "var-bound" {
						_, err = s.AddVarBound(0, LConstraintGE, &v.X)
					} else if strings.HasPrefix(op, "term-bound") {
						_, err = s.AddVarBound(term, LConstraintGE, &v.X)
					} else {
						_, err = s.AddTerm(inputs.Data, &constant.X)
					}
					h.budget.FailAfter(-1)
					if err != nil {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						prefix := [8]int{len(s.ExternalToColumns), len(s.ColumnsToExternal), int(s.Core.RA.ColumnCount()), int(s.Core.RA.RowCount()), len(s.Constraints), len(s.Terms), len(s.OriginalTerms), int(s.Bounds.Witnesses.Size())}
						raw, division, comparison := h.m.RawInvocationState(), h.m.DivisionInvocationState(), h.m.ComparisonInvocationState()
						if _, e := s.AddVar(83); !errors.Is(e, ErrExactVectorPending) {
							t.Fatal("failed owner accepted variable registration", e)
						}
						if _, e := s.AddTerm(inputs.Data, &constant.X); !errors.Is(e, ErrExactVectorPending) {
							t.Fatal("failed owner accepted term registration", e)
						}
						if _, e := s.AddVarBound(0, LConstraintGE, &v.X); !errors.Is(e, ErrExactVectorPending) {
							t.Fatal("failed owner accepted bound registration", e)
						}
						if e := s.AdjustInitialStateForTableauRows(); !errors.Is(e, ErrExactVectorPending) {
							t.Fatal("failed owner accepted deferred registration", e)
						}
						after := [8]int{len(s.ExternalToColumns), len(s.ColumnsToExternal), int(s.Core.RA.ColumnCount()), int(s.Core.RA.RowCount()), len(s.Constraints), len(s.Terms), len(s.OriginalTerms), int(s.Bounds.Witnesses.Size())}
						if prefix != after || raw != h.m.RawInvocationState() || division != h.m.DivisionInvocationState() || comparison != h.m.ComparisonInvocationState() {
							t.Fatal("stopped entry point changed publication or invocation ownership")
						}
						if len(s.Constraints) == 1 && s.Bounds.Witnesses.values[len(s.Bounds.Witnesses.values)-1].LowBoundWitness == InvalidLARIndex {
							beforeBound++
						}
						if op == "term-bound-unmapped" && len(s.Constraints) == 0 && s.Bounds.Witnesses.Size() > 2 && s.Bounds.Types.values[2] != ColumnFree {
							afterBound++
						}
					} else {
						done = true
					}
					// Newly stopped children retain their own receipts; no outer call frame
					// wraps term/constraint construction, row filling or bound updates.
					if s.pendingTerm != nil && s.calls.comparison != 0 && s.calls.comparison == s.pendingTerm.calls.comparison {
						t.Fatal("term comparison adopted by parent")
					}
					h.must(s.Close())
					h.must(inputs.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					success = true
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal(direct, op, success, failures)
			}
		}
	}
	if beforeBound == 0 || afterBound == 0 {
		t.Fatal("missing actual constraint/bound publication prefixes", beforeBound, afterBound)
	}
}

func TestLARModelDeferredEmptyTermsAndSourceGuards(t *testing.T) {
	h := newLPPairHarness(t)
	if s, err := NewLARSolver(nil); s != nil || !errors.Is(err, ErrBounds) {
		t.Fatal("nil manager", err)
	}
	if c, err := NewLARCoreSolver(h.m, nil, nil); c != nil || !errors.Is(err, ErrBounds) {
		t.Fatal("nil settings", err)
	}
	s, err := NewLARSolver(h.m)
	h.must(err)
	s.Settings.SimplexStrategy = StrategyUndecided
	_, err = s.AddVar(17)
	h.must(err)
	constant := h.pair("7/13", "0")
	first, err := s.AddTerm(nil, &constant.X)
	h.must(err)
	inputs := NewSparseRationalVector(h.m)
	coefficient := h.pair("3/17", "0")
	h.must(inputs.Append(0, &coefficient.X))
	second, err := s.AddTerm(inputs.Data, &constant.X)
	h.must(err)
	if s.Core.RA.RowCount() != 0 || s.Core.RA.ColumnCount() != 1 || second != first+1 {
		t.Fatal("undecided terms registered eagerly")
	}
	s.Settings.SimplexStrategy = StrategyTableauRows
	s.Settings.BoundPropagation = false
	h.must(s.AdjustInitialStateForTableauRows())
	if s.Core.RA.RowCount() != 2 || s.Core.RA.ColumnCount() != 3 || s.RowsWithChangedBounds.Size() != 0 || s.RowsWithChangedBounds.DataSize() != 2 {
		t.Fatal("deferred registration/propagation-disabled profile")
	}
	if s.Core.RightSides.DataSize() != 3 {
		t.Fatal("source RHS grows to column count")
	}
	n := s.Core.RA.ColumnCount()
	h.must(s.AdjustInitialStateForTableauRows())
	if s.Core.RA.ColumnCount() != n {
		t.Fatal("registered term added twice")
	}
	_, err = s.AddVarBound(first, LConstraintLE, &constant.X)
	h.must(err)
	up, err := s.Bounds.Upper.At(s.ExternalToColumns[first])
	h.must(err)
	h.check(up, big.NewRat(0, 1), big.NewRat(0, 1))
	if _, err = s.Constraint(99); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	if _, err = s.Value(99); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	h.must(inputs.Close())
	h.must(s.Close())
	// A live empty LU constructor owns both actual factors; closing it leaves
	// caller-owned settings and names intact.
	settings, err := NewSettings()
	h.must(err)
	settings.SimplexStrategy = StrategyLU
	c, err := NewLARCoreSolver(h.m, settings, nil)
	h.must(err)
	if c.RPrimal.Factorization == nil || c.DPrimal.Factorization == nil || !c.NeedToPresolveWithDoubleSolver() || !c.IsTiny() || !c.IsEmpty() {
		t.Fatal("real dual-core construction")
	}
	for i := 0; i < 20; i++ {
		h.must(c.RA.AddColumn())
	}
	if c.IsTiny() || c.IsEmpty() {
		t.Fatal("source tiny column threshold")
	}
	h.must(c.Close())
	if settings.SimplexStrategy != StrategyLU {
		t.Fatal("borrowed settings destroyed")
	}
	invalid, err := NewLARSolver(h.m)
	h.must(err)
	_, err = invalid.AddVar(LARTermStartIndex)
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) {
			t.Fatal(err)
		}
	} else if !errors.Is(err, ErrBounds) {
		t.Fatal("source throw adaptation", err)
	}
	if invalid.Core.RA.ColumnCount() != 0 || invalid.Bounds.Witnesses.Size() != 0 || len(invalid.ExternalToColumns) != 0 {
		t.Fatal("invalid external index published state")
	}
	h.must(invalid.Close())
	if sourceDebugEnabled {
		broken, err := NewLARSolver(h.m)
		h.must(err)
		broken.ColumnsToExternal = append(broken.ColumnsToExternal, 91)
		_, err = broken.AddVar(71)
		if !errors.Is(err, ErrAssertion) || broken.ExternalToColumns[71] != 0 || len(broken.ColumnsToExternal) != 1 || broken.Bounds.Witnesses.Size() != 1 || broken.Core.RA.ColumnCount() != 0 {
			t.Fatal("map-before-directory assertion prefix", err)
		}
		h.must(broken.Close())
		// The deferred-row helper reaches map/witness publication before its
		// own LU-dimension assertion; it has no wrapper size assertion.
		mixed, err := NewLARSolver(h.m)
		h.must(err)
		_, err = mixed.AddVar(11)
		h.must(err)
		mixed.Settings.SimplexStrategy = StrategyLU
		coeff := NewSparseRationalVector(h.m)
		h.must(coeff.Append(0, &coefficient.X))
		term, err := mixed.AddTerm(coeff.Data, &constant.X)
		h.must(err)
		_, err = mixed.AddVarBound(term, LConstraintGE, &constant.X)
		if !errors.Is(err, ErrAssertion) || mixed.ExternalToColumns[term] != 1 || mixed.Bounds.Witnesses.Size() != 2 || mixed.Bounds.Types.Size() != 1 || len(mixed.Constraints) != 0 {
			t.Fatal("deferred row dimension assertion prefix", err)
		}
		h.must(mixed.Close())
		h.must(coeff.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARModelLeavesOlderComparisonOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			_, err = s.AddVar(31)
			h.must(err)
			initial := h.pair("1/13", "0")
			_, err = s.AddVarBound(0, LConstraintGE, &initial.X)
			h.must(err)
			v := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
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
				_, err = s.AddVarBound(0, LConstraintGE, &v.X)
				if err == nil || s.calls.comparison != 0 || s.Bounds.calls.comparison != 0 {
					t.Fatal("older comparison acquired by model/bounds", err)
				}
				if s.NumberOfConstraints() != 2 {
					t.Fatal("constraint should publish before the borrowed comparison failure")
				}
			} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("model destruction consumed older caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real separately owned comparison refusal")
	}
}
