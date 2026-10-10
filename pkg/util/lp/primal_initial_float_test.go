package lp

import (
	"bytes"
	"errors"
	"math"
	"math/rand"
	"testing"
)

func TestLPFloatPrimalInitialIndependentPredicate(t *testing.T) {
	m := lpFloatPrimalCore(t, 2, 5)
	rng := rand.New(rand.NewSource(937999))
	for world := 0; world < 2000; world++ {
		perm := rng.Perm(5)
		*m.Basis = []uint32{uint32(perm[0]), uint32(perm[1])}
		want := true
		for j := 0; j < 5; j++ {
			typ := ColumnType(rng.Intn(6)) // Unknown type still has both bounds.
			x, hi := float64(rng.Intn(5)-1), float64(1+rng.Intn(3))
			(*m.ColumnTypes)[j], (*m.X)[j], (*m.UpperBounds)[j] = typ, x, hi
			(*m.LowBounds)[j] = 99 // Source deliberately checks zero, not this.
			basic := j == perm[0] || j == perm[1]
			if typ != ColumnFree && typ != ColumnUpperBound && x < 0 {
				want = false
			}
			if typ != ColumnFree && typ != ColumnLowBound && x > hi {
				want = false
			}
			if !basic && (typ == ColumnLowBound && x != 0 || typ == ColumnBoxed && x != 0 && x != hi) {
				want = false
			}
		}
		got, err := m.InitialXIsCorrect()
		if err != nil || got != want {
			t.Fatal(world, got, want, err)
		}
	}
}

func TestLPFloatPrimalInitialIEEEAndOutput(t *testing.T) {
	m := lpFloatPrimalCore(t, 0, 1)
	var out bytes.Buffer
	m.Settings.DebugOut = &out
	for _, tc := range []struct {
		typ   ColumnType
		x, hi float64
		want  bool
		text  string
	}{
		{ColumnLowBound, -1, 0, false, "low bound for variable 0 does not hold: this->m_x[0] = -1 is negative \n"},
		{ColumnUpperBound, 3, 2, false, "upper bound for 0 does not hold: 2>3\n"},
		{ColumnLowBound, 1, 0, false, "only low bound is set for 0 but low bound value 0 is not equal to 1\n"},
		{ColumnBoxed, 1, 2, false, ""},
		{ColumnBoxed, 2, 2, true, ""},
		{ColumnBoxed, 0, 2, true, ""},
		{ColumnFixed, 1, 2, true, ""}, // Fixed type has no nonbasic endpoint test.
		{ColumnUpperBound, -3, 2, true, ""},
		{ColumnFree, math.NaN(), 0, true, ""},
		{ColumnUpperBound, math.NaN(), 2, true, ""},
		{ColumnLowBound, math.NaN(), 2, false, "only low bound is set for 0 but low bound value 0 is not equal to nan\n"},
		{ColumnBoxed, math.NaN(), 2, false, ""},
		{ColumnLowBound, math.Copysign(0, -1), 0, true, ""},
		{ColumnLowBound, math.SmallestNonzeroFloat64, 0, false, "only low bound is set for 0 but low bound value 0 is not equal to 4.94066e-324\n"},
		{ColumnBoxed, math.Nextafter(2, 0), 2, false, ""},
		{ColumnBoxed, math.Nextafter(2, 3), 2, false, "upper bound for 0 does not hold: 2>2\n"},
	} {
		(*m.ColumnTypes)[0], (*m.X)[0], (*m.UpperBounds)[0] = tc.typ, tc.x, tc.hi
		out.Reset()
		got, err := m.InitialXIsCorrect()
		if err != nil || got != tc.want || out.String() != tc.text {
			t.Fatal(tc, got, err, out.String())
		}
	}
	wantErr := errors.New("initial output refused")
	m.Settings.DebugOut = &lpValidationWriter{allowed: 0, err: wantErr}
	for _, tc := range []struct {
		typ ColumnType
		x   float64
	}{{ColumnLowBound, -1}, {ColumnUpperBound, 3}, {ColumnLowBound, 1}} {
		(*m.ColumnTypes)[0], (*m.X)[0] = tc.typ, tc.x
		if ok, err := m.InitialXIsCorrect(); ok || err != wantErr {
			t.Fatal(ok, err)
		}
	}
}

