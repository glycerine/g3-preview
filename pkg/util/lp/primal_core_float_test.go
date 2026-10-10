package lp

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func lpFloatPrimalCore(t *testing.T, rows, cols uint32) *FloatPrimalCoreSolver {
	t.Helper()
	in := lpFloatCoreInput(t, rows, cols)
	in.Settings.SimplexStrategy = StrategyLU
	for i := uint32(0); i < rows; i++ {
		if err := in.A.Set(i, i, 1); err != nil {
			t.Fatal(err)
		}
	}
	m, err := NewFloatPrimalCoreSolver(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	return m
}

func TestLPFloatPrimalCoreConstructionAndCostIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(881799))
	for world := 0; world < 150; world++ {
		in := lpFloatCoreInput(t, 10, 22)
		in.Settings.PrimalFeasibilityTolerance = 0
		in.Settings.UseBreakpointsInFeasibilitySearch = world%2 == 0
		order := rng.Perm(22)
		basic := make(map[uint32]bool)
		for i := 0; i < 10; i++ {
			j := uint32(order[i])
			(*in.Basis)[i] = j
			basic[j] = true
		}
		wantCost := make([]float64, 22)
		wantInf := make(map[uint32]bool)
		for j := 0; j < 22; j++ {
			typ := ColumnType(j % 5)
			low, upper := float64(rng.Intn(9)-4), float64(rng.Intn(9)+5)
			x := float64(rng.Intn(35) - 12)
			if typ == ColumnFixed {
				upper = low
			}
			(*in.ColumnTypes)[j] = typ
			(*in.LowBounds)[j] = low
			(*in.UpperBounds)[j] = upper
			(*in.X)[j] = x
			(*in.Costs)[j] = 100 + float64(j)
			if !basic[uint32(j)] {
				continue
			}
			if (typ == ColumnFixed || typ == ColumnBoxed || typ == ColumnUpperBound) && x > upper {
				wantCost[j] = 1
			} else if (typ == ColumnFixed || typ == ColumnBoxed || typ == ColumnLowBound) && x < low {
				wantCost[j] = -1
			}
			wantInf[uint32(j)] = wantCost[j] != 0
			if !in.Settings.UseBreakpointsInFeasibilitySearch {
				wantCost[j] = -wantCost[j]
			}
		}
		m, err := NewFloatPrimalCoreSolver(in)
		if err != nil {
			t.Fatal(world, err)
		}
		if m.Status != StatusUnknown || m.EpsilonOfReducedCost != 1e-7 || m.ConvertedHarrisEpsilon != in.Settings.HarrisFeasibilityTolerance || m.BlandModeThreshold != 1000 || len(m.Beta.Data) != 10 || m.BreakpointQueue.Size() != 0 || len(m.BreakpointQueue.heap) != 1 {
			t.Fatal("stage-one constructor")
		}
		if err := m.InitInfSet(); err != nil {
			t.Fatal(err)
		}
		var ascending []uint32
		for j := uint32(0); j < 22; j++ {
			if wantInf[j] {
				ascending = append(ascending, j)
			}
		}
		if len(ascending) != len(m.InfSet.Indices()) || (len(ascending) > 0 && !reflect.DeepEqual(ascending, m.InfSet.Indices())) {
			t.Fatal("ascending basic infeasibility order", world, m.InfSet.Indices(), ascending)
		}
		for j := uint32(0); j < 22; j++ {
			v, err := m.GetInfeasibilityCostForColumn(j)
			if err != nil || v != wantCost[j] || (*m.Costs)[j] != 100+float64(j) {
				t.Fatal("nonmutating cost", world, j, v, wantCost[j], err)
			}
		}
		if err := m.InitInfeasibilityCosts(); err != nil || !m.UsingInfeasCosts {
			t.Fatal(err)
		}
		for j, v := range *m.Costs {
			if v != wantCost[j] || m.InfSet.Contains(uint32(j)) != wantInf[uint32(j)] {
				t.Fatal("cost/membership", world, j, v, wantCost[j])
			}
		}
		// Recompute only the actual indexed basic rows; every other cost stays.
		before := append([]float64(nil), (*m.Costs)...)
		m.Ed.Index = []uint32{7, 0, 7}
		(*m.X)[(*m.Basis)[7]] = 1e3
		if err := m.InitInfeasibilityCostsForChangedBasisOnly(); err != nil {
			t.Fatal(err)
		}
		for j, v := range *m.Costs {
			if uint32(j) == (*m.Basis)[0] || uint32(j) == (*m.Basis)[7] {
				continue
			}
			if v != before[j] {
				t.Fatal("unchanged cost", world, j)
			}
		}
		m.Close()
	}
}

