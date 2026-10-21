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
