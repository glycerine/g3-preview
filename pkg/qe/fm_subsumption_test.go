package qe

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func (f *fmFixture) rawConstraint(s *fmStore, lits []int32, xs []uint32, as []string, constant string, strict bool) *fmConstraint {
	f.t.Helper()
	var result *fmConstraint
	err := func() (err error) {
		scope, err := s.begin()
		if err != nil {
			return err
		}
		defer s.finish(scope, &err)
		n := f.r.manager.NumeralService()
		var numbers []*numeral.OwnedRational
		for _, text := range as {
			a, err := scope.number()
			if err != nil {
				return err
			}
			if err = n.SetString(a, text); err != nil {
				return err
			}
			numbers = append(numbers, a)
		}
		c, err := scope.number()
		if err != nil {
			return err
		}
		if err = n.SetString(c, constant); err != nil {
			return err
		}
		result, err = s.newConstraint(lits, xs, numbers, c, strict)
		return err
	}()
	fmMust(f.t, err)
	return result
}

func TestFMSubsumptionPolynomialLiteralAndStrictness(t *testing.T) {
	f := newFMFixture(t)
	s := f.store()
	x, err := s.variable(f.variable(0, f.real))
	fmMust(t, err)
	y, err := s.variable(f.variable(1, f.real))
	fmMust(t, err)
	p, err := s.boolean(f.constant("fm-p", f.boolean))
	fmMust(t, err)
	q, err := s.boolean(f.constant("fm-q", f.boolean))
	fmMust(t, err)
	base := f.rawConstraint(s, []int32{p}, []uint32{x, y}, []string{"2", "-1"}, "3", false)
	cases := []struct {
		name string
		c    *fmConstraint
		want bool
	}{
		{"self", base, false},
		{"permuted-polynomial", f.rawConstraint(s, []int32{q, p}, []uint32{y, x}, []string{"-1", "2"}, "4", false), true},
		{"equal", f.rawConstraint(s, []int32{p}, []uint32{x, y}, []string{"2", "-1"}, "3", false), true},
		{"stronger-bound", f.rawConstraint(s, []int32{p}, []uint32{x, y}, []string{"2", "-1"}, "2", false), false},
		{"strict-equal", f.rawConstraint(s, []int32{p}, []uint32{x, y}, []string{"2", "-1"}, "3", true), false},
		{"strict-looser", f.rawConstraint(s, []int32{p}, []uint32{x, y}, []string{"2", "-1"}, "4", true), true},
		{"opposite-literal", f.rawConstraint(s, []int32{-p}, []uint32{x, y}, []string{"2", "-1"}, "4", false), false},
		{"missing-literal", f.rawConstraint(s, nil, []uint32{x, y}, []string{"2", "-1"}, "4", false), false},
		{"coefficient-mismatch", f.rawConstraint(s, []int32{p}, []uint32{x, y}, []string{"3", "-1"}, "4", false), false},
		{"variable-mismatch", f.rawConstraint(s, []int32{p}, []uint32{x}, []string{"2"}, "4", false), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			yes, err := s.subsumes(base, tc.c)
			fmMust(t, err)
			if yes != tc.want {
				t.Fatalf("subsumes=%v, want %v", yes, tc.want)
			}
		})
	}
	strict := f.rawConstraint(s, []int32{p}, []uint32{x, y}, []string{"2", "-1"}, "3", true)
	yes, err := s.subsumes(strict, base)
	fmMust(t, err)
	if !yes {
		t.Fatal("strict bound did not subsume equal non-strict bound")
	}
}

func TestFMRegistrationBackwardSubsumptionAndConstantConstraints(t *testing.T) {
	f := newFMFixture(t)
	s := f.store()
	m := f.r.manager
	x, err := s.variable(f.variable(0, f.real))
	fmMust(t, err)
	weak := f.rawConstraint(s, nil, []uint32{x}, []string{"1"}, "5", false)
	strong := f.rawConstraint(s, nil, []uint32{x}, []string{"1"}, "3", false)
	for _, c := range []*fmConstraint{weak, strong} {
		indexed, err := s.register(c)
		fmMust(t, err)
		if !indexed {
			t.Fatal("constraint not indexed")
		}
	}
	fmMust(t, s.subsume())
	if !weak.dead || strong.dead || len(s.variables[x].uppers) != 1 || s.variables[x].uppers[0] != strong {
		t.Fatal("backward subsumption or index compaction")
	}
	if len(s.todo) != 0 {
		t.Fatal("pending subsumption not drained")
	}
	// Registered constants and forbidden-variable-only constraints are emitted
	// immediately, with the owning constraint retired.
	p, err := s.boolean(f.constant("fm-p", f.boolean))
	fmMust(t, err)
	clause := f.rawConstraint(s, []int32{p}, nil, nil, "-1", false)
	indexed, err := s.register(clause)
	fmMust(t, err)
	if indexed || !clause.closed || len(s.outputs) != 1 {
		t.Fatal("constant clause registration")
	}
	h, err := m.RefGet(s.outputs[0])
	fmMust(t, err)
	decl, err := m.GetDecl(h)
	fmMust(t, err)
	if decl.IsNull() {
		t.Fatal("missing literal output")
	}
	s.variables[x].forbidden = true
	forbidden := f.rawConstraint(s, nil, []uint32{x}, []string{"2"}, "8", false)
	indexed, err = s.register(forbidden)
	fmMust(t, err)
	if indexed || !forbidden.closed || len(s.outputs) != 2 {
		t.Fatal("forbidden row registration")
	}
	truth := f.rawConstraint(s, []int32{p}, nil, nil, "0", false)
	ref, err := s.expression(truth)
	fmMust(t, err)
	h, err = m.RefGet(ref)
	fmMust(t, err)
	if !m.IsTrue(h) {
		t.Fatal("true inequality did not absorb clause")
	}
	fmMust(t, m.DelRef(&ref))
	falseRow := f.rawConstraint(s, nil, nil, nil, "0", true)
	indexed, err = s.register(falseRow)
	fmMust(t, err)
	if indexed || !falseRow.closed || !s.inconsistent {
		t.Fatal("false constraint not detected")
	}
	// The raw zero-variable false-expression path is separate from registration.
	falseRow = f.rawConstraint(s, nil, nil, nil, "-1", false)
	ref, err = s.expression(falseRow)
	fmMust(t, err)
	h, err = m.RefGet(ref)
	fmMust(t, err)
	if !m.IsFalse(h) {
		t.Fatal("false inequality expression")
	}
	fmMust(t, m.DelRef(&ref))
}
