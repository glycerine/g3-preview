package rewriter

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBoolRewriterNativeLocalContextGrowthAndReferences(t *testing.T) {
	m := astTestManager(t)
	args := make([]ast.ASTHandle, 257)
	counts := make([]uint32, len(args))
	for i := range args {
		args[i] = goalTestAtom(t, m, fmt.Sprintf("native-local-%d", i))
		var e error
		counts[i], e = m.GetRefCount(args[i])
		satOK(t, e)
	}
	truth, e := m.MkTrue()
	satOK(t, e)
	out := astTestPin(t, m, truth)
	rw := boolRewriter{manager: m, flat: true}
	// Warm only the actual temporary AST-reference owner. Argument slices/maps
	// are released on every invocation, so the following backing growth is cold.
	changed, e := rw.localContextSimplify(args, out)
	satOK(t, e)
	if changed {
		t.Fatal("unrelated atoms changed")
	}
	used := m.Budget().Used()
	rw.localContextCost = 0
	m.Budget().FailAfter(0)
	changed, e = rw.localContextSimplify(args, out)
	m.Budget().FailAfter(-1)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if changed || got != truth || rw.localContextCost != 10*uint32(len(args)) {
		t.Fatal("unchanged native context", changed, got, rw.localContextCost)
	}
	if rw.localActive || rw.localArg != (ast.ASTRef{}) || rw.localOld != nil || rw.localNew != nil {
		t.Fatal("local roots survived completed cleanup")
	}
	if m.Budget().Used() != used {
		t.Fatal("native context backing charged arena")
	}
	for i, h := range args {
		count, e := m.GetRefCount(h)
		satOK(t, e)
		if count != counts[i] {
			t.Fatal("local context reference leak", i, count, counts[i])
		}
	}
}

func TestBoolRewriterNativeMembershipLeavesForeignMarks(t *testing.T) {
	m := astTestManager(t)
	p, q := goalTestAtom(t, m, "native-mark-p"), goalTestAtom(t, m, "native-mark-q")
	expected := goalTestApp(t, m, ast.ASTOpOr, p, q)
	truth, e := m.MkTrue()
	satOK(t, e)
	out := astTestPin(t, m, truth)
	neg, e := ast.NewASTFastMark1(m)
	satOK(t, e)
	defer func() { satOK(t, neg.Close()) }()
	pos, e := ast.NewASTFastMark2(m)
	satOK(t, e)
	defer func() { satOK(t, pos.Close()) }()
	satOK(t, neg.Mark(p, true))
	satOK(t, pos.Mark(q, true))
	rw := boolRewriter{manager: m, flat: true}
	status, e := rw.mkNFlatOrCore([]ast.ASTHandle{p, q, p}, out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != brDone || got != expected {
		t.Fatal("foreign marks influenced native duplicate detection", status, got, expected)
	}
	satOK(t, m.SetRef(out, truth))
	changed, e := rw.localContextSimplify([]ast.ASTHandle{p, q}, out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if changed || got != truth {
		t.Fatal("foreign marks influenced local context", changed, got)
	}
	// All three arguments are different uninterpreted constants, so DISTINCT remains
	// unreduced. A foreign mark must not turn its first argument into a duplicate.
	u, e := m.MkUninterpretedSort(astTestSymbol(t, m, "native-distinct-sort"), nil)
	satOK(t, e)
	astTestPin(t, m, u)
	var values []ast.ASTHandle
	for i := 0; i < 3; i++ {
		h, e := m.MkFreshConst("native-distinct", u)
		satOK(t, e)
		astTestPin(t, m, h)
		values = append(values, h)
	}
	satOK(t, neg.Mark(values[0], true))
	satOK(t, m.SetRef(out, truth))
	status, e = rw.mkDistinctCore(values, out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != brFailed || got != truth {
		t.Fatal("foreign mark influenced DISTINCT", status, got)
	}
	for _, h := range []ast.ASTHandle{p, values[0]} {
		marked, e := neg.IsMarked(h)
		satOK(t, e)
		if !marked {
			t.Fatal("native rewrite cleared foreign plane one")
		}
	}
	marked, e := pos.IsMarked(q)
	satOK(t, e)
	if !marked {
		t.Fatal("native rewrite cleared foreign plane two")
	}
}
