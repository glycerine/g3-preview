package qe

import (
	"fmt"
	"reflect"
	"testing"
)

func TestFMCandidateStableOrderCostsAndIntegerBounds(t *testing.T) {
	f := newFMFixture(t)
	s := f.store()
	for i := uint32(0); i < 7; i++ {
		sort := f.integer
		if i == 1 || i == 3 || i == 4 {
			sort = f.real
		}
		_, err := s.variable(f.variable(i, sort))
		fmMust(t, err)
	}
	for i := range s.variables {
		if i != 2 && i != 3 {
			s.variables[i].lowers = make([]*fmConstraint, 2)
			s.variables[i].uppers = make([]*fmConstraint, 2)
		}
	}
	s.variables[5].lowers = s.variables[5].lowers[:1]
	s.variables[5].uppers = s.variables[5].uppers[:1]
	s.variables[6].forbidden = true
	got, err := s.candidates()
	fmMust(t, err)
	if !reflect.DeepEqual(got, []uint32{2, 3, 1, 4, 5, 0}) {
		t.Fatal("source stable candidate order", got)
	}
	for _, tc := range []struct {
		l, u uint64
		want uint32
	}{{0, 1 << 40, 0}, {1 << 32, 1, ^uint32(0)}, {65535, 65535, 4294836225}, {1 << 63, 1 << 63, ^uint32(0)}} {
		if cost := fmCost(tc.l, tc.u); cost != tc.want {
			t.Fatal("cost saturation", cost, tc)
		}
	}
	// Clear synthetic use-list lengths before testing concrete bound rows.
	for i := range s.variables {
		s.variables[i].lowers = nil
		s.variables[i].uppers = nil
	}
	x, y, real := uint32(0), uint32(2), uint32(1)
	lower := f.rawConstraint(s, nil, []uint32{x, y}, []string{"-1", "2"}, "3", false)
	upper := f.rawConstraint(s, nil, []uint32{x, y}, []string{"2", "1"}, "5", false)
	s.variables[x].lowers = []*fmConstraint{lower}
	s.variables[x].uppers = []*fmConstraint{upper}
	yes, err := s.canEliminate(x)
	fmMust(t, err)
	if !yes {
		t.Fatal("unit lower coefficients rejected")
	}
	lower = f.rawConstraint(s, nil, []uint32{x, y}, []string{"-3", "2"}, "3", false)
	s.variables[x].lowers = []*fmConstraint{lower}
	yes, err = s.canEliminate(x)
	fmMust(t, err)
	if yes {
		t.Fatal("non-unit integer coefficients accepted")
	}
	upper = f.rawConstraint(s, nil, []uint32{x, real}, []string{"1", "1"}, "5", false)
	s.variables[x].uppers = []*fmConstraint{upper}
	yes, err = s.canEliminate(x)
	fmMust(t, err)
	if yes {
		t.Fatal("mixed-real integer elimination accepted")
	}
	s.variables[x].lowers = nil
	yes, err = s.tryEliminate(x)
	fmMust(t, err)
	if !yes || !upper.dead {
		t.Fatal("one-sided integer bound must precede mixed-sort refusal")
	}
}

