package qe

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func (f *fmFixture) lite(useArray bool) *Lite {
	l, err := NewLite(f.r.manager, nil, useArray, f.r.config)
	fmMust(f.t, err)
	f.t.Cleanup(func() { fmMust(f.t, l.Close()) })
	return l
}
func liteVectorEquals(t *testing.T, f *fmFixture, refs []ast.ASTRef, hs ...ast.ASTHandle) {
	t.Helper()
	if len(refs) != len(hs) {
		t.Fatalf("%d results, want %d", len(refs), len(hs))
	}
	for i, want := range hs {
		h, err := f.r.manager.RefGet(refs[i])
		fmMust(t, err)
		if h != want {
			t.Fatalf("formula %d mismatch", i)
		}
	}
}

func TestLiteConjunctionPipelineAndComplementSelection(t *testing.T) {
	for _, mode := range []string{"real", "real-large", "integer-unit", "integer-nonunit", "literal-resolution"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			l := f.lite(true)
			forms, want, _, _ := fmDriverCase(f, mode)
			v := f.conjunctions(forms...)
			fmMust(t, l.EliminateConjunctions([]uint32{0, 1, 2}, true, v))
			liteVectorEquals(t, f, *v, want...)
		})
	}
	f := newFMFixture(t)
	m := f.r.manager
	l := f.lite(false)
	x, y := f.variable(0, f.integer), f.variable(1, f.integer)
	three := f.number("3")
	definition := f.own(m.MkEq(x, three))
	v := f.conjunctions(definition, f.predicate("lite-p", x, y))
	fmMust(t, l.EliminateConjunctions([]uint32{1}, false, v))
	liteVectorEquals(t, f, *v, f.predicate("lite-p", three, y))
}

func TestLiteEqualityThenFourierMotzkinPipeline(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	l := f.lite(true)
	x, y, z := f.variable(0, f.real), f.variable(1, f.real), f.variable(2, f.real)
	three := f.own(m.ArithmeticRealNumeralText("3"))
	forms := f.conjunctions(f.own(m.MkEq(z, three)), fmLinear(f, true, []ast.ASTHandle{x, y}, []string{"-1", "1"}, "0", false), fmLinear(f, true, []ast.ASTHandle{x, z}, []string{"1", "1"}, "5", false), f.predicate("lite-p", y))
	fmMust(t, l.EliminateConjunctions([]uint32{0, 1, 2}, true, forms))
	liteVectorEquals(t, f, *forms, f.predicate("lite-p", y), f.own(m.MkArithmeticLE(y, f.own(m.ArithmeticRealNumeralText("2")))))
}

func TestLiteDisjunctionDistributionUniqueNonGroundAndFlattening(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	l := f.lite(true)
	x := f.variable(0, f.integer)
	three, four := f.number("3"), f.number("4")
	p, q := f.predicate("lite-p", x), f.predicate("lite-q", x)
	a := f.own(m.MkAndArray([]ast.ASTHandle{f.own(m.MkEq(x, three)), p}))
	b := f.own(m.MkAndArray([]ast.ASTHandle{f.own(m.MkEq(x, four)), q}))
	disj := f.own(m.MkOrArray([]ast.ASTHandle{a, b}))
	result := f.conjunctions(disj)
	fmMust(t, l.EliminateIndices([]uint32{0}, true, (*result)[0]))
	want := f.own(m.MkOrArray([]ast.ASTHandle{f.predicate("lite-q", four), f.predicate("lite-p", three)}))
	liteVectorEquals(t, f, *result, want)
	ground := f.constant("lite-ground", f.boolean)
	vector := f.conjunctions(ground, disj)
	fmMust(t, l.EliminateConjunctions([]uint32{0}, true, vector))
	liteVectorEquals(t, f, *vector, ground, want)
	// flatten_and removes true rows, expands negated OR and preserves its
	// source child-slot order before any variable phase runs.
	other := f.constant("lite-other", f.boolean)
	groundDisj := f.own(m.MkOrArray([]ast.ASTHandle{ground, other}))
	root := f.own(m.MkNot(groundDisj))
	vector = f.conjunctions(f.own(m.MkTrue()), root)
	fmMust(t, l.EliminateConjunctions(nil, true, vector))
	liteVectorEquals(t, f, *vector, f.own(m.MkNot(other)), f.own(m.MkNot(ground)))
}

