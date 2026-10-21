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
	if want := "WARNING: ignoring nopats annotation because Z3 couldn't find any other pattern (quantifier id: reduce-q)\n"; output.String() != want {
		t.Fatalf("source no-pattern warning: got %q, want %q", output.String(), want)
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
	if want := "WARNING: using arith. in pattern (quantifier id: reduce-q), the weight was increased to 5 (this value can be modified using PI_ARITH_WEIGHT=<val>).\n"; output.String() != want {
		t.Fatalf("source conservative arithmetic warning: got %q, want %q", output.String(), want)
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
	if want := "WARNING: using non nested arith. pattern (quantifier id: reduce-q), the weight was increased to 10 (this value can be modified using PI_NON_NESTED_ARITH_WEIGHT=<val>).\n"; output.String() != want {
		t.Fatalf("source non-nested arithmetic warning: got %q, want %q", output.String(), want)
	}
	patternMust(t, m.DelRef(&result))
	patternMust(t, c.close())

	// The source casts the unsigned pi.arith value to its enum without
	// narrowing or clamping. An unnamed value follows the non-nested retry,
	// unlike AP_NO or the conservative-only retry.
	output.Reset()
	c = newOwner()
	c.forbiddenFamilies[ast.NullFamilyID] = struct{}{}
	opts.arithmeticMode = arithmeticPatternMode(257)
	result, changed, err = c.reduceQuantifierNoDatabase(q, pmul, nil, opts)
	patternMust(t, err)
	if !changed {
		t.Fatal("unnamed unsigned arithmetic mode must reach non-nested retry")
	}
	check(result, mul, 10)
	if want := "WARNING: using non nested arith. pattern (quantifier id: reduce-q), the weight was increased to 10 (this value can be modified using PI_NON_NESTED_ARITH_WEIGHT=<val>).\n"; output.String() != want {
		t.Fatalf("unnamed mode non-nested retry warning: got %q, want %q", output.String(), want)
	}
	patternMust(t, m.DelRef(&result))
	patternMust(t, c.close())

	output.Reset()
	c = newOwner()
	c.forbiddenFamilies[ast.NullFamilyID] = struct{}{}
	opts.arithmeticMode = arithmeticPatternsNo
	c.forbiddenFamilies[arith] = struct{}{} // Source constructor caches AP_NO before reduction.
	result, changed, err = c.reduceQuantifierNoDatabase(q, pmul, nil, opts)
	patternMust(t, err)
	if want := "WARNING: failed to find a pattern for quantifier (quantifier id: reduce-q)\n"; changed || result != (ast.ASTRef{}) || output.String() != want {
		t.Fatalf("AP_NO warning: changed=%v result=%v got %q, want %q", changed, result, output.String(), want)
	}
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
	if ref, changed, e := c.reduceQuantifierNoDatabase(q, body, nil, opts); e != nil || changed || ref != (ast.ASTRef{}) {
		t.Fatal("pull pass without nested quantifiers is unchanged", ref, changed, e)
	}
	ref, changed, err = c.reduceQuantifierNoDatabase(q, otherBody, nil, opts)
	patternMust(t, err)
	if !changed {
		t.Fatal("quantifier-free pull must publish the changed body")
	}
	updated, err = m.RefGet(ref)
	patternMust(t, err)
	actualBody, err = m.GetQuantifierBody(updated)
	patternMust(t, err)
	if actualBody != otherBody {
		t.Fatal("quantifier-free pull lost the changed body", actualBody, otherBody)
	}
	patternMust(t, m.DelRef(&ref))
	inner, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: false, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(852)}, Body: truth,
	})
	patternMust(t, err)
	patternMust(t, m.IncRef(inner))
	t.Cleanup(func() { patternMust(t, m.DecRef(inner)) })
	nestedBody, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{x, inner})
	patternMust(t, err)
	patternMust(t, m.IncRef(nestedBody))
	t.Cleanup(func() { patternMust(t, m.DecRef(nestedBody)) })
	if _, _, e := c.reduceQuantifierNoDatabase(q, nestedBody, nil, opts); !errors.Is(e, ast.ErrASTPending) {
		t.Fatal("mixed-polarity nested pull remains explicit", e)
	}
	opts.pullQuantifiers = false
	opts.useDatabase = true
	if ref, changed, e := c.reduceQuantifierNoDatabase(q, body, nil, opts); e != nil || changed || ref != (ast.ASTRef{}) {
		t.Fatal("database miss must continue ordinary inference", ref, changed, e)
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
