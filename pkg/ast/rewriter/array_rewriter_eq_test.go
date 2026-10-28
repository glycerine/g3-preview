package rewriter

// Source: src/ast/rewriter/array_rewriter.cpp:381-417,32-34.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"

	uparams "github.com/glycerine/g3/pkg/util/params"
	"testing"
)

func TestArrayRewriterOriginalStoreEqualityFlagBaseAndConstraintOrder(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	key := "expand_store_eq"
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	manager, e := rw.Manager()
	satOK(t, e)
	if manager != m {
		t.Fatal("array rewriter borrowed manager")
	}
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	out := astTestPin(t, m, yes)
	st, e := rw.MkEqCore(ast.ASTHandle{}, ast.ASTHandle{}, out)
	satOK(t, e)
	if st != RewriteFailed {
		t.Fatal("disabled equality reads operands")
	}
	satOK(t, c.SetBoolName(p, &key, true))
	satOK(t, rw.UpdtParams(p))
	bs, e := m.MkBoolSort()
	satOK(t, e)
	s, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bs}), bs)
	satOK(t, e)
	atom := func(name string) ast.ASTHandle {
		d, e := m.MkFuncDecl(astTestSymbol(t, m, name), nil, s, ast.ASTFuncDeclInfo{})
		satOK(t, e)
		h, e := m.MkApp(d, nil)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	base, other := atom("equality-base"), atom("equality-other-base")
	makeApp := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, op, nil, args)
		satOK(t, e)
		astTestPin(t, m, h)
		return h
	}
	inner := makeApp(ast.ASTArrayOpStore, base, no, yes)
	lhs := makeApp(ast.ASTArrayOpStore, inner, yes, no)
	rhs := makeApp(ast.ASTArrayOpStore, base, no, no)
	firstRight := makeApp(ast.ASTArrayOpSelect, base, yes)
	secondLeft := makeApp(ast.ASTArrayOpSelect, inner, no)
	first, e := m.MkEq(no, firstRight)
	satOK(t, e)
	astTestPin(t, m, first)
	second, e := m.MkEq(secondLeft, no)
	satOK(t, e)
	astTestPin(t, m, second)
	expected, e := m.MkAndArgs([]ast.ASTHandle{first, second, second})
	satOK(t, e)
	astTestPin(t, m, expected)
	st, e = rw.MkEqCore(lhs, rhs, out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if st != RewriteFull || got != expected {
		t.Fatal("source duplicate constraint order", st, got, expected)
	}
	satOK(t, m.SetRef(out, yes))
	st, e = rw.MkEqCore(lhs, other, out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if st != RewriteFailed || got != yes {
		t.Fatal("different bases must decline")
	}
	st, e = rw.MkEqCore(base, base, out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	d, e := m.GetDeclKind(got)
	satOK(t, e)
	n, e := m.GetNumArgs(got)
	satOK(t, e)
	if st != RewriteFull || d != ast.ASTOpAnd || n != 0 {
		t.Fatal("source empty conjunction", st, d, n)
	}
}

func TestArrayRewriterOriginalStoreEqualityBufferRefusals(t *testing.T) {
	m, c, g, p := boolRewriterConfigFixture(t)
	key := "expand_store_eq"
	fid, e := m.RegisterArrayPlugin()
	satOK(t, e)
	satOK(t, c.SetBoolName(p, &key, true))
	rw, e := NewArrayRewriter(m, c, g, p)
	satOK(t, e)
	defer func() { satOK(t, rw.Close()) }()
	bs, e := m.MkBoolSort()
	satOK(t, e)
	domains := make([]ast.ASTHandle, 18)
	for i := range domains {
		domains[i] = bs
	}
	s, e := m.MkArraySort(ast.SliceArguments(domains), bs)
	satOK(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "wide-store-equality"), nil, s, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	base, e := m.MkApp(d, nil)
	satOK(t, e)
	astTestPin(t, m, base)
	yes, e := m.MkTrue()
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	lhs := base
	for i := 0; i < 18; i++ {
		args := make([]ast.ASTHandle, 20)
		args[0] = lhs
		for j := 1; j < 19; j++ {
			args[j] = yes
		}
		args[19] = no
		if i%2 != 0 {
			args[19] = yes
		}
		lhs, e = m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, args)
		satOK(t, e)
		astTestPin(t, m, lhs)
	}
	out := astTestPin(t, m, yes)
	_, e = rw.MkEqCore(lhs, base, out)
	satOK(t, e)
	want, e := m.RefGet(out)
	satOK(t, e)
	astTestPin(t, m, want)
	satOK(t, m.SetRef(out, yes))
	baseline := m.Budget().Used()
	for fault := int64(0); fault < 256; fault++ {
		m.Budget().FailAfter(fault)
		st, failure := rw.MkEqCore(lhs, base, out)
		m.Budget().FailAfter(-1)
		got, e := m.RefGet(out)
		satOK(t, e)
		if failure == nil {
			if st != RewriteFull || got != want {
				t.Fatal("equality reuse", st, got, want)
			}
			return
		}
		if !errors.Is(failure, arena.ErrInjectedFailure) || got != yes || m.Budget().Used() != baseline {
			t.Fatal("equality refusal/cleanup", fault, failure, got, m.Budget().Used(), baseline)
		}
	}
	t.Fatal("equality refusal campaign never succeeded")
}

func TestArrayRewriterOriginalParameterDescriptionWrapper(t *testing.T) {
	m, _, _, _ := boolRewriterConfigFixture(t)
	d, e := uparams.NewParamDescrs(m.Symbols, uparams.ParamDescrsConfig{Budget: m.Budget()})
	satOK(t, e)
	defer func() { satOK(t, d.Close()) }()
	var receiver *ArrayRewriter // Source get_param_descrs is static.
	satOK(t, receiver.GetParamDescrs(d))
	n, e := d.Size()
	satOK(t, e)
	if n != 3 {
		t.Fatal("array parameter description size", n)
	}
	for _, name := range []string{"expand_select_store", "expand_store_eq", "sort_store"} {
		s, e := m.Symbols.Intern(name)
		satOK(t, e)
		kind, e := d.Kind(s)
		satOK(t, e)
		if kind != uparams.ParamBool {
			t.Fatal("array parameter kind", name, kind)
		}
		def, present, e := d.Default(s)
		satOK(t, e)
		if !present || def != "false" {
			t.Fatal("array parameter default", name, def)
		}
		module, present, e := d.Module(s)
		satOK(t, e)
		if !present || module != "rewriter" {
			t.Fatal("array parameter module", name, module)
		}
	}
	satOK(t, receiver.GetParamDescrs(d))
	n, e = d.Size()
	satOK(t, e)
	if n != 3 {
		t.Fatal("duplicate parameter descriptions")
	}
}