func TestLiteConstantAbstractionFilteringAndDEROnlyBoundary(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	l := f.lite(true)
	k, n, z := f.constant("lite-k", f.integer), f.constant("lite-n", f.integer), f.constant("lite-z", f.integer)
	three := f.number("3")
	root := f.own(m.MkAndArray([]ast.ASTHandle{f.own(m.MkEq(k, n)), f.own(m.MkEq(n, three)), f.predicate("lite-p", k, z)}))
	formula := f.conjunctions(root)
	variables := f.conjunctions(k, n, z)
	fmMust(t, l.EliminateConstants(variables, (*formula)[0]))
	liteVectorEquals(t, f, *variables, z)
	liteVectorEquals(t, f, *formula, f.predicate("lite-p", three, z))
	ineq := f.own(m.MkArithmeticLE(k, three))
	formula = f.conjunctions(ineq)
	variables = f.conjunctions(k)
	fmMust(t, l.EliminateConstants(variables, (*formula)[0]))
	liteVectorEquals(t, f, *variables, k)
	liteVectorEquals(t, f, *formula, ineq)
	empty := new([]ast.ASTRef)
	fmMust(t, l.EliminateConstants(empty, ast.ASTRef{}))
	b := f.constant("lite-bool", f.boolean)
	variables = f.conjunctions(b)
	formula = f.conjunctions(f.own(m.MkNot(b)))
	fmMust(t, l.EliminateConstants(variables, (*formula)[0]))
	if len(*variables) != 0 {
		t.Fatal("Boolean constant not eliminated")
	}
	h, err := m.RefGet((*formula)[0])
	fmMust(t, err)
	if !m.IsTrue(h) {
		t.Fatal("existential Boolean literal not true")
	}
	r := f.constant("lite-real", f.real)
	variables = f.conjunctions(r)
	product := f.own(m.MkArithmeticMul(f.own(m.ArithmeticRealNumeralText("2")), r))
	formula = f.conjunctions(f.own(m.MkAndArray([]ast.ASTHandle{f.own(m.MkEq(product, f.own(m.ArithmeticRealNumeralText("3")))), f.predicate("lite-real-p", r)})))
	fmMust(t, l.EliminateConstants(variables, (*formula)[0]))
	if len(*variables) != 0 {
		t.Fatal("real constant not eliminated")
	}
	// The source constant overload does not run the theory rewriter after
	// DER. Preserve its raw reciprocal product rather than folding 3/2.
	half := f.own(m.ArithmeticRealNumeralText("1/2"))
	scaled := f.own(m.MkArithmeticMul(half, f.own(m.ArithmeticRealNumeralText("3"))))
	liteVectorEquals(t, f, *formula, f.predicate("lite-real-p", scaled))
}

func TestLiteFullTraversalExistentialUniversalNestedAndIdentity(t *testing.T) {
	for _, mode := range []string{"exists", "forall", "nested", "nonquantified"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			l := f.lite(true)
			x := f.variable(0, f.integer)
			three := f.number("3")
			p := f.predicate("lite-p", x)
			definition := f.own(m.MkEq(x, three))
			name, err := m.Symbols.Intern("lite-bound")
			fmMust(t, err)
			body := f.own(m.MkAndArray([]ast.ASTHandle{definition, p}))
			if mode == "forall" {
				body = f.own(m.MkOrArray([]ast.ASTHandle{f.own(m.MkNot(definition)), p}))
			}
			root := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: mode == "forall", Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{name}, Body: body}))
			if mode == "nested" {
				root = f.predicate("lite-wrap", root)
			}
			if mode == "nonquantified" {
				root = body
			}
			formula := f.conjunctions(root)
			proof := f.conjunctions(ast.ASTHandle{})
			fmMust(t, l.Rewrite((*formula)[0], (*proof)[0]))
			want := f.predicate("lite-p", three)
			if mode == "nested" {
				want = f.predicate("lite-wrap", want)
			}
			if mode == "nonquantified" {
				want = root
			}
			liteVectorEquals(t, f, *formula, want)
		})
	}
}

