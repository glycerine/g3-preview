package smtcore

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func boundClauseSet(ls []SMTLiteral) string {
	copy := append([]SMTLiteral(nil), ls...)
	sort.Slice(copy, func(i, j int) bool { return copy[i].Index() < copy[j].Index() })
	return fmt.Sprint(copy)
}

func TestSMTArithmeticBoundAxiomAllocationFrontiersAndRetry(t *testing.T) {
	succeeded := false
	refusals := 0
	for stop := int64(0); stop < 128 && !succeeded; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m := astTestManager(t)
			astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
			family, err := m.RegisterArithmeticPlugin()
			astTestMust(t, err)
			_, _, s := smtInternalizerFixture(t, m, nil)
			a, err := NewSMTLinearArithmeticTheory(family)
			astTestMust(t, err)
			astTestMust(t, s.RegisterTheory(a))
			sort, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			f := &arithmeticObjectiveFixture{t: t, m: m, s: s, a: a, sort: sort}
			x := f.variable("bound-stopped-x")
			h1 := f.app(smtArithOpGE, x, f.number("1"))
			h2 := f.app(smtArithOpGE, x, f.number("2"))
			// Fill actual warm parameter slots until the manager must grow.
			// Keep them alive so this tests a real parameter-storage frontier,
			// rather than assuming that a warm inline coefficient allocates.
			var padding []ast.ASTParameter
			t.Cleanup(func() { astTestMust(t, smtArithmeticReleaseParameters(m, padding)) })
			cold := false
			m.Budget().FailAfter(0)
			for i := 0; i < 8192; i++ {
				p, err := m.NewParameterInt(int32(i))
				if err != nil {
					cold = errors.Is(err, arena.ErrInjectedFailure)
					break
				}
				padding = append(padding, p)
			}
			m.Budget().FailAfter(-1)
			if !cold {
				t.Fatal("did not reach a cold parameter allocation")
			}
			baseline := m.Stats().LiveParameters
			astTestMust(t, s.PushScope())
			astTestMust(t, s.Internalize(h1, true))
			astTestMust(t, s.Internalize(h2, true))
			m.Budget().FailAfter(stop)
			err = a.Propagate()
			m.Budget().FailAfter(-1)
			if err != nil {
				refusals++
				if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal("unexpected bound-axiom allocation refusal", err)
				}
				astTestMust(t, a.Propagate())
			} else {
				succeeded = true
			}
			v1, _ := s.BoolVar(h1)
			v2, _ := s.BoolVar(h2)
			astTestMust(t, s.AssignLiteral(NewSMTLiteral(v2, false), NewSMTBooleanAxiom()))
			consistent, err := s.PropagateCore()
			astTestMust(t, err)
			if !consistent {
				t.Fatal("retry introduced a spurious conflict")
			}
			value, err := s.Assignment(NewSMTLiteral(v1, false))
			astTestMust(t, err)
			if value != lbool.True {
				t.Fatal("retry lost the stronger-to-weaker bound axiom", value)
			}
			astTestMust(t, s.PopScopes(1))
			if m.Stats().LiveParameters != baseline {
				t.Fatal("stopped bound-axiom proof parameters leaked", m.Stats().LiveParameters, baseline)
			}
		})
		if t.Failed() {
			t.Fatalf("bound-axiom allocation frontier %d", stop)
		}
	}
	if !succeeded {
		t.Fatal("no complete bound-axiom allocation frontier")
	}
	if refusals == 0 {
		t.Fatal("bound-axiom campaign reached no actual allocation refusal")
	}
}

func TestSMTArithmeticBoundAxiomsKindsAndIntegerAdjacency(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, delayed := range []bool{false, true} {
			for _, test := range []struct {
				first, second int32
				k1, k2        int
				form          string
			}{
				{smtArithOpGE, smtArithOpGE, 1, 2, "strong-lower"},
				{smtArithOpLE, smtArithOpLE, 1, 2, "weak-upper"},
				{smtArithOpGE, smtArithOpLE, 1, 2, "cover"},
				{smtArithOpLE, smtArithOpGE, 2, 1, "cover"},
				{smtArithOpGE, smtArithOpLE, 2, 1, "gap"},
				{smtArithOpLE, smtArithOpGE, 1, 2, "gap"},
				{smtArithOpGE, smtArithOpGE, 2, 2, "same"},
				{smtArithOpGE, smtArithOpLE, 2, 2, "cover"},
			} {
				t.Run(fmt.Sprint(integer, "/", delayed, "/", test), func(t *testing.T) {
					f := newArithmeticObjectiveFixture(t, integer)
					x := f.variable("bound-kind-x")
					f.s.searching = !delayed
					h1 := f.app(test.first, x, f.number(fmt.Sprint(test.k1)))
					h2 := f.app(test.second, x, f.number(fmt.Sprint(test.k2)))
					astTestMust(t, f.s.Internalize(h1, true))
					astTestMust(t, f.s.Internalize(h2, true))
					v1, _ := f.s.BoolVar(h1)
					v2, _ := f.s.BoolVar(h2)
					l1, l2 := NewSMTLiteral(v1, false), NewSMTLiteral(v2, false)
					if delayed && len(f.s.clauses) != 0 {
						t.Fatal("bound axioms emitted before deferred flush")
					}
					astTestMust(t, f.a.Propagate())
					f.s.searching = false
					var want []string
					switch test.form {
					case "strong-lower":
						want = append(want, boundClauseSet([]SMTLiteral{l1, l2.Negated()}))
					case "weak-upper":
						want = append(want, boundClauseSet([]SMTLiteral{l1.Negated(), l2}))
					case "cover":
						want = append(want, boundClauseSet([]SMTLiteral{l1, l2}))
					case "gap":
						want = append(want, boundClauseSet([]SMTLiteral{l1.Negated(), l2.Negated()}))
						if integer {
							want = append(want, boundClauseSet([]SMTLiteral{l1, l2}))
						}
					}
					var got []string
					for _, clause := range f.s.clauses {
						got = append(got, boundClauseSet(clause.Literals))
					}
					sort.Strings(got)
					sort.Strings(want)
					if !reflect.DeepEqual(got, want) {
						t.Fatal("native bound clause membership", got, want)
					}
				})
			}
		}
	}
}

