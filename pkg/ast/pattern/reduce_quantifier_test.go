package pattern

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceQuantifierRetriesWarningsAndWeights(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	arith, err := m.RegisterArithmeticPlugin()
	patternMust(t, err)
	ints, err := m.MkFamilySort(arith, 1, nil)
	patternMust(t, err)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		patternMust(t, m.IncRef(h))
		t.Cleanup(func() { patternMust(t, m.DecRef(h)) })
		return h
	}
	xb, err := m.MkVar(0, b)
	patternMust(t, err)
	pin(xb)
	xi, err := m.MkVar(0, ints)
	patternMust(t, err)
	pin(xi)
	f, err := m.MkFuncDecl(symbol.Numerical(841), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	patternMust(t, err)
	pin(f)
	fx, err := m.MkApp(f, []ast.ASTHandle{xb})
	patternMust(t, err)
	pin(fx)
	mul, err := m.MkFamilyApp(arith, 9, nil, []ast.ASTHandle{xi, xi})
	patternMust(t, err)
	pin(mul)
	p, err := m.MkFuncDecl(symbol.Numerical(842), []ast.ASTHandle{ints}, b, ast.ASTFuncDeclInfo{})
	patternMust(t, err)
	pin(p)
	pmul, err := m.MkApp(p, []ast.ASTHandle{mul})
	patternMust(t, err)
	pin(pmul)
	qid, err := m.Symbols.Intern("reduce-q")
	patternMust(t, err)
	quantifier := func(body ast.ASTHandle, noPatterns []ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
			Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(843)},
			Body: body, Weight: 2, QID: qid, NoPatterns: noPatterns,
		})
		patternMust(t, e)
		return pin(q)
	}
	newOwner := func() *patternCandidates {
		t.Helper()
		c, e := newPatternCandidates(m, 1, nil, false)
		patternMust(t, e)
		t.Cleanup(func() { patternMust(t, c.close()) })
		return c
	}
	check := func(ref ast.ASTRef, child ast.ASTHandle, weight int32) {
		t.Helper()
		q, e := m.RefGet(ref)
		patternMust(t, e)
		count, e := m.GetNumPatterns(q)
		patternMust(t, e)
		if count != 1 {
			t.Fatal("inferred pattern count", count)
		}
		pattern, e := m.GetPattern(q, 0)
		patternMust(t, e)
		got, e := m.GetArg(pattern, 0)
		patternMust(t, e)
		gotWeight, e := m.GetQuantifierWeight(q)
		patternMust(t, e)
		if got != child || gotWeight != weight {
			t.Fatal("inferred pattern and weight", got, child, gotWeight, weight)
		}
	}
	var output bytes.Buffer
	warnings := diagnostic.NewWarningService(&output)
	base := quantifier(fx, []ast.ASTHandle{fx})
	c := newOwner()
	opts := quantifierInferenceOptions{arithmeticMode: arithmeticPatternsNo, noPatternWeight: -1, warningService: warnings, warnings: true}
	result, changed, err := c.reduceQuantifierNoDatabase(base, fx, []ast.ASTHandle{fx}, opts)
	patternMust(t, err)
	if !changed {
		t.Fatal("no-pattern retry must infer a replacement")
	}
	check(result, fx, 2)
	if !strings.Contains(output.String(), "ignoring nopats annotation") || !strings.Contains(output.String(), "reduce-q") {
		t.Fatal("source no-pattern warning", output.String())
	}
	patternMust(t, m.DelRef(&result))
	patternMust(t, c.close())

	output.Reset()
	c = newOwner()
	q := quantifier(pmul, nil)
	opts.arithmeticMode = arithmeticPatternsConservative
	opts.arithmeticWeight = 5
	result, changed, err = c.reduceQuantifierNoDatabase(q, pmul, nil, opts)
	patternMust(t, err)
	if !changed {
		t.Fatal("conservative arithmetic retry must infer")
	}
	check(result, pmul, 5)
	if !strings.Contains(output.String(), "using arith. in pattern") {
		t.Fatal("source conservative arithmetic warning", output.String())
	}
	patternMust(t, m.DelRef(&result))
	patternMust(t, c.close())

	output.Reset()
	c = newOwner()
	c.forbiddenFamilies[ast.NullFamilyID] = struct{}{} // Only arithmetic child may become a candidate.
	opts.arithmeticMode = arithmeticPatternsFull
	opts.nonNestedArithmeticWeight = 10
	result, changed, err = c.reduceQuantifierNoDatabase(q, pmul, nil, opts)
	patternMust(t, err)
	if !changed {
		t.Fatal("non-nested arithmetic retry must infer")
	}
	check(result, mul, 10)
	if !strings.Contains(output.String(), "using non nested arith. pattern") {
		t.Fatal("source non-nested arithmetic warning", output.String())
	}
	patternMust(t, m.DelRef(&result))
	patternMust(t, c.close())
}

