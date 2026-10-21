package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func pb2BVFixture(t *testing.T) (*ast.ASTManager, *PB2BVRewriter) {
	t.Helper()
	m := astTestManager(t)
	r, err := NewPB2BVRewriter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	return m, r
}

func TestPB2BVCardinalityTraversalAndOrShortcut(t *testing.T) {
	m, r := pb2BVFixture(t)
	a := goalTestAtom(t, m, "pb2bv-a")
	b := goalTestAtom(t, m, "pb2bv-b")
	c := goalTestAtom(t, m, "pb2bv-c")

	atLeastOne, err := m.MkPseudoBooleanAtLeast([]ast.ASTHandle{a, b, c}, 1)
	astTestMust(t, err)
	out := astTestPin(t, m, atLeastOne)
	astTestMust(t, r.Rewrite(atLeastOne, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if family, e := m.GetFamilyID(got); e != nil || family != m.BasicFamilyID() {
		t.Fatal("at-least-one OR shortcut", family, e)
	}
	if kind, e := m.GetDeclKind(got); e != nil || kind != ast.ASTOpOr {
		t.Fatal("at-least-one result", kind, e)
	}

	atMostOne, err := m.MkPseudoBooleanAtMost([]ast.ASTHandle{a, b, c}, 1)
	astTestMust(t, err)
	astTestMust(t, r.Rewrite(atMostOne, out))
	got, err = m.RefGet(out)
	astTestMust(t, err)
	if got == atMostOne {
		t.Fatal("sorting-network cardinality was not translated")
	}
	if family, e := m.GetFamilyID(got); e != nil || family == r.pseudo.family {
		t.Fatal("PB result remained in PB family", family, e)
	}
	if translated, e := r.NumTranslated(); e != nil || translated != 2 {
		t.Fatal("translated count", translated, e)
	}
	if steps, e := r.NumSteps(); e != nil || steps == 0 {
		t.Fatal("traversal steps", steps, e)
	}
	clauses, err := r.FlushSideConstraints()
	astTestMust(t, err)
	for _, clause := range clauses {
		astTestMust(t, m.DecRef(clause))
	}
}

func TestPB2BVWeightedSliceRemainsVisibleAndScopeGuards(t *testing.T) {
	m, r := pb2BVFixture(t)
	a := goalTestAtom(t, m, "pb2bv-weight-a")
	b := goalTestAtom(t, m, "pb2bv-weight-b")
	values := pbTestRationals(t, m, "2", "3", "4")
	weighted, err := m.MkPseudoBooleanGE(values[:2], []ast.ASTHandle{a, b}, &values[2])
	astTestMust(t, err)
	out := astTestPin(t, m, a)
	astTestMust(t, r.Rewrite(weighted, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != weighted {
		t.Fatal("unported weighted slice was hidden", got, weighted)
	}
	if translated, e := r.NumTranslated(); e != nil || translated != 0 {
		t.Fatal("weighted refusal count", translated, e)
	}

	astTestMust(t, r.Push())
	astTestMust(t, r.context.Clause(0, ast.ASTLiteralVectorSpan{}))
	if err = r.Pop(1); err != ast.ErrASTDomain {
		t.Fatal("pop accepted unflushed clauses", err)
	}
	clauses, err := r.FlushSideConstraints()
	astTestMust(t, err)
	for _, clause := range clauses {
		astTestMust(t, m.DecRef(clause))
	}
	astTestMust(t, r.Pop(1))
	if err = r.Pop(1); err != ast.ErrASTDomain {
		t.Fatal("pop accepted missing scope", err)
	}
}

func TestPB2BVCloseGuards(t *testing.T) {
	m, r := pb2BVFixture(t)
	a := goalTestAtom(t, m, "pb2bv-close")
	out := astTestPin(t, m, a)
	astTestMust(t, r.Close())
	if err := r.Rewrite(a, out); err != ast.ErrASTHandle {
		t.Fatal("closed rewriter accepted traversal", err)
	}
}
