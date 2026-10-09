// Source: src/sat/tactic/goal2sat.cpp:110-423;
// src/sat/tactic/sat_tactic.cpp:80-110.
package tactic

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/sat"
)

// The truth table is an independent bounded correctness check. Production
// must use the source translator and the existing CDCL SAT implementation.
func TestSATGoalTranslatorNestedBooleanTruthTableAndModel(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		t.Run(fmt.Sprint(bits), func(t *testing.T) {
			m := astTestManager(t)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			astTestPin(t, m, sort)
			vars := make([]ast.ASTHandle, 3)
			for i, name := range []string{"p", "q", "r"} {
				vars[i], e = m.MkConst(astTestSymbol(t, m, name), sort)
				astTestMust(t, e)
				astTestPin(t, m, vars[i])
			}
			mk := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
				h, e := m.MkFamilyApp(0, op, nil, args)
				astTestMust(t, e)
				astTestPin(t, m, h)
				return h
			}
			// (p or q) and not(p and q), additionally conjoined with ite(r,p,q).
			shared := mk(ast.ASTOpAnd, vars[0], vars[1])
			formula := mk(ast.ASTOpAnd, mk(ast.ASTOpOr, vars[0], vars[1]), mk(ast.ASTOpNot, shared), mk(ast.ASTOpIte, vars[2], vars[0], vars[1]))
			assertions := []ast.ASTHandle{formula}
			for i, v := range vars {
				if bits&(1<<i) == 0 {
					v = mk(ast.ASTOpNot, v)
				}
				assertions = append(assertions, v)
			}
			s, _, _ := satSolverCheckFixture(t)
			tr, e := NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			satOK(t, tr.Compile(assertions))
			got, e := s.Check(nil)
			satOK(t, e)
			p, q, r := bits&1 != 0, bits&2 != 0, bits&4 != 0
			want := lbool.False
			selected := q
			if r {
				selected = p
			}
			if (p || q) && !(p && q) && selected {
				want = lbool.True
			}
			if got != want {
				t.Fatalf("original Boolean goal: got %v want %v", got, want)
			}
			if got == lbool.True {
				model, e := s.AppendModel(nil)
				satOK(t, e)
				for i, h := range vars {
					v, e := tr.Variable(h)
					satOK(t, e)
					if v == sat.NullBoolVar || int(v) >= len(model) {
						t.Fatal("missing source atom binding", i, v)
					}
					truth := model[v] == lbool.True
					if truth != (bits&(1<<i) != 0) {
						t.Fatal("model violates original Boolean assertion", i, model[v])
					}
				}
			}
		})
	}
}

func TestSATGoalTranslatorConstants(t *testing.T) {
	for _, truth := range []bool{false, true} {
		t.Run(fmt.Sprint(truth), func(t *testing.T) {
			m := astTestManager(t)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			h, e := m.MkFalse()
			if truth {
				h, e = m.MkTrue()
			}
			astTestMust(t, e)
			satOK(t, tr.Compile([]ast.ASTHandle{h, h}))
			got, e := s.Check(nil)
			satOK(t, e)
			want := lbool.False
			if truth {
				want = lbool.True
			}
			if got != want {
				t.Fatal(got, want)
			}
			v, e := tr.Variable(h)
			satOK(t, e)
			if v != sat.NullBoolVar {
				t.Fatal("source true/false fake variable must not enter atom map", v)
			}
		})
	}
}

