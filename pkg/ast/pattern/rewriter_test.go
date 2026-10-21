package pattern

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func patternDatabaseStoreQuantifier(t *testing.T, m *ast.ASTManager, annotated bool) (ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	arith, err := m.RegisterArithmeticPlugin()
	patternMust(t, err)
	integer, err := m.MkFamilySort(arith, 1, nil)
	patternMust(t, err)
	decl := func(name string, arity int) ast.ASTHandle {
		t.Helper()
		s, e := m.Symbols.Intern(name)
		patternMust(t, e)
		domain := make([]ast.ASTHandle, arity)
		for i := range domain {
			domain[i] = integer
		}
		d, e := m.MkFuncDecl(s, domain, integer, ast.ASTFuncDeclInfo{})
		patternMust(t, e)
		patternMust(t, m.IncRef(d))
		t.Cleanup(func() { patternMust(t, m.DecRef(d)) })
		return d
	}
	store, selectDecl := decl("actual-store", 3), decl("actual-select", 2)
	variables := make([]ast.ASTHandle, 3)
	for i := range variables {
		variables[i], err = m.MkVar(uint32(2-i), integer)
		patternMust(t, err)
	}
	stored, err := m.MkApp(store, variables)
	patternMust(t, err)
	selected, err := m.MkApp(selectDecl, []ast.ASTHandle{stored, variables[1]})
	patternMust(t, err)
	body, err := m.MkEq(selected, variables[2])
	patternMust(t, err)
	var patterns []ast.ASTHandle
	var originalPattern ast.ASTHandle
	if annotated {
		originalPattern, err = m.MkPattern([]ast.ASTHandle{selected})
		patternMust(t, err)
		patterns = []ast.ASTHandle{originalPattern}
	}
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{integer, integer, integer},
		Names: []symbol.Symbol{symbol.Numerical(901), symbol.Numerical(902), symbol.Numerical(903)},
		Body:  body, Weight: 8, Patterns: patterns,
	})
	patternMust(t, err)
	patternMust(t, m.IncRef(q))
	t.Cleanup(func() { patternMust(t, m.DecRef(q)) })
	return q, stored, originalPattern
}

func TestPatternInferenceDatabaseTraversal(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "proof"}[proofs], func(t *testing.T) {
			m := patternManager(t)
			if proofs {
				patternMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
			}
			q, stored, _ := patternDatabaseStoreQuantifier(t, m, false)
			opts := DefaultInferenceOptions()
			opts.UseDatabase = true
			r, err := NewInferenceRewriterWithOptions(m, opts)
			patternMust(t, err)
			t.Cleanup(func() { patternMust(t, r.Close()) })
			out, err := m.NewRef(ast.ASTHandle{})
			patternMust(t, err)
			t.Cleanup(func() { patternMust(t, m.DelRef(&out)) })
			var proof ast.ASTRef
			if proofs {
				proof, err = m.NewRef(ast.ASTHandle{})
				patternMust(t, err)
				t.Cleanup(func() { patternMust(t, m.DelRef(&proof)) })
				patternMust(t, r.RewriteWithProof(q, out, proof))
			} else {
				patternMust(t, r.Rewrite(q, out))
			}
			got, err := m.RefGet(out)
			patternMust(t, err)
			weight, err := m.GetQuantifierWeight(got)
			patternMust(t, err)
			p, err := m.GetPattern(got, 0)
			patternMust(t, err)
			term, err := m.GetArg(p, 0)
			patternMust(t, err)
			if weight != 0 || term != stored {
				t.Fatal("database trigger and weight", term, stored, weight)
			}
			if proofs {
				pr, e := m.RefGet(proof)
				patternMust(t, e)
				fact, e := m.GetProofFact(pr)
				patternMust(t, e)
				want, e := m.MkIff(q, got)
				patternMust(t, e)
				if fact != want {
					t.Fatal("database rewrite proof", fact, want)
				}
			}
		})
	}
	m := patternManager(t)
	q, _, original := patternDatabaseStoreQuantifier(t, m, true)
	opts := DefaultInferenceOptions()
	opts.UseDatabase = true
	r, err := NewInferenceRewriterWithOptions(m, opts)
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, r.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	patternMust(t, err)
	t.Cleanup(func() { patternMust(t, m.DelRef(&out)) })
	patternMust(t, r.Rewrite(q, out))
	got, err := m.RefGet(out)
	patternMust(t, err)
	p, err := m.GetPattern(got, 0)
	patternMust(t, err)
	weight, err := m.GetQuantifierWeight(got)
	patternMust(t, err)
	if p != original || weight != 0 {
		t.Fatal("database weight-only update", p, original, weight)
	}
}

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