func TestSMTArithmeticBoundAxiomsNearestNeighborsAndAxisIdentity(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		t.Run(fmt.Sprint(delayed), func(t *testing.T) {
			f := newArithmeticObjectiveFixture(t, false)
			x, y := f.variable("bound-neighbor-x"), f.variable("bound-neighbor-y")
			var formulas []ast.ASTHandle
			for _, n := range []int{0, 3, 7, 10} {
				formulas = append(formulas, f.app(smtArithOpGE, x, f.number(fmt.Sprint(n))))
			}
			for _, n := range []int{-1, 4, 8, 11} {
				formulas = append(formulas, f.app(smtArithOpLE, x, f.number(fmt.Sprint(n))))
			}
			for _, h := range formulas {
				astTestMust(t, f.s.Internalize(h, true))
			}
			astTestMust(t, f.a.Propagate())
			before := len(f.s.clauses)
			f.s.searching = !delayed
			h := f.app(smtArithOpGE, x, f.number("5"))
			astTestMust(t, f.s.Internalize(h, true))
			unrelated := f.app(smtArithOpGE, y, f.number("5"))
			astTestMust(t, f.s.Internalize(unrelated, true))
			astTestMust(t, f.a.Propagate())
			f.s.searching = false
			v, _ := f.s.BoolVar(h)
			var want []string
			for _, index := range []int{1, 2, 5, 6} {
				other, _ := f.s.BoolVar(formulas[index])
				l1, l2 := NewSMTLiteral(v, false), NewSMTLiteral(other, false)
				switch index {
				case 1:
					l1 = l1.Negated()
				case 2:
					l2 = l2.Negated()
				case 5:
					l1 = l1.Negated()
					l2 = l2.Negated()
				}
				want = append(want, boundClauseSet([]SMTLiteral{l1, l2}))
			}
			var got []string
			for _, c := range f.s.clauses[before:] {
				got = append(got, boundClauseSet(c.Literals))
			}
			sort.Strings(got)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Fatal("bound selection included a non-neighbor or another axis", got, want)
			}
		})
	}
}

func TestSMTArithmeticBoundAxiomProofParametersAndScopeOwnership(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, _, s := smtInternalizerFixture(t, m, nil)
	a, err := NewSMTLinearArithmeticTheory(family)
	astTestMust(t, err)
	astTestMust(t, s.RegisterTheory(a))
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	f := &arithmeticObjectiveFixture{t: t, m: m, s: s, a: a, sort: sort}
	x := f.variable("bound-proof-x")
	h1 := f.app(smtArithOpGE, x, f.number("1"))
	h2 := f.app(smtArithOpGE, x, f.number("2"))
	baseline := m.Stats().LiveParameters
	for repeat := 0; repeat < 2; repeat++ {
		astTestMust(t, s.PushScope())
		astTestMust(t, s.Internalize(h1, true))
		astTestMust(t, s.Internalize(h2, true))
		astTestMust(t, a.Propagate())
		if len(s.clauses) == 0 {
			t.Fatal("missing proof-bearing bound axiom")
		}
		clause := s.clauses[len(s.clauses)-1].clause
		j, ok := clause.Justification().(*SMTTheoryAxiomJustification)
		if !ok {
			t.Fatal("bound axiom lost its theory justification")
		}
		ps := j.Parameters()
		if len(ps) != 3 {
			t.Fatal("expected farkas,1,1", len(ps))
		}
		sym, err := m.ParameterSymbol(ps[0])
		astTestMust(t, err)
		name, _, err := m.Symbols.BareString(sym)
		astTestMust(t, err)
		if name != "farkas" {
			t.Fatal(name)
		}
		for _, p := range ps[1:] {
			one, err := m.NewParameterRat(big.NewRat(1, 1))
			astTestMust(t, err)
			equal, err := m.ParameterEqual(p, one)
			astTestMust(t, err)
			astTestMust(t, m.DelParameter(&one))
			if !equal {
				t.Fatal("non-unit Farkas bound coefficient")
			}
		}
		builder := &smtConflictProofBuilder{state: s, marked: make(map[int32]struct{}), literal: make(map[SMTLiteral]ast.ASTHandle), objects: make(map[SMTJustification]ast.ASTHandle), equality: make(map[smtEnodePairKey]ast.ASTHandle)}
		proof, err := builder.clauseProof(clause)
		astTestMust(t, err)
		ref, err := m.NewRef(proof)
		astTestMust(t, err)
		fact, err := m.GetProofFact(proof)
		astTestMust(t, err)
		kind, err := m.GetDeclKind(fact)
		astTestMust(t, err)
		if kind != ast.ASTOpOr {
			t.Fatal("bound axiom proof lost its clause fact", kind)
		}
		astTestMust(t, m.DelRef(&ref))
		astTestMust(t, s.PopScopes(1))
		if _, err := m.ParameterKind(ps[1]); !errors.Is(err, ast.ErrASTHandle) {
			t.Fatal("popped region bound parameters retained", err)
		}
		if m.Stats().LiveParameters != baseline {
			t.Fatal("bound proof parameter ownership leaked", m.Stats().LiveParameters, baseline)
		}
	}
}