func TestSATGoalTranslatorConnectivesTruthTable(t *testing.T) {
	for _, op := range []int32{ast.ASTOpOr, ast.ASTOpAnd, ast.ASTOpIff, ast.ASTOpEq, ast.ASTOpIte} {
		for _, nested := range []bool{false, true} {
			for _, negate := range []bool{false, true} {
				for bits := 0; bits < 8; bits++ {
					t.Run(fmt.Sprintf("%d/nested=%v/negate=%v/%d", op, nested, negate, bits), func(t *testing.T) {
						m := astTestManager(t)
						sort, e := m.MkBoolSort()
						astTestMust(t, e)
						astTestPin(t, m, sort)
						vars := make([]ast.ASTHandle, 3)
						for i, name := range []string{"p", "q", "r"} {
							vars[i], e = m.MkConst(astTestSymbol(t, m, name), sort)
							astTestMust(t, e)
							astTestPin(t, m, vars[i])
						}
						mk := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
							h, e := m.MkFamilyApp(0, op, nil, args)
							astTestMust(t, e)
							astTestPin(t, m, h)
							return h
						}
						args := vars[:2]
						if op == ast.ASTOpIte {
							args = vars
						}
						term := mk(op, args...)
						if nested {
							term = mk(ast.ASTOpOr, term, term)
						} // Two uses must share one auxiliary variable.
						if negate {
							term = mk(ast.ASTOpNot, term)
						}
						assertions := []ast.ASTHandle{term}
						for i, v := range vars {
							if bits&(1<<i) == 0 {
								v = mk(ast.ASTOpNot, v)
							}
							assertions = append(assertions, v)
						}
						s, _, _ := satSolverCheckFixture(t)
						tr, e := NewGoalTranslator(m, s)
						satOK(t, e)
						t.Cleanup(func() { satOK(t, tr.Close()) })
						satOK(t, tr.Compile(assertions))
						p, q, r := bits&1 != 0, bits&2 != 0, bits&4 != 0
						truth := false
						switch op {
						case ast.ASTOpOr:
							truth = p || q
						case ast.ASTOpAnd:
							truth = p && q
						case ast.ASTOpIff, ast.ASTOpEq:
							truth = p == q
						case ast.ASTOpIte:
							truth = q
							if !p {
								truth = r
							}
						}
						if negate {
							truth = !truth
						}
						want := lbool.False
						if truth {
							want = lbool.True
						}
						got, e := s.Check(nil)
						satOK(t, e)
						if got != want {
							t.Fatal("model must satisfy original connective", got, want)
						}
						if nested && s.Variables.NumVars() != 4 {
							t.Fatal("shared compound must reuse exactly one auxiliary variable", s.Variables.NumVars())
						}
					})
				}
			}
		}
	}
}

func TestSATGoalTranslatorAtomOwnershipAndClosedGuard(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, sort)
	p, e := m.MkConst(astTestSymbol(t, m, "p"), sort)
	astTestMust(t, e)
	astTestPin(t, m, p)
	before, e := m.GetRefCount(p)
	astTestMust(t, e)
	s, _, _ := satSolverCheckFixture(t)
	tr, e := NewGoalTranslator(m, s)
	satOK(t, e)
	satOK(t, tr.Compile([]ast.ASTHandle{p, p}))
	after, e := m.GetRefCount(p)
	astTestMust(t, e)
	if after != before+1 {
		t.Fatal("atom map must retain once", before, after)
	}
	satOK(t, tr.Close())
	satOK(t, tr.Close())
	after, e = m.GetRefCount(p)
	astTestMust(t, e)
	if after != before {
		t.Fatal("atom map reference must be released", before, after)
	}
	if e = tr.Compile([]ast.ASTHandle{p}); e == nil {
		t.Fatal("closed translator accepted assertions")
	}
}

