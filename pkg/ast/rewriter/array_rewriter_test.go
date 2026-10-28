package rewriter

// Source: src/ast/rewriter/array_rewriter.cpp:70-224.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"

	uparams "github.com/glycerine/g3/pkg/util/params"
	"testing"
)

func TestArrayRewriterOriginalStoreSelectAndFirstUncertainIndex(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	bs, e := m.MkBoolSort()
	satOK(t, e)
	param, e := m.NewParameterAST(bs)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&param)) }()
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{param, param, param})
	satOK(t, e)
	sym, e := m.Symbols.Intern("array-core-base")
	satOK(t, e)
	decl, e := m.MkFuncDecl(sym, nil, sort, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	base, e := m.MkApp(decl, nil)
	satOK(t, e)
	astTestPin(t, m, base)
	x := goalTestAtom(t, m, "array-core-x")
	y := goalTestAtom(t, m, "array-core-y")
	makeApp := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, op, nil, args)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	store := makeApp(ast.ASTArrayOpStore, base, x, yes, no)
	out := astTestPin(t, m, yes)
	check := func(op int32, args []ast.ASTHandle, status RewriteStatus, want ast.ASTHandle) {
		t.Helper()
		term := makeApp(op, args...)
		d, e := m.GetDecl(term)
		satOK(t, e)
		gotStatus, e := rw.MkAppCore(d, ast.SliceArguments(args), out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if gotStatus != status || got != want {
			t.Fatal("rewrite", op, gotStatus, status, got, want)
		}
	}
	check(ast.ASTArrayOpSelect, []ast.ASTHandle{store, x, yes}, RewriteDone, no)
	distinctWant := makeApp(ast.ASTArrayOpSelect, base, x, no)
	check(ast.ASTArrayOpSelect, []ast.ASTHandle{store, x, no}, Rewrite1, distinctWant)
	satOK(t, m.SetRef(out, yes))
	check(ast.ASTArrayOpSelect, []ast.ASTHandle{store, y, no}, RewriteFailed, yes)
	satOK(t, rw.SetExpandSelectStore(true))
	eq1, e := m.MkEq(x, y)
	satOK(t, e)
	eq2, e := m.MkEq(yes, no)
	satOK(t, e)
	cond, e := m.MkAndArgs([]ast.ASTHandle{eq1, eq2})
	satOK(t, e)
	fallback := makeApp(ast.ASTArrayOpSelect, base, y, no)
	expanded, e := m.MkIte(cond, no, fallback)
	satOK(t, e)
	astTestPin(t, m, expanded)
	check(ast.ASTArrayOpSelect, []ast.ASTHandle{store, y, no}, Rewrite3, expanded)
	overwrite := makeApp(ast.ASTArrayOpStore, base, x, yes, yes)
	check(ast.ASTArrayOpStore, []ast.ASTHandle{store, x, yes, yes}, RewriteDone, overwrite)
	selected := makeApp(ast.ASTArrayOpSelect, base, x, yes)
	check(ast.ASTArrayOpStore, []ast.ASTHandle{base, x, yes, selected}, RewriteDone, base)
}

