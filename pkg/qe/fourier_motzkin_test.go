package qe

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func (f *fmFixture) conjunctions(forms ...ast.ASTHandle) *[]ast.ASTRef {
	result := new([]ast.ASTRef)
	f.t.Cleanup(func() {
		for i := len(*result) - 1; i >= 0; i-- {
			if (*result)[i] != (ast.ASTRef{}) {
				fmMust(f.t, f.r.manager.DelRef(&(*result)[i]))
			}
		}
	})
	for _, h := range forms {
		ref, err := f.r.manager.NewRef(h)
		fmMust(f.t, err)
		*result = append(*result, ref)
	}
	return result
}
func (f *fmFixture) eliminator() *FourierMotzkin {
	r, err := NewFourierMotzkin(f.r.manager, f.r.selection, f.r.config)
	fmMust(f.t, err)
	f.t.Cleanup(func() { fmMust(f.t, r.Close()) })
	return r
}
func fmLinear(f *fmFixture, real bool, xs []ast.ASTHandle, as []string, constant string, strict bool) ast.ASTHandle {
	m := f.r.manager
	num := func(text string) ast.ASTHandle {
		if real {
			return f.own(m.ArithmeticRealNumeralText(text))
		}
		return f.number(text)
	}
	var mons []ast.ASTHandle
	for i, x := range xs {
		if as[i] != "1" {
			x = f.own(m.MkArithmeticMul(num(as[i]), x))
		}
		mons = append(mons, x)
	}
	lhs := mons[0]
	if len(mons) > 1 {
		lhs = f.own(m.MkFamilyApp(f.r.arithmetic, 6, nil, mons))
	}
	if strict {
		return f.own(m.MkNot(f.own(m.MkArithmeticGE(lhs, num(constant)))))
	}
	return f.own(m.MkArithmeticLE(lhs, num(constant)))
}

func fmDriverCase(f *fmFixture, mode string) (forms, expected, sorts []ast.ASTHandle, declarations string) {
	m := f.r.manager
	real := mode != "integer-unit" && mode != "integer-nonunit" && mode != "boolean-clause"
	sort, sortText := f.integer, "Int"
	if real {
		sort, sortText = f.real, "Real"
	}
	x, y, z := f.variable(0, sort), f.variable(1, sort), f.variable(2, sort)
	keep := f.predicate("fm-keep", y, z)
	sorts = []ast.ASTHandle{sort, sort, sort}
	declarations = "(declare-fun fm-keep (" + sortText + " " + sortText + ") Bool)\n"
	la, ua := "-1", "1"
	if mode == "integer-unit" {
		ua = "2"
	}
	if mode == "integer-nonunit" || mode == "boolean-clause" {
		la, ua = "-2", "3"
	}
	lower := fmLinear(f, real, []ast.ASTHandle{x, y}, []string{la, "1"}, "0", false)
	upper := fmLinear(f, real, []ast.ASTHandle{x, z}, []string{ua, "1"}, "5", false)
	forms = []ast.ASTHandle{lower, upper, keep}
	residual := fmLinear(f, real, []ast.ASTHandle{y, z}, []string{ua, "1"}, "5", false)
	expected = []ast.ASTHandle{keep, residual}
	switch mode {
	case "real-large":
		lower = fmLinear(f, true, []ast.ASTHandle{x, y}, []string{"-18446744073709551617", "1"}, "0", false)
		upper = fmLinear(f, true, []ast.ASTHandle{x, z}, []string{"18446744073709551619", "1"}, "5", false)
		forms = []ast.ASTHandle{lower, upper, keep}
		expected = []ast.ASTHandle{keep, fmLinear(f, true, []ast.ASTHandle{y, z}, []string{"18446744073709551619", "18446744073709551617"}, "92233720368547758085", false)}
	case "integer-nonunit":
		expected = []ast.ASTHandle{keep, upper, lower}
	case "boolean-clause":
		p, q := f.constant("fm-p", f.boolean), f.constant("fm-q", f.boolean)
		lower = f.own(m.MkOrArray([]ast.ASTHandle{p, lower}))
		upper = f.own(m.MkOrArray([]ast.ASTHandle{f.own(m.MkNot(q)), upper}))
		forms = []ast.ASTHandle{lower, upper, keep}
		expected = []ast.ASTHandle{keep, upper, lower}
		declarations += "(declare-const fm-p Bool)\n(declare-const fm-q Bool)\n"
	case "one-sided":
		forms = []ast.ASTHandle{lower, keep}
		expected = []ast.ASTHandle{keep}
	case "strict":
		lower = fmLinear(f, true, []ast.ASTHandle{x, y}, []string{"-1", "1"}, "0", true)
		forms = []ast.ASTHandle{lower, upper, keep}
		expected = []ast.ASTHandle{keep, fmLinear(f, true, []ast.ASTHandle{y, z}, []string{"1", "1"}, "5", true)}
	case "inconsistent":
		forms = []ast.ASTHandle{fmLinear(f, true, []ast.ASTHandle{x}, []string{"-1"}, "-2", false), fmLinear(f, true, []ast.ASTHandle{x}, []string{"1"}, "1", false)}
		expected = []ast.ASTHandle{f.own(m.MkFalse())}
	case "tautology":
		forms = []ast.ASTHandle{fmLinear(f, true, []ast.ASTHandle{x}, []string{"-1"}, "0", false), fmLinear(f, true, []ast.ASTHandle{x}, []string{"1"}, "0", false)}
		expected = nil
	case "literal-resolution":
		p, q := f.constant("fm-p", f.boolean), f.constant("fm-q", f.boolean)
		l := fmLinear(f, true, []ast.ASTHandle{x}, []string{"-1"}, "-2", false)
		u := fmLinear(f, true, []ast.ASTHandle{x}, []string{"1"}, "1", false)
		forms = []ast.ASTHandle{f.own(m.MkOrArray([]ast.ASTHandle{p, l})), f.own(m.MkOrArray([]ast.ASTHandle{q, u}))}
		expected = []ast.ASTHandle{f.own(m.MkOrArray([]ast.ASTHandle{p, q}))}
		declarations += "(declare-const fm-p Bool)\n(declare-const fm-q Bool)\n"
	}
	return
}

