package arith

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func boundParams(t *testing.T) (*params.ParamsContext, *params.ParamsRef) {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	sy, err := symbol.NewManager(symbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	equationMust(t, err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	equationMust(t, err)
	c, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b, PageEntries: 1,
		Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	equationMust(t, err)
	v, err := c.NewValue()
	equationMust(t, err)
	t.Cleanup(func() {
		equationMust(t, c.Del(&v))
		equationMust(t, c.Close())
		equationMust(t, q.Close())
		equationMust(t, sy.Close())
		if used := b.Used(); used != 0 {
			t.Fatalf("parameter owners retained %d bytes", used)
		}
	})
	return c, &v
}

func TestBoundEquationRetainedParameterReadsAndProgressCounts(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	c, v := boundParams(t)
	equationMust(t, c.SetUIntName(v, namePtr("bound_max_refinements"), 1))
	equationMust(t, c.SetDoubleName(v, namePtr("bound_threshold"), 0.5))
	equationMust(t, c.SetDoubleName(v, namePtr("bound_small_interval"), 0))
	equationMust(t, c.SetDoubleName(v, namePtr("strict2double"), 0.25))
	equationMust(t, s.UpdateParams(c, v))
	if s.maxRefinements != 1 || s.threshold != 0.5 || s.smallInterval != 0 || s.strict2double != 0.25 {
		t.Fatalf("parameter values = %d, %g, %g, %g", s.maxRefinements, s.threshold, s.smallInterval, s.strict2double)
	}
	equationMust(t, s.MkVar(0, false))
	equationMust(t, s.MkVar(1, false))
	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	equationMust(t, m.SetInt32(&value, 0))
	_, err = s.AssertLower(0, &value, false)
	equationMust(t, err)
	if s.relevantBound(0, 1, true) {
		t.Fatal("configured unbounded refinement limit was ignored")
	}
	equationMust(t, m.SetInt32(&value, 10))
	_, err = s.AssertUpper(0, &value, false)
	equationMust(t, err)
	if s.relevantBound(0, 1, true) {
		t.Fatal("configured narrow-interval gate was ignored")
	}
	equationMust(t, c.SetDoubleName(v, namePtr("bound_small_interval"), 20))
	equationMust(t, s.UpdateParams(c, v))
	if !s.relevantBound(0, 1, true) {
		t.Fatal("configured small interval did not override the refinement limit")
	}
	coeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(1)}
	ids := []uint32{0, 1}
	index, err := s.MkIntEquation(coeff, ids)
	equationMust(t, err)
	for i := range coeff {
		equationMust(t, m.Integers().Del(&coeff[i]))
	}
	propagations, falseAlarms, err := s.PropagationCounts()
	equationMust(t, err)
	if propagations == 0 || falseAlarms != 0 {
		t.Fatalf("progress counts after insertion = %d, %d", propagations, falseAlarms)
	}
	_, err = s.propagateExact(index, 1, true) // exact duplicate counts as a false alarm
	equationMust(t, err)
	_, falseAlarms, err = s.PropagationCounts()
	equationMust(t, err)
	if falseAlarms != 1 {
		t.Fatalf("false-alarm count = %d", falseAlarms)
	}
	hasLower, err := s.HasLower(1)
	equationMust(t, err)
	hasUpper, err := s.HasUpper(1)
	equationMust(t, err)
	if !hasLower || !hasUpper {
		t.Fatalf("retained tactic bound queries = %v, %v", hasLower, hasUpper)
	}
	inconsistent, err := s.Inconsistent()
	equationMust(t, err)
	if inconsistent {
		t.Fatal("nonconflicting bounds reported inconsistent")
	}
	equationMust(t, c.Reset(v))
	equationMust(t, s.UpdateParams(c, v))
	if s.maxRefinements != 16 || s.threshold != 0.05 || s.smallInterval != 128 || s.strict2double != 0.00001 {
		t.Fatal("default parameter values were not restored")
	}
}

func namePtr(s string) *string { return &s }