func TestLPFloatPrimalCoreInitRunActualCostLifecycle(t *testing.T) {
	m := lpFloatPrimalCore(t, 2, 3)
	if err := m.A.Set(0, 2, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.A.Set(1, 2, 1); err != nil {
		t.Fatal(err)
	}
	*m.ColumnTypes = []ColumnType{ColumnLowBound, ColumnFree, ColumnLowBound}
	*m.X = []float64{-1, 2, 0}
	*m.B = []float64{-1, 2}
	*m.Costs = []float64{6, 3, 2}
	m.Settings.RandomSeed(723)
	m.BasisSortCounter = 9
	m.TotalIterations = 12
	m.IterationsNoCostGrowth = 34
	m.ColumnNormUpdateCounter = 56
	if err := m.InitRun(); err != nil {
		t.Fatal(err)
	}
	if m.BasisSortCounter != 0 || m.TotalIterations != 0 || m.IterationsNoCostGrowth != 0 || m.ColumnNormUpdateCounter != 0 || !m.UsingInfeasCosts || !reflect.DeepEqual(*m.Costs, []float64{1, 0, 0}) {
		t.Fatal("run phase state", *m.Costs)
	}
	if !reflect.DeepEqual(m.CostsBackup, []float64{1, .5, 1.0 / 3}) || math.Abs(m.D[2]+2) > 1e-12 || len(m.BreakpointQueue.priorities) != 3 {
		t.Fatal("real reduced costs", m.CostsBackup, m.D)
	}
	state := uint32(723)
	for j, nz := range []float64{2, 2, 3} {
		state = state*214013 + 2531011
		want := nz + float64(((state>>16)&32767)%10000)/100000
		if m.ColumnNorms[j] != want {
			t.Fatal("literal random norm", j, m.ColumnNorms[j], want)
		}
	}
	(*m.X)[0] = 1
	if err := m.InitInfSet(); err != nil {
		t.Fatal(err)
	}
	if err := m.InitReducedCosts(); err != nil || m.UsingInfeasCosts || !reflect.DeepEqual(*m.Costs, m.CostsBackup) || math.Abs(m.D[2]+13.0/6) > 1e-12 {
		t.Fatal("return to normalized objective", err, *m.Costs, m.D)
	}
	// The feasible-only early return precedes cost backup, queue resize and LU.
	m = lpFloatPrimalCore(t, 1, 2)
	m.LookForFeasibleSolutionOnly = true
	m.UsingInfeasCosts = true
	m.ColumnNormUpdateCounter = 88
	m.EpsilonOfReducedCost = 9
	m.Close()
	if err := m.InitRun(); err != nil || !m.UsingInfeasCosts || m.ColumnNormUpdateCounter != 88 || m.EpsilonOfReducedCost != 9 || len(m.CostsBackup) != 0 || len(m.BreakpointQueue.priorities) != 0 {
		t.Fatal("feasible-only run early return", err)
	}
	if err := m.InitReducedCosts(); err != nil {
		t.Fatal("feasible-only reduced-cost early return", err)
	}
}

func TestLPFloatPrimalCoreNormalizationSourceIEEEAndRepeat(t *testing.T) {
	for _, tc := range []struct{ in, want []float64 }{
		{[]float64{4, -2, 1}, []float64{1, -.5, .25}},
		{[]float64{.5, -.25}, []float64{.5, -.25}},
		{[]float64{8, math.NaN(), 2}, []float64{4, math.NaN(), 1}},
		{[]float64{8, 2, math.NaN()}, []float64{math.NaN(), math.NaN(), math.NaN()}},
		{[]float64{math.Inf(1), 2}, []float64{math.NaN(), 0}},
	} {
		m := lpFloatPrimalCore(t, 0, 0)
		*m.Costs = append([]float64(nil), tc.in...)
		if err := m.BackupAndNormalizeCosts(); err != nil {
			t.Fatal(err)
		}
		for j, v := range *m.Costs {
			if v != tc.want[j] && !(math.IsNaN(v) && math.IsNaN(tc.want[j])) {
				t.Fatal(tc.in, j, v, tc.want[j])
			}
		}
		if len(m.CostsBackup) != len(tc.want) {
			t.Fatal("backup size")
		}
		err := m.BackupAndNormalizeCosts()
		if sourceDebugEnabled {
			if !errors.Is(err, ErrAssertion) {
				t.Fatal(err)
			}
		} else if err != nil || len(m.CostsBackup) != 2*len(tc.want) {
			t.Fatal(err)
		}
	}
	m := lpFloatPrimalCore(t, 0, 0)
	m.LookForFeasibleSolutionOnly = true
	m.CostsBackup = []float64{9}
	if err := m.BackupAndNormalizeCosts(); err != nil || m.CostsBackup[0] != 9 {
		t.Fatal(err)
	}
}

func TestLPFloatPrimalCoreCostPublicationAndGuards(t *testing.T) {
	if m, err := NewFloatPrimalCoreSolver(FloatCoreInputs{}); m != nil || !errors.Is(err, ErrBounds) {
		t.Fatal(m, err)
	}
	in := lpFloatCoreInput(t, 1, 2)
	(*in.Basis)[0] = 9
	if m, err := NewFloatPrimalCoreSolver(in); m == nil || !errors.Is(err, ErrBounds) || m.Beta != nil || m.Status != StatusFeasible {
		t.Fatal(m, err)
	}
	m := lpFloatPrimalCore(t, 1, 2)
	*m.Costs = []float64{9, 8}
	*m.ColumnTypes = []ColumnType{99, 99}
	if err := m.InfSet.Insert(1); err != nil {
		t.Fatal(err)
	}
	// Nonbasic publication does not inspect type, value or bound arrays.
	*m.X = nil
	*m.LowBounds = nil
	*m.UpperBounds = nil
	if err := m.InitInfeasibilityCostForColumn(1); err != nil || (*m.Costs)[1] != 0 || math.Signbit((*m.Costs)[1]) || m.InfSet.Contains(1) {
		t.Fatal(err, *m.Costs)
	}
	if v, err := m.GetInfeasibilityCostForColumn(1); err != nil || v != 0 || math.Signbit(v) {
		t.Fatal(v, err)
	}
	err := m.InitInfeasibilityCostForColumn(0)
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) || (*m.Costs)[0] != 9 {
			t.Fatal(err)
		}
	} else if err != nil || (*m.Costs)[0] != -9 || !m.InfSet.Contains(0) {
		t.Fatal("unknown source type retains old cost", err, *m.Costs)
	}
	if v, err := m.GetInfeasibilityCostForColumn(0); (sourceDebugEnabled && !errors.Is(err, ErrAssertion)) || (!sourceDebugEnabled && (err != nil || v != 0 || !math.Signbit(v))) {
		t.Fatal(v, err)
	}
	*m.ColumnTypes = []ColumnType{ColumnBoxed, ColumnFree}
	*m.X = []float64{100, 0}
	*m.UpperBounds = []float64{1, 1}
	if err := m.InitInfeasibilityCostForColumn(0); err != nil || (*m.Costs)[0] != -1 {
		t.Fatal("upper first avoids missing low", err)
	}
	(*m.X)[0] = 0
	if err := m.InitInfeasibilityCostForColumn(0); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	if _, err := m.GetInfeasibilityCostForColumn(9); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	if err := m.InitInfeasibilityCostForColumn(9); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	*m.ColumnTypes = []ColumnType{ColumnLowBound, ColumnFree}
	*m.LowBounds = []float64{0, 0}
	*m.X = []float64{-1, 0}
	m.Ed.Index = []uint32{0, 9}
	m.UsingInfeasCosts = false
	if err := m.InitInfeasibilityCostsForChangedBasisOnly(); !errors.Is(err, ErrBounds) || m.UsingInfeasCosts || (*m.Costs)[0] != 1 {
		t.Fatal("changed-basis prefix", err)
	}
	m.Ed.Index = []uint32{0}
	m.UsingInfeasCosts = true
	if err := m.InitInfeasibilityAfterUpdateXIfInf(1); err != nil || (*m.Costs)[1] != 0 || m.InfSet.Contains(1) {
		t.Fatal(err)
	}
	m.UsingInfeasCosts = false
	if err := m.InitInfeasibilityAfterUpdateXIfInf(99); err != nil {
		t.Fatal("inactive early return", err)
	}
	*m.Heading = nil
	if err := m.InitInfSet(); !errors.Is(err, ErrBounds) || !m.InfSet.Empty() {
		t.Fatal("clear precedes scan guard", err)
	}
}

