package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func theoryRewriterFixture(t *testing.T) (*ast.ASTManager, *TheoryRewriter) {
	t.Helper()
	m, context, globals, local := boolRewriterConfigFixture(t)
	r, err := NewTheoryRewriter(m, context, globals, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	return m, r
}

func TestTheoryRewriterCoreTraversalAcrossFamilies(t *testing.T) {
	m, r := theoryRewriterFixture(t)
	two := arithRewriterNumeral(t, m, "2", true)
	three := arithRewriterNumeral(t, m, "3", true)
	sum, err := m.MkFamilyApp(r.configuration.arithmetic.family, arithOpAdd, nil, []ast.ASTHandle{two, three})
	astTestMust(t, err)
	out := astTestPin(t, m, sum)
	astTestMust(t, r.Rewrite(sum, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	arithRewriterWantNumeral(t, m, got, "5", true)

	a := goalTestAtom(t, m, "theory-pb-a")
	b := goalTestAtom(t, m, "theory-pb-b")
	constraint, err := m.MkPseudoBooleanAtLeast([]ast.ASTHandle{a, b}, 1)
	astTestMust(t, err)
	astTestMust(t, r.Rewrite(constraint, out))
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(got); e != nil || kind != ast.ASTOpOr {
		t.Fatal("pseudo-Boolean family dispatch", kind, e)
	}
	if steps, e := r.NumSteps(); e != nil || steps == 0 {
		t.Fatal("theory traversal steps", steps, e)
	}
}

func TestTheoryRewriterTamagotchiAndRawMakeApp(t *testing.T) {
	m, r := theoryRewriterFixture(t)
	sort, err := m.BitVectorSort(1)
	astTestMust(t, err)
	x, err := m.MkVar(0, sort)
	astTestMust(t, err)
	one, err := m.BitVectorNumeralUnsigned(1, 1)
	astTestMust(t, err)
	equality, err := m.MkEq(x, one)
	astTestMust(t, err)
	condition := goalTestAtom(t, m, "theory-tamagotchi")
	iff, err := m.MkIff(equality, condition)
	astTestMust(t, err)
	decl, err := m.GetDecl(iff)
	astTestMust(t, err)
	out := astTestPin(t, m, iff)
	status, err := r.configuration.reduceAppCore(decl, ast.SliceArguments([]ast.ASTHandle{equality, condition}), out)
	astTestMust(t, err)
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if status != Rewrite2 {
		t.Fatal("tamagotchi status", status)
	}
	if kind, e := m.GetDeclKind(got); e != nil || kind != ast.ASTOpEq {
		t.Fatal("tamagotchi equality", kind, e)
	}
	right, err := m.GetArg(got, 1)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(right); e != nil || kind != ast.ASTOpIte {
		t.Fatal("tamagotchi ITE", kind, e)
	}

	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	name := astTestSymbol(t, m, "theory-raw")
	info, err := m.NewFuncDeclInfo(ast.NullFamilyID, -1, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, err)
	rawDecl, err := m.MkFuncDecl(name, nil, boolSort, info)
	astTestMust(t, err)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	want, err := m.MkApp(rawDecl, nil)
	astTestMust(t, err)
	astTestMust(t, r.MakeApp(rawDecl, ast.AppArguments{}, out))
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if got != want {
		t.Fatal("raw make-app fallback", got, want)
	}
}

func TestTheoryRewriterPullCheapITE(t *testing.T) {
	m, context, globals, local := boolRewriterConfigFixture(t)
	key := "pull_cheap_ite"
	astTestMust(t, context.SetBoolName(local, &key, true))
	r, err := NewTheoryRewriter(m, context, globals, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })

	condition := goalTestAtom(t, m, "theory-pull-ite")
	two := arithRewriterNumeral(t, m, "2", true)
	three := arithRewriterNumeral(t, m, "3", true)
	choice, err := m.MkIte(condition, two, three)
	astTestMust(t, err)
	equality, err := m.MkEq(choice, two)
	astTestMust(t, err)
	decl, err := m.GetDecl(equality)
	astTestMust(t, err)
	out := astTestPin(t, m, equality)
	makeOut := astTestPin(t, m, equality)
	astTestMust(t, r.MakeApp(decl, ast.SliceArguments([]ast.ASTHandle{choice, two}), makeOut))
	made, err := m.RefGet(makeOut)
	astTestMust(t, err)
	if made != condition {
		t.Fatal("MakeApp skipped configured ITE pulling", made, condition)
	}

	astTestMust(t, r.Rewrite(equality, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != condition {
		t.Fatal("pulled equality did not simplify to condition", got, condition)
	}

}

func TestTheoryRewriterPushITE(t *testing.T) {
	t.Run("arithmetic", func(t *testing.T) {
		m, context, globals, local := boolRewriterConfigFixture(t)
		key := "push_ite_arith"
		astTestMust(t, context.SetBoolName(local, &key, true))
		r, err := NewTheoryRewriter(m, context, globals, local)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, r.Close()) })

		condition := goalTestAtom(t, m, "theory-push-arith")
		two := arithRewriterNumeral(t, m, "2", true)
		three := arithRewriterNumeral(t, m, "3", true)
		sort, err := m.GetSort(two)
		astTestMust(t, err)
		x, err := m.MkVar(0, sort)
		astTestMust(t, err)
		left, err := m.MkFamilyApp(r.configuration.arithmetic.family, arithOpAdd, nil, []ast.ASTHandle{two, x})
		astTestMust(t, err)
		right, err := m.MkFamilyApp(r.configuration.arithmetic.family, arithOpAdd, nil, []ast.ASTHandle{three, x})
		astTestMust(t, err)
		choice, err := m.MkIte(condition, left, right)
		astTestMust(t, err)
		decl, err := m.GetDecl(choice)
		astTestMust(t, err)
		out := astTestPin(t, m, choice)
		astTestMust(t, r.MakeApp(decl, ast.SliceArguments([]ast.ASTHandle{condition, left, right}), out))
		got, err := m.RefGet(out)
		astTestMust(t, err)
		inner, err := m.MkIte(condition, two, three)
		astTestMust(t, err)
		want, err := m.MkFamilyApp(r.configuration.arithmetic.family, arithOpAdd, nil, []ast.ASTHandle{inner, x})
		astTestMust(t, err)
		if got != want {
			t.Fatal("arithmetic ITE was not pushed", got, want)
		}
	})

	t.Run("bit-vector", func(t *testing.T) {
		m, context, globals, local := boolRewriterConfigFixture(t)
		key := "push_ite_bv"
		astTestMust(t, context.SetBoolName(local, &key, true))
		r, err := NewTheoryRewriter(m, context, globals, local)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, r.Close()) })

		condition := goalTestAtom(t, m, "theory-push-bv")
		sort, err := m.BitVectorSort(8)
		astTestMust(t, err)
		x, err := m.MkVar(0, sort)
		astTestMust(t, err)
		two, err := m.BitVectorNumeralUnsigned(2, 8)
		astTestMust(t, err)
		three, err := m.BitVectorNumeralUnsigned(3, 8)
		astTestMust(t, err)
		left, err := m.MkFamilyApp(r.configuration.bitVector.bvFamily, theoryBVOpAdd, nil, []ast.ASTHandle{two, x})
		astTestMust(t, err)
		right, err := m.MkFamilyApp(r.configuration.bitVector.bvFamily, theoryBVOpAdd, nil, []ast.ASTHandle{x, three})
		astTestMust(t, err)
		choice, err := m.MkIte(condition, left, right)
		astTestMust(t, err)
		decl, err := m.GetDecl(choice)
		astTestMust(t, err)
		out := astTestPin(t, m, choice)
		astTestMust(t, r.MakeApp(decl, ast.SliceArguments([]ast.ASTHandle{condition, left, right}), out))
		got, err := m.RefGet(out)
		astTestMust(t, err)
		inner, err := m.MkIte(condition, two, three)
		astTestMust(t, err)
		want, err := m.MkFamilyApp(r.configuration.bitVector.bvFamily, theoryBVOpAdd, nil, []ast.ASTHandle{inner, x})
		astTestMust(t, err)
		if got != want {
			t.Fatal("bit-vector commutative ITE was not pushed", got, want)
		}
	})
}