func TestFMResolutionCoefficientsCancellationStrictnessAndLiterals(t *testing.T) {
	for _, mode := range []string{"general", "true", "inconsistent", "tautology", "strict-integer-result", "literal-only"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			s := f.store()
			for i := uint32(0); i < 4; i++ {
				sort := f.real
				if i == 3 {
					sort = f.integer
				}
				_, err := s.variable(f.variable(i, sort))
				fmMust(t, err)
			}
			p, err := s.boolean(f.constant("fm-p", f.boolean))
			fmMust(t, err)
			q, err := s.boolean(f.constant("fm-q", f.boolean))
			fmMust(t, err)
			lower := f.rawConstraint(s, []int32{p}, []uint32{0, 1, 2}, []string{"-2", "3", "5"}, "7", false)
			upper := f.rawConstraint(s, []int32{p, q}, []uint32{0, 1, 3}, []string{"3", "-4", "2"}, "11", true)
			switch mode {
			case "true":
				lower = f.rawConstraint(s, nil, []uint32{0, 1}, []string{"-1", "1"}, "0", false)
				upper = f.rawConstraint(s, nil, []uint32{0, 1}, []string{"1", "-1"}, "0", false)
			case "inconsistent", "tautology", "literal-only":
				var l, u []int32
				if mode == "tautology" {
					l, u = []int32{p}, []int32{-p}
				}
				if mode == "literal-only" {
					l, u = []int32{p}, []int32{q}
				}
				lower = f.rawConstraint(s, l, []uint32{0}, []string{"-1"}, "-2", false)
				upper = f.rawConstraint(s, u, []uint32{0}, []string{"1"}, "1", false)
			case "strict-integer-result":
				lower = f.rawConstraint(s, nil, []uint32{0, 3}, []string{"-1", "1"}, "0", true)
				upper = f.rawConstraint(s, nil, []uint32{0}, []string{"1"}, "0", false)
			}
			c, err := s.resolve(lower, upper, 0)
			fmMust(t, err)
			switch mode {
			case "true", "tautology":
				if c != nil || s.inconsistent {
					t.Fatal("tautological resolvent retained")
				}
			case "inconsistent":
				if c != nil || !s.inconsistent {
					t.Fatal("inconsistency missed")
				}
			case "literal-only":
				if c == nil || len(c.variables) != 0 || !reflect.DeepEqual(c.literals, []int32{p, q}) {
					t.Fatal("literal-only resolvent", c)
				}
				fmNumbers(t, s, c, "-1")
			case "strict-integer-result":
				if c == nil || c.strict || !reflect.DeepEqual(c.variables, []uint32{3}) {
					t.Fatal("integer result strictness")
				}
				fmNumbers(t, s, c, "-1", "1")
			default:
				if c == nil || !c.strict || !reflect.DeepEqual(c.variables, []uint32{1, 2, 3}) || !reflect.DeepEqual(c.literals, []int32{p, q}) {
					t.Fatal("resolvent ordering or literals")
				}
				fmNumbers(t, s, c, "43", "1", "15", "4")
			}
		})
	}
}

func TestFMEliminationGrowthCutoffsAndFinalPairBoundary(t *testing.T) {
	for _, tc := range []struct {
		nl, nu int
		want   bool
	}{{3, 3, false}, {2, 3, true}, {9, 9, false}, {2, 129, false}} {
		t.Run(fmt.Sprintf("%dx%d", tc.nl, tc.nu), func(t *testing.T) {
			f := newFMFixture(t)
			s := f.store()
			for i := uint32(0); i < 3; i++ {
				_, err := s.variable(f.variable(i, f.real))
				fmMust(t, err)
			}
			s.variables[1].forbidden = true
			s.variables[2].forbidden = true
			for i := 0; i < tc.nl; i++ {
				c := f.rawConstraint(s, nil, []uint32{0, 1}, []string{"-1", "1"}, fmt.Sprint(i), false)
				_, err := s.register(c)
				fmMust(t, err)
			}
			for i := 0; i < tc.nu; i++ {
				c := f.rawConstraint(s, nil, []uint32{0, 2}, []string{"1", "1"}, fmt.Sprint(i), false)
				_, err := s.register(c)
				fmMust(t, err)
			}
			yes, err := s.tryEliminate(0)
			fmMust(t, err)
			if yes != tc.want {
				t.Fatal("growth cutoff", yes, tc)
			}
			if tc.want {
				if len(s.outputs) != tc.nl*tc.nu {
					t.Fatal("last-pair boundary changed", len(s.outputs))
				}
			} else {
				for _, c := range append(append([]*fmConstraint(nil), s.variables[0].lowers...), s.variables[0].uppers...) {
					if c.dead {
						t.Fatal("refused elimination killed original row")
					}
				}
				for i := tc.nl + tc.nu; i < len(s.constraints); i++ {
					if !s.constraints[i].closed {
						t.Fatal("aborted batch retained owned constraint")
					}
				}
			}
		})
	}
}