func TestLPFloatPrimalCoreInitializationFailurePrefixes(t *testing.T) {
	m := lpFloatPrimalCore(t, 1, 2)
	*m.ColumnTypes = []ColumnType{ColumnLowBound, ColumnFree}
	*m.Costs = []float64{7, 8}
	*m.LowBounds = nil
	if err := m.InitInfeasibilityCosts(); !errors.Is(err, ErrBounds) || (*m.Costs)[1] != 0 || m.UsingInfeasCosts {
		t.Fatal("descending partial initialization", err, *m.Costs)
	}
	m.Ed.Index = []uint32{0}
	m.UsingInfeasCosts = true
	if err := m.InitInfeasibilityAfterUpdateXIfInf(1); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	*m.LowBounds = []float64{0, 0}
	if err := m.InitInfeasibilityAfterUpdateXIfInf(99); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	*m.Costs = nil
	if err := m.InitInfeasibilityCostForColumn(0); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	if err := m.InitInfeasibilityCostForColumn(1); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	*m.ColumnTypes = nil
	if _, err := m.GetInfeasibilityCostForColumn(0); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	*m.ColumnTypes = []ColumnType{ColumnUpperBound, ColumnFree}
	*m.UpperBounds = nil
	if _, err := m.GetInfeasibilityCostForColumn(0); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	*m.ColumnTypes = []ColumnType{ColumnLowBound, ColumnFree}
	*m.X = nil
	if err := m.InitInfeasibilityCostForColumn(0); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	if err := m.InitInfSet(); !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	err := m.InitInfeasibilityCosts()
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) {
			t.Fatal(err)
		}
	} else if !errors.Is(err, ErrBounds) {
		t.Fatal(err)
	}
	// Run counters publish before the first feasibility access fails.
	m.TotalIterations = 9
	m.IterationsNoCostGrowth = 8
	m.BasisSortCounter = 7
	if err := m.InitRun(); !errors.Is(err, ErrBounds) || m.TotalIterations != 0 || m.IterationsNoCostGrowth != 0 || m.BasisSortCounter != 0 {
		t.Fatal(err)
	}
	m = lpFloatPrimalCore(t, 1, 2)
	m.Settings.SimplexStrategy = StrategyTableauRows
	err = m.InitReducedCosts()
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	m.Settings.SimplexStrategy = StrategyLU
	(*m.ColumnTypes)[0] = ColumnLowBound
	*m.LowBounds = nil
	if err := m.InfSet.Insert(0); err != nil {
		t.Fatal(err)
	}
	if err := m.InitReducedCosts(); !errors.Is(err, ErrBounds) || m.UsingInfeasCosts {
		t.Fatal(err)
	}
	m = lpFloatPrimalCore(t, 1, 2)
	m.CostsBackup = []float64{9}
	*m.Costs = []float64{4, 2}
	err = m.InitRun()
	if sourceDebugEnabled {
		if !errors.Is(err, ErrAssertion) || (*m.Costs)[0] != 4 {
			t.Fatal(err)
		}
	} else if err != nil || !reflect.DeepEqual(m.CostsBackup, []float64{9, 1, .5}) {
		t.Fatal(err, m.CostsBackup)
	}
	m = lpFloatPrimalCore(t, 1, 2)
	m.ColumnNormUpdateCounter = 88
	m.EpsilonOfReducedCost = 9
	m.Close()
	if err := m.InitRun(); !errors.Is(err, ErrBounds) || m.EpsilonOfReducedCost != 1e-7 || len(m.BreakpointQueue.priorities) != 2 || m.ColumnNormUpdateCounter != 88 {
		t.Fatal("cost phase before missing LU", err)
	}
	m = lpFloatPrimalCore(t, 1, 2)
	m.ColumnNormUpdateCounter = 88
	m.ColumnNorms = nil
	if err := m.InitRun(); !errors.Is(err, ErrBounds) || m.ColumnNormUpdateCounter != 0 {
		t.Fatal("norm counter before short norm output", err)
	}
}

