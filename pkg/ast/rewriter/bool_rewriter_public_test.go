// Source: src/ast/rewriter/bool_rewriter.h:86-105;
// src/model/model_evaluator.cpp:66-93,146-169.
package rewriter

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

type publicBooleanConfiguration struct {
	DefaultConfiguration
	manager *ast.ASTManager
	reducer *BoolRewriter
}

func (c *publicBooleanConfiguration) ReduceApp(d ast.ASTHandle, args ast.AppArguments, out, proof ast.ASTRef) (RewriteStatus, error) {
	if e := c.manager.ResetRef(proof); e != nil {
		return RewriteFailed, e
	}
	fid, e := c.manager.GetFamilyID(d)
	if e != nil || fid != 0 {
		return RewriteFailed, e
	}
	return c.reducer.MkAppCore(d, args, out)
}
func TestBoolRewriterPublicCoreThroughOriginalTraversal(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	key := "flat"
	satOK(t, c.SetBoolName(p, &key, false))
	rw, e := NewBoolRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	flat, e := rw.Flat()
	satOK(t, e)
	if flat {
		t.Fatal("local parameter ignored")
	}
	satOK(t, rw.SetFlat(true)) // original evaluator overrides constructor parameters
	a, b := goalTestAtom(t, m, "public-bool-a"), goalTestAtom(t, m, "public-bool-b")
	yes, e := m.MkTrue()
	satOK(t, e)
	term := goalTestApp(t, m, ast.ASTOpAnd, yes, a, goalTestApp(t, m, ast.ASTOpAnd, b, a))
	want := goalTestApp(t, m, ast.ASTOpAnd, a, b)
	out := astTestPin(t, m, ast.ASTHandle{})
	cfg := publicBooleanConfiguration{manager: m, reducer: rw}
	traversal, e := NewTraversal(m, &cfg)
	satOK(t, e)
	defer func() { satOK(t, traversal.Close()) }()
	satOK(t, traversal.Rewrite(term, out))
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != want {
		t.Fatal("original Boolean nested reduction", got, want)
	}
	// A core miss preserves the caller's output; traversal reconstructs the app.
	satOK(t, m.SetRef(out, yes))
	unchanged := goalTestApp(t, m, ast.ASTOpAnd, a, b)
	decl, e := m.GetDecl(unchanged)
	satOK(t, e)
	status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{a, b}), out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != RewriteFailed || got != yes {
		t.Fatal("core miss changed output", status, got)
	}
	copyOwner := *rw
	if _, e = copyOwner.Flat(); e != util.ErrParamsOwner {
		t.Fatal("copied reducer", e)
	}
	satOK(t, rw.Close())
	if _, e = rw.MkAppCore(ast.ASTHandle{}, ast.SliceArguments(nil), out); e != util.ErrParamsOwner {
		t.Fatal("closed reducer", e)
	}
}
