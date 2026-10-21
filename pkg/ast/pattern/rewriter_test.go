package pattern

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceActualRewriterTraversalAndOwnership(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		patternMust(t, m.IncRef(h))
		t.Cleanup(func() { patternMust(t, m.DecRef(h)) })
		return h
	}
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	pin(x)
	decl := func(name uint32) ast.ASTHandle {
		t.Helper()
		d, e := m.MkFuncDecl(symbol.Numerical(name), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		return pin(d)
	}
	app := func(d ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		h, e := m.MkApp(d, []ast.ASTHandle{x})
		patternMust(t, e)
		return pin(h)
	}
	f, g := decl(861), decl(862)
	fx, gx := app(f), app(g)
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{fx, gx})
	patternMust(t, err)
	pin(body)
	quantifier := func(forall bool, expr ast.ASTHandle, noPatterns []ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
			Forall: forall, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(863)},
			Body: expr, NoPatterns: noPatterns,
		})
		patternMust(t, e)
		return pin(q)
	}
	qFirst := quantifier(true, fx, nil)
	qPreferred := quantifier(true, body, nil)
	qNoPattern := quantifier(true, fx, []ast.ASTHandle{fx})
	qExists := quantifier(false, fx, nil)
	r, err := NewInferenceRewriter(m)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, r.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, m.DelRef(&out)) })
	check := func(want ast.ASTHandle) {
		t.Helper()
		q, e := m.RefGet(out)
		patternMust(t, e)
		count, e := m.GetNumPatterns(q)
		patternMust(t, e)
		if count != 1 {
			t.Fatal("rewriter inferred pattern count", count)
		}
		p, e := m.GetPattern(q, 0)
		patternMust(t, e)
		child, e := m.GetArg(p, 0)
		patternMust(t, e)
		if child != want {
			t.Fatal("rewriter candidate selection", child, want)
		}
	}
	patternMust(t, r.Rewrite(qFirst, out))
	check(fx)
	patternMust(t, r.RegisterPreferred(g))
	patternMust(t, r.Rewrite(qPreferred, out))
	check(gx)
	patternMust(t, r.Rewrite(qNoPattern, out))
	check(fx)
	patternMust(t, r.Rewrite(qExists, out))
	got, err := m.RefGet(out)
	patternMust(t, err)
	if got != qExists {
		t.Fatal("existential quantifier must pass through the traversal", got, qExists)
	}
	patternMust(t, r.Close())
	if e := r.Rewrite(qFirst, out); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("closed inference rewriter", e)
	}
	copyOwner := *r
	if e := copyOwner.RegisterPreferred(f); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied owner must be refused", e)
	}
}

func TestPatternInferenceProofProducingTraversal(t *testing.T) {
	m := patternManager(t)
	patternMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	b, err := m.MkBoolSort()
	patternMust(t, err)
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	d, err := m.MkFuncDecl(symbol.Numerical(864), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	patternMust(t, err)
	fx, err := m.MkApp(d, []ast.ASTHandle{x})
	patternMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(865)}, Body: fx,
	})
	patternMust(t, err)
	for _, h := range []ast.ASTHandle{d, fx, q} {
		patternMust(t, m.IncRef(h))
		h := h
		t.Cleanup(func() { patternMust(t, m.DecRef(h)) })
	}
	r, err := NewInferenceRewriter(m)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, r.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	patternMust(t, err)
	proof, err := m.NewRef(ast.ASTHandle{})
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, m.DelRef(&proof)); patternMust(t, m.DelRef(&out)) })
	patternMust(t, r.RewriteWithProof(q, out, proof))
	got, err := m.RefGet(out)
	patternMust(t, err)
	count, err := m.GetNumPatterns(got)
	patternMust(t, err)
	if count != 1 {
		t.Fatal("proof inference pattern count", count)
	}
	p, err := m.RefGet(proof)
	patternMust(t, err)
	fact, err := m.GetProofFact(p)
	patternMust(t, err)
	want, err := m.MkIff(q, got)
	patternMust(t, err)
	if fact != want {
		t.Fatal("proof inference fact", fact, want)
	}
}

func TestPatternInferenceActualRewriterPendingPullAndResume(t *testing.T) {
	m := patternManager(t)
	b, err := m.MkBoolSort()
	patternMust(t, err)
	x, err := m.MkVar(0, b)
	patternMust(t, err)
	patternMust(t, m.IncRef(x))
	t.Cleanup(func() { patternMust(t, m.DecRef(x)) })
	truth, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpTrue, nil, nil)
	patternMust(t, err)
	inner, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: false, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(870)}, Body: truth,
	})
	patternMust(t, err)
	patternMust(t, m.IncRef(inner))
	t.Cleanup(func() { patternMust(t, m.DecRef(inner)) })
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{x, inner})
	patternMust(t, err)
	patternMust(t, m.IncRef(body))
	t.Cleanup(func() { patternMust(t, m.DecRef(body)) })
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(871)}, Body: body,
	})
	patternMust(t, err)
	patternMust(t, m.IncRef(q))
	t.Cleanup(func() { patternMust(t, m.DecRef(q)) })
	r, err := NewInferenceRewriter(m)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, r.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, m.DelRef(&out)) })
	if e := r.Rewrite(q, out); !errors.Is(e, ast.ErrASTPending) {
		t.Fatal("mixed-polarity pull branch must remain explicit", e)
	}
	options := DefaultInferenceOptions()
	options.PullQuantifiers = false
	patternMust(t, r.UpdateOptions(options))
	patternMust(t, r.Resume(out))
	got, err := m.RefGet(out)
	patternMust(t, err)
	if got != q {
		t.Fatal("resume after disabled pull should preserve unchanged quantifier", got, q)
	}
}

func TestPatternInferenceConstructorSnapshotsArithmeticBanAndLoopPolicy(t *testing.T) {
	m := patternManager(t)
	options := DefaultInferenceOptions()
	options.Arithmetic = 0
	options.BlockLoopPatterns = false
	r, err := NewInferenceRewriterWithOptions(m, options)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, r.Close()) })
	arith, err := m.ArithmeticFamilyID()
	patternMust(t, err)
	if _, banned := r.candidates.forbiddenFamilies[arith]; !banned || r.candidates.blockLoops {
		t.Fatal("source constructor must cache AP_NO and loop policy")
	}
	options.Arithmetic = 257
	options.BlockLoopPatterns = true
	options.AvoidSkolems = false
	patternMust(t, r.UpdateOptions(options))
	if _, banned := r.candidates.forbiddenFamilies[arith]; !banned || r.candidates.blockLoops ||
		r.candidates.avoidSkolems || r.options.arithmeticMode != arithmeticPatternMode(257) {
		t.Fatal("borrowed parameter refresh changed constructor-owned policy", r.options, r.candidates.blockLoops)
	}
	patternMust(t, r.Close())
	if err := r.UpdateOptions(DefaultInferenceOptions()); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed options refresh", err)
	}
}