func TestPatternInferenceQuantifierNoPatternReturnAndGuards(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	patternMust(t, m.IncRef(x))
	t.Cleanup(func() { patternMust(t, m.DecRef(x)) })
	truth, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpTrue, nil, nil)
	patternMust(t, err)
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{x, truth})
	patternMust(t, err)
	patternMust(t, m.IncRef(body))
	t.Cleanup(func() { patternMust(t, m.DecRef(body)) })
	qid, err := m.Symbols.Intern("no-candidate")
	patternMust(t, err)
	makeQ := func(forall bool, patterns []ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
			Forall: forall, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(851)},
			Body: body, QID: qid, Patterns: patterns,
		})
		patternMust(t, e)
		patternMust(t, m.IncRef(q))
		t.Cleanup(func() { patternMust(t, m.DecRef(q)) })
		return q
	}
	q := makeQ(true, nil)
	c, err := newPatternCandidates(m, 1, nil, false)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, c.close()) })
	var output bytes.Buffer
	opts := quantifierInferenceOptions{noPatternWeight: -1, warnings: true, warningService: diagnostic.NewWarningService(&output)}
	ref, changed, err := c.reduceQuantifierNoDatabase(q, body, nil, opts)
	patternMust(t, err)
	if changed || ref != (ast.ASTRef{}) || !strings.Contains(output.String(), "failed to find a pattern") {
		t.Fatal("unchanged body without patterns returns false after warning", changed, ref, output.String())
	}
	if len(c.candidates) != 0 || len(c.collectRefs) != 0 {
		t.Fatal("failed inference must reset candidate state")
	}
	otherBody, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpOr, nil, []ast.ASTHandle{x, truth})
	patternMust(t, err)
	patternMust(t, m.IncRef(otherBody))
	t.Cleanup(func() { patternMust(t, m.DecRef(otherBody)) })
	opts.noPatternWeight = 7
	ref, changed, err = c.reduceQuantifierNoDatabase(q, otherBody, nil, opts)
	patternMust(t, err)
	if !changed {
		t.Fatal("changed body must publish quantifier even without patterns")
	}
	updated, err := m.RefGet(ref)
	patternMust(t, err)
	actualBody, err := m.GetQuantifierBody(updated)
	patternMust(t, err)
	weight, err := m.GetQuantifierWeight(updated)
	patternMust(t, err)
	count, err := m.GetNumPatterns(updated)
	patternMust(t, err)
	if actualBody != otherBody || weight != 7 || count != 0 {
		t.Fatal("changed-body publication and no-pattern weight", actualBody, weight, count)
	}
	patternMust(t, m.DelRef(&ref))
	opts.pullQuantifiers = true
	if _, _, e := c.reduceQuantifierNoDatabase(q, body, nil, opts); !errors.Is(e, ast.ErrASTPending) {
		t.Fatal("unported pull-quantifier path must be explicit", e)
	}
	opts.pullQuantifiers = false
	opts.useDatabase = true
	if _, _, e := c.reduceQuantifierNoDatabase(q, body, nil, opts); !errors.Is(e, ast.ErrASTPending) {
		t.Fatal("unported database path must be explicit", e)
	}
	opts.useDatabase = false
	qExists := makeQ(false, nil)
	if ref, changed, e := c.reduceQuantifierNoDatabase(qExists, body, nil, opts); e != nil || changed || ref != (ast.ASTRef{}) {
		t.Fatal("existential quantifier is skipped", ref, changed, e)
	}
	pattern, err := m.MkFamilyApp(2, 0, nil, []ast.ASTHandle{body})
	patternMust(t, err)
	qAnnotated := makeQ(true, []ast.ASTHandle{pattern})
	if ref, changed, e := c.reduceQuantifierNoDatabase(qAnnotated, body, nil, opts); e != nil || changed || ref != (ast.ASTRef{}) {
		t.Fatal("already annotated quantifier is skipped", ref, changed, e)
	}
}