func TestLPFloatPrimalZeroLowConstructorPublication(t *testing.T) {
	in := lpFloatCoreInput(t, 1, 2)
	in.Settings.SimplexStrategy = StrategyLU
	lpStaticMust(t, in.A.Set(0, 0, 1))
	*in.X, *in.B, *in.ColumnTypes = []float64{3, 0}, []float64{3}, []ColumnType{ColumnLowBound, ColumnLowBound}
	callerLow := []float64{99}
	in.LowBounds = &callerLow
	m, err := NewFloatPrimalCoreSolverWithZeroLowBounds(in)
	lpStaticMust(t, err)
	t.Cleanup(func() { m.Close() })
	if m.Status != StatusFeasible || m.Factorization == nil || m.ConvertedHarrisEpsilon != in.Settings.HarrisFeasibilityTolerance || len(m.Beta.Data) != 1 || len(*m.LowBounds) != 2 || (*m.LowBounds)[0] != 0 || callerLow[0] != 99 {
		t.Fatal("constructor state")
	}
	// Owner's slice header is the actual borrowed lower vector.
	m.lowBoundsDummy = []float64{7}
	if len(*m.LowBounds) != 1 || (*m.LowBounds)[0] != 7 {
		t.Fatal("lower header detached")
	}
	*m.LowBounds = []float64{0, 0}
	lpStaticMust(t, m.InitRun())
	if m.EpsilonOfReducedCost != 1e-7 {
		t.Fatal("pricing initialized by InitRun")
	}
	(*in.X)[1] = 1 // Nonbasic low value is not zero.
	partial, err := NewFloatPrimalCoreSolverWithZeroLowBounds(in)
	if partial != nil {
		defer partial.Close()
	}
	if sourceDebugEnabled {
		if err != ErrAssertion || partial == nil || len(*partial.LowBounds) != 0 || partial.Beta == nil || partial.Status != StatusFeasible {
			t.Fatal(partial, err)
		}
	} else if err != nil || len(*partial.LowBounds) != 2 {
		t.Fatal(err)
	}
	// Boxed source base assertion reads the not-yet-populated dummy vector.
	(*in.X)[1], (*in.ColumnTypes)[1], (*in.UpperBounds)[1] = 0, ColumnBoxed, 2
	partial, err = NewFloatPrimalCoreSolverWithZeroLowBounds(in)
	if partial != nil {
		defer partial.Close()
	}
	if sourceDebugEnabled {
		if err != ErrBounds || partial == nil || partial.Beta != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	(*in.ColumnTypes)[1] = ColumnLowBound
	*in.Basis = []uint32{9}
	partial, err = NewFloatPrimalCoreSolverWithZeroLowBounds(in)
	if partial == nil || err != ErrBounds || partial.Beta != nil {
		t.Fatal("base initialization prefix", partial, err)
	}
	partial.Close()
	in.A = nil
	if partial, err := NewFloatPrimalCoreSolverWithZeroLowBounds(in); partial != nil || err != ErrBounds {
		t.Fatal(partial, err)
	}
}

func TestLPFloatPrimalInitialMetadataPrefixes(t *testing.T) {
	m := lpFloatPrimalCore(t, 1, 1)
	*m.Basis = nil
	if _, err := m.InitialXIsCorrect(); err != ErrBounds {
		t.Fatal(err)
	}
	*m.Basis = []uint32{0}
	*m.ColumnTypes = nil
	if _, err := m.InitialXIsCorrect(); err != ErrBounds {
		t.Fatal(err)
	}
	*m.ColumnTypes = []ColumnType{ColumnLowBound}
	*m.X = nil
	if _, err := m.InitialXIsCorrect(); err != ErrBounds {
		t.Fatal(err)
	}
	*m.ColumnTypes = []ColumnType{ColumnUpperBound}
	if _, err := m.InitialXIsCorrect(); err != ErrBounds {
		t.Fatal(err)
	}
	*m.X = []float64{0}
	*m.UpperBounds = nil
	if _, err := m.InitialXIsCorrect(); err != ErrBounds {
		t.Fatal(err)
	}
	// Free columns do not read their X/upper slots in the source predicate.
	*m.ColumnTypes, *m.X = []ColumnType{ColumnFree}, nil
	if ok, err := m.InitialXIsCorrect(); !ok || err != nil {
		t.Fatal(ok, err)
	}
	// A failed source diagnostic precedes dummy resize and preserves base status.
	in := lpFloatCoreInput(t, 0, 1)
	(*in.ColumnTypes)[0], (*in.X)[0] = ColumnLowBound, -1
	wantErr := errors.New("constructor initial output refused")
	in.Settings.DebugOut = &lpValidationWriter{allowed: 0, err: wantErr}
	partial, err := NewFloatPrimalCoreSolverWithZeroLowBounds(in)
	if partial != nil {
		defer partial.Close()
	}
	if sourceDebugEnabled {
		if err != wantErr || len(*partial.LowBounds) != 0 {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
}