func TestArrayRewriterOriginalSortingConstantAndIteRules(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	bs, e := m.MkBoolSort()
	satOK(t, e)
	bp, e := m.NewParameterAST(bs)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&bp)) }()
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	satOK(t, e)
	ap, e := m.NewParameterAST(sort)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&ap)) }()
	mk := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, op, nil, args)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	constant, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{ap}, []ast.ASTHandle{yes})
	satOK(t, e)
	astTestPin(t, m, constant)
	check := func(term, want ast.ASTHandle, status RewriteStatus) {
		t.Helper()
		d, e := m.GetDecl(term)
		satOK(t, e)
		args, e := m.AppArgumentView(term)
		satOK(t, e)
		out := astTestPin(t, m, no)
		gotStatus, e := rw.MkAppCore(d, args, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if gotStatus != status || got != want {
			t.Fatal("source rule/status", gotStatus, status, got, want)
		}
	}
	check(mk(ast.ASTArrayOpSelect, constant, no), yes, RewriteDone)
	check(mk(ast.ASTArrayOpStore, constant, no, yes), constant, RewriteDone)
	old := mk(ast.ASTArrayOpStore, constant, yes, no)
	term := mk(ast.ASTArrayOpStore, old, no, no)
	check(term, no, RewriteFailed)
	key := "sort_store"
	satOK(t, c.SetBoolName(p, &key, true))
	satOK(t, rw.UpdtParams(p))
	sorted := mk(ast.ASTArrayOpStore, mk(ast.ASTArrayOpStore, constant, no, no), yes, no)
	check(term, sorted, Rewrite2)
	check(sorted, no, RewriteFailed)
	x := goalTestAtom(t, m, "single-array-index")
	sel := mk(ast.ASTArrayOpSelect, old, x)
	satOK(t, rw.SetExpandSelectStore(true))
	eq, e := m.MkEq(yes, x)
	satOK(t, e)
	wanted, e := m.MkIte(eq, no, mk(ast.ASTArrayOpSelect, constant, x))
	satOK(t, e)
	astTestPin(t, m, wanted)
	check(sel, wanted, Rewrite2)
	condition := goalTestAtom(t, m, "array-condition")
	branch, e := m.MkIte(condition, old, constant)
	satOK(t, e)
	astTestPin(t, m, branch)
	selected := mk(ast.ASTArrayOpSelect, branch, x)
	check(selected, no, RewriteFailed)
	fallbackOut := astTestPin(t, m, no)
	fallbackArgs, e := m.AppArgumentView(selected)
	satOK(t, e)
	satOK(t, rw.MkSelect(fallbackArgs, fallbackOut))
	fallbackGot, e := m.RefGet(fallbackOut)
	satOK(t, e)
	if fallbackGot != selected {
		t.Fatal("source select fallback")
	}
	satOK(t, rw.SetExpandSelectIte(true))
	wanted, e = m.MkIte(condition, mk(ast.ASTArrayOpSelect, old, x), mk(ast.ASTArrayOpSelect, constant, x))
	satOK(t, e)
	astTestPin(t, m, wanted)
	check(selected, wanted, Rewrite2)
	// The lexical handle vectors use Go-managed slices and remain available when
	// the AST allocation budget injector is armed.
	d, e := m.GetDecl(selected)
	satOK(t, e)
	args, e := m.AppArgumentView(selected)
	satOK(t, e)
	out := astTestPin(t, m, yes)
	baseline := m.Budget().Used()
	m.Budget().FailAfter(0)
	status, e := rw.MkAppCore(d, args, out)
	m.Budget().FailAfter(-1)
	if ast.ASTSourceDebug {
		got, getErr := m.RefGet(out)
		satOK(t, getErr)
		if status != RewriteFailed || !errors.Is(e, arena.ErrInjectedFailure) || got != yes || m.Budget().Used() != baseline {
			t.Fatal("debug direct AST allocation refusal must keep the output", status, e, got, baseline, m.Budget().Used())
		}
		check(selected, wanted, Rewrite2)
		satOK(t, rw.UpdtParams(p))
		if rw.expandSelectIte || rw.expandSelectStore {
			t.Fatal("parameter update did not reset source flags")
		}
		return
	}
	if e != nil || status != Rewrite2 {
		t.Fatal("native vector rewrite", status, e)
	}
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != wanted || m.Budget().Used() != baseline {
		t.Fatal("native vector changed result or AST charge")
	}
	check(selected, wanted, Rewrite2)
	satOK(t, rw.UpdtParams(p))
	if rw.expandSelectIte || rw.expandSelectStore {
		t.Fatal("parameter update did not reset source flags")
	}
	copyOwner := *rw
	if e = copyOwner.SetExpandSelectIte(true); e != uparams.ErrParamsOwner {
		t.Fatal("copied owner accepted", e)
	}
	satOK(t, rw.Close())
	if _, e = rw.MkAppCore(ast.ASTHandle{}, ast.SliceArguments(nil), out); e != uparams.ErrParamsOwner {
		t.Fatal("closed owner accepted", e)
	}
}

