// Source: src/ast/rewriter/array_rewriter.cpp:340-388.
package rewriter

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestArrayRewriterOriginalSetStatusesAndSourceTrees(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	bs, e := m.MkBoolSort()
	satOK(t, e)
	s, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bs}), bs)
	satOK(t, e)
	astTestPin(t, m, s)
	full, e := m.MkFullSet(s)
	satOK(t, e)
	astTestPin(t, m, full)
	empty, e := m.MkEmptySet(s)
	satOK(t, e)
	astTestPin(t, m, empty)
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	or, e := m.MkOrDecl()
	satOK(t, e)
	and, e := m.MkAndDecl()
	satOK(t, e)
	not, e := m.MkNotDecl()
	satOK(t, e)
	mapApp := func(f ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkArrayMap(f, ast.SliceArguments(args))
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	difference := mapApp(and, full, mapApp(not, empty))
	subset, e := m.MkEq(difference, empty)
	satOK(t, e)
	astTestPin(t, m, subset)
	notFalse, e := m.MkApp(not, []ast.ASTHandle{no})
	satOK(t, e)
	comp, e := m.MkConstArray(s, notFalse)
	satOK(t, e)
	astTestPin(t, m, comp)
	out := astTestPin(t, m, yes)
	for _, row := range []struct {
		op     int32
		args   []ast.ASTHandle
		status RewriteStatus
		want   ast.ASTHandle
	}{
		{ast.ASTArrayOpSetUnion, []ast.ASTHandle{full}, RewriteDone, full},
		{ast.ASTArrayOpSetIntersect, []ast.ASTHandle{empty}, RewriteDone, empty},
		{ast.ASTArrayOpSetUnion, []ast.ASTHandle{full, empty}, Rewrite1, mapApp(or, full, empty)},
		{ast.ASTArrayOpSetIntersect, []ast.ASTHandle{full, empty}, Rewrite1, mapApp(and, full, empty)},
		{ast.ASTArrayOpSetComplement, []ast.ASTHandle{empty}, Rewrite2, comp},
		{ast.ASTArrayOpSetDifference, []ast.ASTHandle{full, empty}, Rewrite2, difference},
		{ast.ASTArrayOpSetSubset, []ast.ASTHandle{full, empty}, Rewrite3, subset},
	} {
		term, e := m.MkFamilyApp(fid, row.op, nil, row.args)
		satOK(t, e)
		astTestPin(t, m, term)
		d, e := m.GetDecl(term)
		satOK(t, e)
		satOK(t, m.SetRef(out, yes))
		st, e := rw.MkAppCore(d, ast.SliceArguments(row.args), out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if st != row.status || got != row.want {
			t.Fatal("set reduction", row.op, st, row.status, got, row.want)
		}
		satOK(t, m.SetRef(out, yes))
		switch row.op {
		case ast.ASTArrayOpSetUnion:
			st, e = rw.MkSetUnion(ast.SliceArguments(row.args), out)
		case ast.ASTArrayOpSetIntersect:
			st, e = rw.MkSetIntersect(ast.SliceArguments(row.args), out)
		case ast.ASTArrayOpSetComplement:
			st, e = rw.MkSetComplement(row.args[0], out)
		case ast.ASTArrayOpSetDifference:
			st, e = rw.MkSetDifference(row.args[0], row.args[1], out)
		case ast.ASTArrayOpSetSubset:
			st, e = rw.MkSetSubset(row.args[0], row.args[1], out)
		}
		satOK(t, e)
		got, e = m.RefGet(out)
		satOK(t, e)
		if st != row.status || got != row.want {
			t.Fatal("public set method", row.op, st, got, row.want)
		}
	}
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "unreduced-set"), nil, s, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	a, e := m.MkApp(d, nil)
	satOK(t, e)
	astTestPin(t, m, a)
	term, e := m.MkFamilyApp(fid, ast.ASTArrayOpSetComplement, nil, []ast.ASTHandle{a})
	satOK(t, e)
	astTestPin(t, m, term)
	cd, e := m.GetDecl(term)
	satOK(t, e)
	satOK(t, m.SetRef(out, yes))
	st, e := rw.MkAppCore(cd, ast.SliceArguments([]ast.ASTHandle{a}), out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if st != RewriteFailed || got != yes {
		t.Fatal("bare-set complement must decline")
	}
	// Actual associative set applications are binary trees. A direct three-arg
	// core call reaches the original map factory's exact Boolean function arity.
	union, e := m.MkFamilyApp(fid, ast.ASTArrayOpSetUnion, nil, []ast.ASTHandle{full, empty})
	satOK(t, e)
	ud, e := m.GetDecl(union)
	satOK(t, e)
	_, e = rw.MkAppCore(ud, ast.SliceArguments([]ast.ASTHandle{full, empty, full}), out)
	if !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("direct variadic map guard", e)
	}
}

func TestArrayRewriterOriginalSetSubsetFailurePublicationAndReuse(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	bs, e := m.MkBoolSort()
	satOK(t, e)
	s, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bs}), bs)
	satOK(t, e)
	full, e := m.MkFullSet(s)
	satOK(t, e)
	astTestPin(t, m, full)
	empty, e := m.MkEmptySet(s)
	satOK(t, e)
	astTestPin(t, m, empty)
	term, e := m.MkFamilyApp(fid, ast.ASTArrayOpSetSubset, nil, []ast.ASTHandle{full, empty})
	satOK(t, e)
	astTestPin(t, m, term)
	d, e := m.GetDecl(term)
	satOK(t, e)
	yes, e := m.MkTrue()
	satOK(t, e)
	out := astTestPin(t, m, yes)
	args := ast.SliceArguments([]ast.ASTHandle{full, empty})
	_, e = rw.MkAppCore(d, args, out)
	satOK(t, e)
	want, e := m.RefGet(out)
	satOK(t, e)
	astTestPin(t, m, want)
	prefix, e := m.GetArg(want, 0)
	satOK(t, e)
	astTestPin(t, m, prefix)
	satOK(t, m.SetRef(out, yes))
	baseline := m.Budget().Used()
	for fault := int64(0); fault < 128; fault++ {
		m.Budget().FailAfter(fault)
		st, failure := rw.MkAppCore(d, args, out)
		m.Budget().FailAfter(-1)
		got, e := m.RefGet(out)
		satOK(t, e)
		if failure == nil {
			if st != Rewrite3 || got != want {
				t.Fatal("subset result", st, got, want)
			}
			return
		}
		if !errors.Is(failure, arena.ErrInjectedFailure) {
			t.Fatal("subset refusal", fault, failure)
		}
		if got != prefix && got != yes {
			t.Fatal("failure published unexpected prefix", got)
		}
		satOK(t, m.SetRef(out, yes))
		if m.Budget().Used() != baseline {
			t.Fatal("subset refusal leak", fault, m.Budget().Used(), baseline)
		}
	}
	t.Fatal("subset refusal campaign never succeeded")
}