func TestTheoryRewriterSubstitutionDependencies(t *testing.T) {
	m, r := theoryRewriterFixture(t)
	a := goalTestAtom(t, m, "theory-subst-a")
	b := goalTestAtom(t, m, "theory-subst-b")
	decl := rewriterTestDecl(t, m, "theory-subst-f")
	term := rewriterTestApp(t, m, decl, a)
	want := rewriterTestApp(t, m, decl, b)

	dependency, err := m.DependencyLeaf(b)
	astTestMust(t, err)
	astTestMust(t, m.DependencyInc(dependency))
	t.Cleanup(func() { astTestMust(t, m.DependencyDec(dependency)) })
	substitution, err := m.NewExprSubstitutionPolicy(true, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelExprSubstitution(&substitution)) })
	astTestMust(t, m.ExprSubstitutionInsert(substitution, a, b, ast.ASTHandle{}, dependency))
	astTestMust(t, r.SetSubstitution(&substitution))

	out := astTestPin(t, m, term)
	astTestMust(t, r.Rewrite(term, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != want {
		t.Fatal("theory substitution result", got, want)
	}
	used, err := r.UsedDependencies()
	astTestMust(t, err)
	if used.IsNull() {
		t.Fatal("missing used substitution dependency")
	}
	contains, err := m.DependencyContains(used, b)
	astTestMust(t, err)
	if !contains {
		t.Fatal("used dependency omitted substitution dependency")
	}

	astTestMust(t, r.ResetUsedDependencies())
	used, err = r.UsedDependencies()
	astTestMust(t, err)
	if !used.IsNull() {
		t.Fatal("used dependencies survived reset")
	}
	astTestMust(t, m.SetRef(out, term))
	astTestMust(t, r.Rewrite(term, out))
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if got != want {
		t.Fatal("dependency reset discarded substitution", got, want)
	}

	astTestMust(t, r.SetSubstitution(nil))
	astTestMust(t, m.SetRef(out, term))
	astTestMust(t, r.Rewrite(term, out))
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if got != term {
		t.Fatal("nil substitution remained installed", got, term)
	}
}

func TestTheoryRewriterParamsCacheCleanupAndBindings(t *testing.T) {
	m, context, globals, local := boolRewriterConfigFixture(t)
	r, err := NewTheoryRewriter(m, context, globals, local)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	for _, key := range []string{"pull_cheap_ite", "cache_all", "elim_and", "mul_to_power", "bv_sort_ac", "sort_store", "hi_fp_unspecified"} {
		astTestMust(t, context.SetBoolName(local, &key, true))
	}
	astTestMust(t, r.UpdateParams(local))
	bvState, err := r.configuration.bitVector.State()
	astTestMust(t, err)
	if !r.configuration.pullCheapITE || !r.configuration.cacheAll || !r.configuration.boolean.elimAnd ||
		!r.configuration.arithmetic.mulToPower || !bvState.SortAC || !r.configuration.array.sortStore ||
		!r.configuration.floating.hiFPUnspecified {
		t.Fatal("parameter update did not reach every configurable subrewriter")
	}

	a := goalTestAtom(t, m, "theory-cache-a")
	decl := rewriterTestDecl(t, m, "theory-cache-f")
	inner := rewriterTestApp(t, m, decl, a)
	term := rewriterTestApp(t, m, decl, inner)
	out := astTestPin(t, m, term)
	astTestMust(t, r.Rewrite(term, out))
	if size, e := r.CacheSize(); e != nil || size == 0 {
		t.Fatal("theory rewrite cache", size, e)
	}

	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	astTestMust(t, r.RewriteWithBindings(variable, []ast.ASTHandle{a}, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != a {
		t.Fatal("theory inverse binding rewrite", got, a)
	}

	astTestMust(t, r.Cleanup())
	if size, e := r.CacheSize(); e != nil || size != 0 {
		t.Fatal("cleanup retained rewrite cache", size, e)
	}
	if !r.configuration.pullCheapITE || !r.configuration.floating.hiFPUnspecified {
		t.Fatal("cleanup did not preserve owned parameters")
	}
}

func TestTheoryRewriterCloseGuards(t *testing.T) {
	m, r := theoryRewriterFixture(t)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	out := astTestPin(t, m, truth)
	astTestMust(t, r.Close())
	if err = r.Rewrite(truth, out); err != ast.ErrASTHandle {
		t.Fatal("closed theory rewriter accepted traversal", err)
	}
}