func TestArrayRewriterOriginalStoreNativeBufferWarmReuse(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	bs, e := m.MkBoolSort()
	satOK(t, e)
	bp, e := m.NewParameterAST(bs)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&bp)) }()
	parameters := make([]ast.ASTParameter, 19)
	for i := range parameters {
		parameters[i] = bp
	}
	sort, e := m.MkFamilySort(fid, 0, parameters)
	satOK(t, e)
	ap, e := m.NewParameterAST(sort)
	satOK(t, e)
	defer func() { satOK(t, m.DelParameter(&ap)) }()
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	base, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{ap}, []ast.ASTHandle{yes})
	satOK(t, e)
	astTestPin(t, m, base)
	args := make([]ast.ASTHandle, 20)
	args[0] = base
	for i := 1; i < 19; i++ {
		args[i] = no
	}
	args[19] = no
	nested, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, args)
	satOK(t, e)
	astTestPin(t, m, nested)
	wantedArgs := append([]ast.ASTHandle(nil), args...)
	wantedArgs[19] = yes
	want, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, wantedArgs)
	satOK(t, e)
	astTestPin(t, m, want)
	args[0] = nested
	args[19] = yes
	term, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, args)
	satOK(t, e)
	astTestPin(t, m, term)
	d, e := m.GetDecl(term)
	satOK(t, e)
	out := astTestPin(t, m, no)
	status, e := rw.MkAppCore(d, ast.SliceArguments(args), out)
	satOK(t, e)
	if status != RewriteDone {
		t.Fatal("warm overwrite", status)
	}
	satOK(t, m.SetRef(out, no))
	baseline := m.Budget().Used()
	if !ast.ASTSourceDebug {
		// Debug assertion work has its own reached allocation path.
		m.Budget().FailAfter(0)
	}
	status, e = rw.MkAppCore(d, ast.SliceArguments(args), out)
	m.Budget().FailAfter(-1)
	if status != RewriteDone || e != nil {
		t.Fatal("native buffer warm rewrite", status, e)
	}
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != want || m.Budget().Used() != baseline {
		t.Fatal("native buffer charged custom storage", got, m.Budget().Used(), baseline)
	}
	satOK(t, m.SetRef(out, no))
	status, e = rw.MkAppCore(d, ast.SliceArguments(args), out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != RewriteDone || got != want {
		t.Fatal("native buffer reuse")
	}
}

func TestArrayRewriterConstructorRefusalsAndParameterFallback(t *testing.T) {
	succeeded := false
	for fail := int64(0); fail < 40; fail++ {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			m, c, g, p := boolRewriterConfigFixture(t)
			_, e := m.RegisterArrayPlugin()
			satOK(t, e)
			satOK(t, g.SourceSetName("rewriter.sort_store", "true"))
			satOK(t, g.SourceSetName("rewriter.expand_select_store", "true"))
			key := "sort_store"
			satOK(t, c.SetBoolName(p, &key, false))
			m.Budget().FailAfter(fail)
			rw, e := NewArrayRewriter(m, c, g, p)
			m.Budget().FailAfter(-1)
			if rw != nil {
				defer func() { satOK(t, rw.Close()) }()
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal("unexpected constructor error", e)
				}
				return
			}
			succeeded = true
			if rw.sortStore || !rw.expandSelectStore || rw.expandStoreEq || rw.expandSelectIte {
				t.Fatal("parameter precedence")
			}
			empty, e := c.NewValue()
			satOK(t, e)
			defer func() { satOK(t, c.Del(&empty)) }()
			satOK(t, rw.SetExpandSelectIte(true))
			satOK(t, rw.UpdtParams(&empty))
			if !rw.sortStore || !rw.expandSelectStore || rw.expandSelectIte {
				t.Fatal("module fallback/reset")
			}
		})
		if succeeded {
			break
		}
	}
	if !succeeded {
		t.Fatal("constructor did not reach success")
	}
}

func TestArrayRewriterRegistersFamilyWithoutDeclarationPlugin(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	if rw.family != fid {
		t.Fatal("array_util family registration changed ID")
	}
}

func TestArrayRewriterOriginalSelectAsArray(t *testing.T) {
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
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "array-select-function"), []ast.ASTHandle{bs, bs}, bs, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	astTestPin(t, m, f)
	array, e := m.MkAsArray(f)
	satOK(t, e)
	astTestPin(t, m, array)
	selected, e := m.MkFamilyApp(fid, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{array, yes, no})
	satOK(t, e)
	astTestPin(t, m, selected)
	d, e := m.GetDecl(selected)
	satOK(t, e)
	args, e := m.AppArgumentView(selected)
	satOK(t, e)
	want, e := m.MkApp(f, []ast.ASTHandle{yes, no})
	satOK(t, e)
	astTestPin(t, m, want)
	out := astTestPin(t, m, ast.ASTHandle{})
	status, e := rw.MkAppCore(d, args, out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != Rewrite1 || got != want {
		t.Fatal("source select(as-array[f], I)", status, got, want)
	}
}