func TestLiteNestedBindersPreserveOuterVariableAndAnnotations(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	l := f.lite(true)
	x, y := f.variable(0, f.integer), f.variable(1, f.integer)
	innerName, err := m.Symbols.Intern("lite-inner")
	fmMust(t, err)
	outerName, err := m.Symbols.Intern("lite-outer")
	fmMust(t, err)
	trigger := f.predicate("lite-trigger", x)
	pattern := f.own(m.MkPattern([]ast.ASTHandle{trigger}))
	body := f.own(m.MkAndArray([]ast.ASTHandle{f.own(m.MkEq(x, y)), f.predicate("lite-p", x)}))
	inner := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{innerName}, Body: body, Patterns: []ast.ASTHandle{pattern}}))
	outer := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{outerName}, Body: inner}))
	formula := f.conjunctions(outer)
	proof := f.conjunctions(ast.ASTHandle{})
	fmMust(t, l.Rewrite((*formula)[0], (*proof)[0]))
	wantInner := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{innerName}, Body: f.predicate("lite-p", y), Patterns: []ast.ASTHandle{pattern}}))
	want := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{outerName}, Body: wantInner}))
	// The original inner pattern keeps its declaration because the new
	// body still contains the outer variable. DER must not capture that y.
	h, err := m.RefGet((*formula)[0])
	fmMust(t, err)
	if h != want {
		var out bytes.Buffer
		fmMust(t, writeExpression(m, f.r.config, &out, h))
		t.Fatal("nested scope or annotation reduction", out.String())
	}
	plainInner := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{innerName}, Body: body}))
	plainOuter := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{outerName}, Body: plainInner}))
	formula = f.conjunctions(plainOuter)
	fmMust(t, l.Rewrite((*formula)[0], (*proof)[0]))
	want = f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{outerName}, Body: f.predicate("lite-p", x)}))
	liteVectorEquals(t, f, *formula, want)
}

func TestLiteArrayOptionAndOwnerGuards(t *testing.T) {
	for _, useArray := range []bool{false, true} {
		t.Run(fmt.Sprint(useArray), func(t *testing.T) {
			f := newFMFixture(t)
			m := f.r.manager
			l := f.lite(useArray)
			family, err := m.ArrayFamilyID()
			fmMust(t, err)
			sort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
			a := f.variable(0, sort)
			key, other := f.constant("lite-key", f.integer), f.constant("lite-other-key", f.integer)
			read := func(index ast.ASTHandle) ast.ASTHandle {
				return f.own(m.MkArrayAppArguments(family, ast.ASTArrayOpSelect, ast.SliceArguments([]ast.ASTHandle{a, index})))
			}
			definition := f.own(m.MkEq(read(key), f.number("7")))
			remaining := f.own(m.MkEq(read(other), f.number("9")))
			forms := f.conjunctions(definition, remaining)
			fmMust(t, l.EliminateConjunctions([]uint32{0}, true, forms))
			if useArray {
				if len(*forms) != 2 {
					t.Fatal("array phase shape")
				}
				h, err := m.RefGet((*forms)[0])
				fmMust(t, err)
				if !m.IsTrue(h) {
					t.Fatal("array definition not removed")
				}
				h, err = m.RefGet((*forms)[1])
				fmMust(t, err)
				if h == remaining {
					t.Fatal("array store substitution absent")
				}
			} else {
				liteVectorEquals(t, f, *forms, definition, remaining)
			}
			copy := *l
			if err := copy.EliminateConjunctions(nil, true, forms); err == nil {
				t.Fatal("copied owner accepted")
			}
			if err := copy.Close(); err == nil {
				t.Fatal("copied owner closed original")
			}
			fmMust(t, l.Close())
			if err := l.EliminateConjunctions(nil, true, forms); err == nil {
				t.Fatal("closed owner accepted")
			}
		})
	}
}
