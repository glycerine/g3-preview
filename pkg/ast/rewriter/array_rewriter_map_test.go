package rewriter

// Source: src/ast/rewriter/array_rewriter.cpp:220-323,335-338.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func TestArrayRewriterOriginalMapStoreConstAndDecline(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	bs, e := m.MkBoolSort()
	satOK(t, e)
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	not, e := m.MkNotDecl()
	satOK(t, e)
	and, e := m.MkAndDecl()
	satOK(t, e)
	out := astTestPin(t, m, yes)
	for _, n := range []int{1, 2, 18} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			ps := make([]ast.ASTParameter, n+1)
			for i := range ps {
				ps[i], e = m.NewParameterAST(bs)
				satOK(t, e)
			}
			sort, e := m.MkFamilySort(fid, 0, ps)
			satOK(t, e)
			for i := range ps {
				satOK(t, m.DelParameter(&ps[i]))
			}
			sp, e := m.NewParameterAST(sort)
			satOK(t, e)
			defer func() { satOK(t, m.DelParameter(&sp)) }()
			constant := func(v ast.ASTHandle) ast.ASTHandle {
				h, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{sp}, []ast.ASTHandle{v})
				satOK(t, e)
				astTestPin(t, m, h)
				return h
			}
			a, b := constant(yes), constant(no)
			index := make([]ast.ASTHandle, n)
			for i := range index {
				index[i] = yes
			}
			store := func(base, v ast.ASTHandle, ix []ast.ASTHandle) ast.ASTHandle {
				args := append([]ast.ASTHandle{base}, ix...)
				args = append(args, v)
				h, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, args)
				satOK(t, e)
				astTestPin(t, m, h)
				return h
			}
			mapApp := func(f ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
				h, e := m.MkArrayMap(f, ast.SliceArguments(args))
				satOK(t, e)
				astTestPin(t, m, h)
				return h
			}
			check := func(f ast.ASTHandle, args []ast.ASTHandle, status RewriteStatus, want ast.ASTHandle) {
				t.Helper()
				term := mapApp(f, args...)
				d, e := m.GetDecl(term)
				satOK(t, e)
				satOK(t, m.SetRef(out, yes))
				st, e := rw.MkAppCore(d, ast.SliceArguments(args), out)
				satOK(t, e)
				got, e := m.RefGet(out)
				satOK(t, e)
				if st != status || got != want {
					t.Fatal("map reduction", st, status, got, want)
				}
			}
			ny, e := m.MkApp(not, []ast.ASTHandle{yes})
			satOK(t, e)
			astTestPin(t, m, ny)
			check(not, []ast.ASTHandle{a}, Rewrite2, constant(ny))
			s1, s2 := store(a, no, index), store(b, yes, index)
			nv, e := m.MkApp(not, []ast.ASTHandle{no})
			satOK(t, e)
			astTestPin(t, m, nv)
			check(not, []ast.ASTHandle{s1}, Rewrite2, store(mapApp(not, a), nv, index))
			cv, e := m.MkApp(and, []ast.ASTHandle{yes, no})
			satOK(t, e)
			astTestPin(t, m, cv)
			check(and, []ast.ASTHandle{a, b}, Rewrite2, constant(cv))
			sv, e := m.MkApp(and, []ast.ASTHandle{no, yes})
			satOK(t, e)
			astTestPin(t, m, sv)
			check(and, []ast.ASTHandle{s1, s2}, Rewrite2, store(mapApp(and, a, b), sv, index))
			check(and, []ast.ASTHandle{s1, b}, RewriteFailed, yes)
			different := append([]ast.ASTHandle(nil), index...)
			different[n-1] = no
			check(and, []ast.ASTHandle{s1, store(b, yes, different)}, RewriteFailed, yes)
			raw := mapApp(and, s1, b)
			satOK(t, rw.MkMap(and, ast.SliceArguments([]ast.ASTHandle{s1, b}), out))
			got, e := m.RefGet(out)
			satOK(t, e)
			if got != raw {
				t.Fatal("map fallback")
			}
		})
	}
}