func TestFourierMotzkinDriverProjectionAndRepeatedBooleanRuns(t *testing.T) {
	for _, mode := range []string{"real", "real-large", "integer-unit", "integer-nonunit", "boolean-clause", "one-sided", "strict", "inconsistent", "tautology", "literal-resolution"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			forms, want, _, _ := fmDriverCase(f, mode)
			d := f.eliminator()
			// Repeat fresh inputs through the same owner, including unresolved
			// Boolean clauses whose index mappings must be rebuilt per run.
			for run := 0; run < 2; run++ {
				conjs := f.conjunctions(forms...)
				fmMust(t, d.Apply(conjs))
				if len(*conjs) != len(want) {
					t.Fatalf("run %d: %d formulas, want %d", run, len(*conjs), len(want))
				}
				for i, h := range want {
					got, err := f.r.manager.RefGet((*conjs)[i])
					fmMust(t, err)
					if got != h {
						var out bytes.Buffer
						fmMust(t, writeExpression(f.r.manager, f.r.config, &out, got))
						t.Fatalf("run %d formula %d: %s", run, i, out.String())
					}
				}
			}
		})
	}
}

func TestFourierMotzkinQuantifierRefusalCancellationAndOwnerGuards(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	x := f.variable(0, f.integer)
	ineq := fmLinear(f, false, []ast.ASTHandle{x}, []string{"1"}, "3", false)
	name, err := m.Symbols.Intern("fm-bound")
	fmMust(t, err)
	quant := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{name}, Body: f.predicate("fm-quantifier", x)}))
	d := f.eliminator()
	conjs := f.conjunctions(ineq, quant)
	fmMust(t, d.Apply(conjs))
	for i, h := range []ast.ASTHandle{ineq, quant} {
		got, err := m.RefGet((*conjs)[i])
		fmMust(t, err)
		if got != h {
			t.Fatal("quantified refusal changed input")
		}
	}
	conjs = f.conjunctions(ineq)
	m.ResourceLimit().IncCancel()
	err = d.Apply(conjs)
	reason := m.ResourceLimit().CancelMessage()
	m.ResourceLimit().DecCancel()
	var canceled tactic.TacticException
	if !errors.As(err, &canceled) || canceled.Message() != reason {
		t.Fatal("source cancellation", err)
	}
	if len(*conjs) != 1 {
		t.Fatal("canceled call changed input")
	}
	fmMust(t, d.Apply(conjs))
	if len(*conjs) != 0 {
		t.Fatal("cancel recovery failed")
	}
	empty, err := NewVariableSetTest(m, nil, true)
	fmMust(t, err)
	t.Cleanup(func() { fmMust(t, empty.Close()) })
	fmMust(t, d.SetVariableTest(empty))
	conjs = f.conjunctions(ineq)
	fmMust(t, d.Apply(conjs))
	if len(*conjs) != 1 {
		t.Fatal("selector update ignored")
	}
	copy := *d
	if err := copy.Apply(conjs); err == nil {
		t.Fatal("copied owner accepted")
	}
	if err := copy.Close(); err == nil {
		t.Fatal("copied owner closed original")
	}
	fmMust(t, d.Close())
	if err := d.Apply(conjs); err == nil {
		t.Fatal("closed owner accepted")
	}
	foreign := newFMFixture(t)
	if _, err := NewFourierMotzkin(m, foreign.r.selection, f.r.config); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("foreign selector accepted", err)
	}
}