func TestLPFloatPrimalCoreCostIEEEAndMembershipTransition(t *testing.T) {
	m := lpFloatPrimalCore(t, 1, 2)
	(*m.ColumnTypes)[0] = ColumnBoxed
	(*m.LowBounds)[0], (*m.UpperBounds)[0] = -2, 3
	for _, x := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0} {
		(*m.X)[0] = x
		if err := m.InitInfeasibilityCostForColumn(0); err != nil {
			t.Fatal(err)
		}
		want := 0.
		if math.IsInf(x, 1) {
			want = -1
		}
		if math.IsInf(x, -1) {
			want = 1
		}
		if (*m.Costs)[0] != want || m.InfSet.Contains(0) != (want != 0) {
			t.Fatal(x, *m.Costs, m.InfSet.Indices())
		}
		if want == 0 && !math.Signbit((*m.Costs)[0]) {
			t.Fatal("basic zero is negated after membership update")
		}
	}
	m.Settings.BackupCosts = false
	m.LookForFeasibleSolutionOnly = true
	(*m.ColumnTypes)[0] = ColumnLowBound
	(*m.LowBounds)[0] = 0
	(*m.X)[0] = -1
	if err := m.InitRun(); err != nil || !m.UsingInfeasCosts || len(m.CostsBackup) != 0 {
		t.Fatal("infeasible-only run", err)
	}
}