func TestArrayRewriterOriginalMapMultiConstRangeAndUnaryRangeGuard(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	bs, e := m.MkBoolSort()
	satOK(t, e)
	other, e := m.MkUninterpretedSort(astTestSymbol(t, m, "map-output-sort"), nil)
	satOK(t, e)
	astTestPin(t, m, other)
	bp, e := m.NewParameterAST(bs)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&bp)) }()
	input, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	satOK(t, e)
	sp, e := m.NewParameterAST(input)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&sp)) }()
	yes, e := m.MkTrue()
	satOK(t, e)
	a, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{sp}, []ast.ASTHandle{yes})
	satOK(t, e)
	astTestPin(t, m, a)
	unary, e := m.MkFuncDecl(astTestSymbol(t, m, "map-change-unary"), []ast.ASTHandle{bs}, other, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	astTestPin(t, m, unary)
	binary, e := m.MkFuncDecl(astTestSymbol(t, m, "map-change-binary"), []ast.ASTHandle{bs, bs}, other, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	astTestPin(t, m, binary)
	out := astTestPin(t, m, yes)
	st, e := rw.MkMapCore(binary, ast.SliceArguments([]ast.ASTHandle{a, a}), out)
	satOK(t, e)
	if st != Rewrite2 {
		t.Fatal("multi-const status", st)
	}
	h, e := m.RefGet(out)
	satOK(t, e)
	s, e := m.GetSort(h)
	satOK(t, e)
	op, e := m.NewParameterAST(other)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&op)) }()
	expected, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, op})
	satOK(t, e)
	if s != expected {
		t.Fatal("changed multi-const range")
	}
	// Source unary branch explicitly reuses input sort, unlike its multi-array branch.
	satOK(t, m.SetRef(out, yes))
	_, e = rw.MkMapCore(unary, ast.SliceArguments([]ast.ASTHandle{a}), out)
	if !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("source unary range mismatch", e)
	}
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != yes {
		t.Fatal("failed unary output changed")
	}
	satOK(t, rw.MkMap(binary, ast.SliceArguments([]ast.ASTHandle{a, a}), out))
	want, e := m.RefGet(out)
	satOK(t, e)
	astTestPin(t, m, want)
	satOK(t, m.SetRef(out, yes))
	baseline := m.Budget().Used()
	for fault := int64(0); fault < 128; fault++ {
		m.Budget().FailAfter(fault)
		st, failure := rw.MkMapCore(binary, ast.SliceArguments([]ast.ASTHandle{a, a}), out)
		m.Budget().FailAfter(-1)
		got, e := m.RefGet(out)
		satOK(t, e)
		if failure == nil {
			if st != Rewrite2 || got != want {
				t.Fatal("map reuse", st, got, want)
			}
			return
		}
		if !errors.Is(failure, arena.ErrInjectedFailure) || got != yes || m.Budget().Used() != baseline {
			t.Fatal("map refusal/cleanup", fault, failure, got, m.Budget().Used(), baseline)
		}
	}
	t.Fatal("map refusal campaign never succeeded")
}

func TestArrayRewriterOriginalMapManyArraysBufferRefusals(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	bs, e := m.MkBoolSort()
	satOK(t, e)
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	bp, e := m.NewParameterAST(bs)
	satOK(t, e)
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	satOK(t, e)
	satOK(t, m.DelParameter(&bp))
	sp, e := m.NewParameterAST(sort)
	satOK(t, e)
	base, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{sp}, []ast.ASTHandle{no})
	satOK(t, e)
	astTestPin(t, m, base)
	satOK(t, m.DelParameter(&sp))
	stored, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, yes, yes})
	satOK(t, e)
	astTestPin(t, m, stored)
	domains := make([]ast.ASTHandle, 18)
	for i := range domains {
		domains[i] = bs
	}
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "map-eighteen-arrays"), domains, bs, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	astTestPin(t, m, f)
	out := astTestPin(t, m, yes)
	for _, input := range []ast.ASTHandle{base, stored} {
		args := make([]ast.ASTHandle, 18)
		for i := range args {
			args[i] = input
		}
		satOK(t, rw.MkMap(f, ast.SliceArguments(args), out))
		want, e := m.RefGet(out)
		satOK(t, e)
		astTestPin(t, m, want)
		satOK(t, m.SetRef(out, yes))
		baseline := m.Budget().Used()
		success := false
		for fault := int64(0); fault < 256; fault++ {
			m.Budget().FailAfter(fault)
			st, failure := rw.MkMapCore(f, ast.SliceArguments(args), out)
			m.Budget().FailAfter(-1)
			got, e := m.RefGet(out)
			satOK(t, e)
			if failure == nil {
				if st != Rewrite2 || got != want {
					t.Fatal("many-array map result", st, got, want)
				}
				success = true
				break
			}
			if !errors.Is(failure, arena.ErrInjectedFailure) || got != yes || m.Budget().Used() != baseline {
				t.Fatal("many-array map refusal/cleanup", fault, failure, got, m.Budget().Used(), baseline)
			}
		}
		if !success {
			t.Fatal("many-array map campaign never succeeded")
		}
	}
}