func TestSATGoalTranslatorBooleanInterfaceAndInterpretedAtoms(t *testing.T) {
	for _, andRoot := range []bool{false, true} {
		t.Run(fmt.Sprint(andRoot), func(t *testing.T) {
			m := astTestManager(t)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			astTestPin(t, m, sort)
			p, e := m.MkConst(astTestSymbol(t, m, "p"), sort)
			astTestMust(t, e)
			astTestPin(t, m, p)
			decl, e := m.MkFuncDecl(astTestSymbol(t, m, "predicate"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
			astTestMust(t, e)
			astTestPin(t, m, decl)
			atom, e := m.MkApp(decl, []ast.ASTHandle{p})
			astTestMust(t, e)
			astTestPin(t, m, atom)
			root := atom
			if andRoot {
				root, e = m.MkAnd(p, atom)
				astTestMust(t, e)
				astTestPin(t, m, root)
			}
			before, e := m.GetRefCount(atom)
			astTestMust(t, e)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			satOK(t, tr.Compile([]ast.ASTHandle{root, p}))
			v, e := tr.Variable(p)
			satOK(t, e)
			ext, e := s.Variables.IsExternal(v)
			satOK(t, e)
			if ext == andRoot {
				t.Fatal("source Boolean-interface collector must retain AND omission", ext, andRoot)
			}
			v, e = tr.Variable(atom)
			satOK(t, e)
			ext, e = s.Variables.IsExternal(v)
			satOK(t, e)
			if !ext {
				t.Fatal("interpreted atom must be external")
			}
			interpreted, e := tr.HasInterpretedAtoms()
			satOK(t, e)
			if !interpreted {
				t.Fatal("SAT tactic must detect interpreted atoms before claiming a Boolean model")
			}
			after, e := m.GetRefCount(atom)
			astTestMust(t, e)
			if after != before+2 {
				t.Fatal("atom map and interpreted trail each retain the atom", before, after)
			}
			satOK(t, tr.Close())
			after, e = m.GetRefCount(atom)
			astTestMust(t, e)
			if after != before {
				t.Fatal("source references leaked", before, after)
			}
		})
	}
}

func TestSATGoalTranslatorCancellationAndMemoryParams(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(fmt.Sprint(cancel), func(t *testing.T) {
			m := astTestManager(t)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			astTestPin(t, m, sort)
			p, e := m.MkConst(astTestSymbol(t, m, "p"), sort)
			astTestMust(t, e)
			astTestPin(t, m, p)
			h, e := m.MkNot(p)
			astTestMust(t, e)
			astTestPin(t, m, h)
			s, c, params := satSolverCheckFixture(t)
			if !cancel {
				name := "max_memory"
				satOK(t, c.SetUIntName(&params, &name, 0))
				satOK(t, s.UpdateParams(&params))
			}
			tr, e := NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			if cancel {
				m.Limit.Cancel()
				defer m.Limit.ResetCancel()
			}
			if e = tr.Compile([]ast.ASTHandle{h}); e == nil {
				t.Fatal("source frame checkpoint must stop translation")
			}
			if e = tr.Compile([]ast.ASTHandle{h}); !errors.Is(e, ErrSATGoalFailed) {
				t.Fatal("interrupted translator cannot be reused", e)
			}
		})
	}
}

func TestSATGoalTranslatorAtomMapGrowthAndModel(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, sort)
	vars := make([]ast.ASTHandle, 40)
	assertions := make([]ast.ASTHandle, 1, 41)
	for i := range vars {
		vars[i], e = m.MkConst(astTestSymbol(t, m, fmt.Sprintf("p%d", i)), sort)
		astTestMust(t, e)
		astTestPin(t, m, vars[i])
		h := vars[i]
		if i != 0 {
			h, e = m.MkNot(h)
			astTestMust(t, e)
			astTestPin(t, m, h)
		}
		assertions = append(assertions, h)
	}
	assertions[0], e = m.MkOrArray(vars)
	astTestMust(t, e)
	astTestPin(t, m, assertions[0])
	s, _, _ := satSolverCheckFixture(t)
	tr, e := NewGoalTranslator(m, s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	satOK(t, tr.Compile(assertions))
	if len(tr.Atoms.Cells) != 64 || tr.Atoms.size != 40 {
		t.Fatal("source hash-table growth", len(tr.Atoms.Cells), tr.Atoms.size)
	}
	got, e := s.Check(nil)
	satOK(t, e)
	if got != lbool.True {
		t.Fatal(got)
	}
	model, e := s.AppendModel(nil)
	satOK(t, e)
	for i, h := range vars {
		v, e := tr.Variable(h)
		satOK(t, e)
		want := lbool.False
		if i == 0 {
			want = lbool.True
		}
		if v == sat.NullBoolVar || int(v) >= len(model) || model[v] != want {
			t.Fatal("model binding lost across source table growth", i, v)
		}
	}
}

func TestSATGoalTranslatorRejectsUnnormalizedSourceOperators(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	astTestPin(t, m, sort)
	p, e := m.MkConst(astTestSymbol(t, m, "p"), sort)
	astTestMust(t, e)
	astTestPin(t, m, p)
	q, e := m.MkConst(astTestSymbol(t, m, "q"), sort)
	astTestMust(t, e)
	astTestPin(t, m, q)
	for _, op := range []int32{ast.ASTOpXor, ast.ASTOpImplies, ast.ASTOpDistinct} {
		t.Run(fmt.Sprint(op), func(t *testing.T) {
			h, e := m.MkFamilyApp(0, op, nil, []ast.ASTHandle{p, q})
			astTestMust(t, e)
			astTestPin(t, m, h)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := NewGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			if e = tr.Compile([]ast.ASTHandle{h}); !errors.Is(e, ErrSATGoalUnsupported) {
				t.Fatal("source translator requires prior simplification", e)
			}
		})
	}
}
