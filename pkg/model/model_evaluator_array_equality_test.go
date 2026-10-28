package model_test

// Source: src/model/model_evaluator.cpp:292-420.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"testing"
)

func TestModelEvaluatorOriginalArrayEqualityTablesAndStorePrecedence(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "equality-array-function"), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	fi, e := model.NewFunctionInterpretation(m, 1)
	astTestMust(t, e)
	astTestMust(t, fi.SetElse(no))
	astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, yes))
	astTestMust(t, md.Core().RegisterFunction(f, fi))
	array, e := m.MkAsArray(f)
	astTestMust(t, e)
	astTestPin(t, m, array)
	sort, e := m.GetSort(array)
	astTestMust(t, e)
	parameter, e := m.NewParameterAST(sort)
	astTestMust(t, e)
	constant, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{parameter}, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, constant)
	astTestMust(t, m.DelParameter(&parameter))
	mk := func(base, index, value ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, index, value})
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	value := mk(constant, yes, yes)
	override := mk(array, yes, no)
	out := astTestPin(t, m, ast.ASTHandle{})
	check := func(a, b, want ast.ASTHandle) {
		t.Helper()
		eq, e := m.MkEq(a, b)
		astTestMust(t, e)
		astTestPin(t, m, eq)
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(eq, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != want {
			t.Fatal("array equality", got, want)
		}
	}
	check(array, value, yes)
	check(array, constant, no)
	check(override, constant, yes) // earlier left row replaces the older graph row
	check(array, override, no)     // earlier right row wins over its graph row
	check(constant, override, yes) // empty left stores uses the source fallback
}

func TestModelEvaluatorOriginalArrayEqualitySymbolicIndexFallback(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	bp, e := m.NewParameterAST(bs)
	astTestMust(t, e)
	sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
	astTestMust(t, e)
	astTestMust(t, m.DelParameter(&bp))
	parameter, e := m.NewParameterAST(sort)
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	constant, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{parameter}, []ast.ASTHandle{no})
	astTestMust(t, e)
	astTestPin(t, m, constant)
	astTestMust(t, m.DelParameter(&parameter))
	x, y := goalTestAtom(t, m, "array-equality-x"), goalTestAtom(t, m, "array-equality-y")
	a, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{constant, x, yes})
	astTestMust(t, e)
	astTestPin(t, m, a)
	b, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{constant, y, yes})
	astTestMust(t, e)
	astTestPin(t, m, b)
	eq, e := m.MkEq(a, b)
	astTestMust(t, e)
	astTestPin(t, m, eq)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.Evaluate(eq, out))
	residual, e := m.RefGet(out)
	astTestMust(t, e)
	astTestPin(t, m, residual)
	dx, e := m.GetDecl(x)
	astTestMust(t, e)
	dy, e := m.GetDecl(y)
	astTestMust(t, e)
	for bits := 0; bits < 4; bits++ {
		vx, vy := no, no
		if bits&1 != 0 {
			vx = yes
		}
		if bits&2 != 0 {
			vy = yes
		}
		astTestMust(t, md.Core().RegisterConstant(dx, vx))
		astTestMust(t, md.Core().RegisterConstant(dy, vy))
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(residual, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		want := no
		if vx == vy {
			want = yes
		}
		if got != want {
			t.Fatal("fallback truth table", bits, got, want)
		}
	}
}

func TestModelEvaluatorOriginalArrayEqualityMatchedUnknownValueConstraints(t *testing.T) {
	m, md, ev, _, _, _ := evaluatorFixture(t)
	_, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	x, y := goalTestAtom(t, m, "array-value-x"), goalTestAtom(t, m, "array-value-y")
	backed := func(name string, value ast.ASTHandle) ast.ASTHandle {
		f, e := m.MkFuncDecl(astTestSymbol(t, m, name), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
		astTestMust(t, e)
		astTestPin(t, m, f)
		fi, e := model.NewFunctionInterpretation(m, 1)
		astTestMust(t, e)
		astTestMust(t, fi.SetElse(no))
		astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, value))
		astTestMust(t, md.Core().RegisterFunction(f, fi))
		array, e := m.MkAsArray(f)
		astTestMust(t, e)
		astTestPin(t, m, array)
		return array
	}
	a, b := backed("unknown-value-array-left", x), backed("unknown-value-array-right", y)
	eq, e := m.MkEq(a, b)
	astTestMust(t, e)
	astTestPin(t, m, eq)
	out := astTestPin(t, m, ast.ASTHandle{})
	astTestMust(t, ev.Evaluate(eq, out))
	residual, e := m.RefGet(out)
	astTestMust(t, e)
	astTestPin(t, m, residual)
	dx, e := m.GetDecl(x)
	astTestMust(t, e)
	dy, e := m.GetDecl(y)
	astTestMust(t, e)
	// Source leaves an unknown matched row in table1. The final pass also
	// compares it to else2, so these are the source constraints, not x=y alone.
	for bits := 0; bits < 4; bits++ {
		vx, vy := no, no
		if bits&1 != 0 {
			vx = yes
		}
		if bits&2 != 0 {
			vy = yes
		}
		astTestMust(t, md.Core().RegisterConstant(dx, vx))
		astTestMust(t, md.Core().RegisterConstant(dy, vy))
		astTestMust(t, ev.Reset(nil))
		astTestMust(t, ev.Evaluate(residual, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		want := no
		if bits == 0 {
			want = yes
		}
		if got != want {
			t.Fatal("source unknown-row constraints", bits, got, want)
		}
	}
}

func TestModelEvaluatorOriginalArrayEqualityAllocationRefusals(t *testing.T) {
	for _, symbolic := range []bool{false, true} {
		t.Run(fmt.Sprint(symbolic), func(t *testing.T) {
			m, md, ev, _, _, _ := evaluatorFixture(t)
			fid, e := m.RegisterArrayPlugin()
			astTestMust(t, e)
			bs := mustBoolSort(t, m)
			yes, e := m.MkTrue()
			astTestMust(t, e)
			no, e := m.MkFalse()
			astTestMust(t, e)
			x, y := goalTestAtom(t, m, "fault-eq-x"), goalTestAtom(t, m, "fault-eq-y")
			var a, b ast.ASTHandle
			if !symbolic {
				backed := func(name string, v ast.ASTHandle) ast.ASTHandle {
					f, e := m.MkFuncDecl(astTestSymbol(t, m, name), []ast.ASTHandle{bs}, bs, ast.ASTFuncDeclInfo{})
					astTestMust(t, e)
					astTestPin(t, m, f)
					fi, e := model.NewFunctionInterpretation(m, 1)
					astTestMust(t, e)
					astTestMust(t, fi.SetElse(no))
					astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, v))
					astTestMust(t, md.Core().RegisterFunction(f, fi))
					h, e := m.MkAsArray(f)
					astTestMust(t, e)
					astTestPin(t, m, h)
					return h
				}
				a, b = backed("fault-eq-left", x), backed("fault-eq-right", y)
			} else {
				bp, e := m.NewParameterAST(bs)
				astTestMust(t, e)
				sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, bp})
				astTestMust(t, e)
				astTestMust(t, m.DelParameter(&bp))
				p, e := m.NewParameterAST(sort)
				astTestMust(t, e)
				base, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{p}, []ast.ASTHandle{no})
				astTestMust(t, e)
				astTestPin(t, m, base)
				astTestMust(t, m.DelParameter(&p))
				a, e = m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, x, yes})
				astTestMust(t, e)
				astTestPin(t, m, a)
				b, e = m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, y, yes})
				astTestMust(t, e)
				astTestPin(t, m, b)
			}
			eq, e := m.MkEq(a, b)
			astTestMust(t, e)
			astTestPin(t, m, eq)
			out := astTestPin(t, m, eq)
			astTestMust(t, ev.Evaluate(eq, out))
			expected, e := m.RefGet(out)
			astTestMust(t, e)
			astTestPin(t, m, expected)
			roots := []ast.ASTHandle{a, b, x, y, yes, no}
			for fault := int64(0); fault < 256; fault++ {
				astTestMust(t, ev.Reset(nil))
				astTestMust(t, m.SetRef(out, eq))
				counts := make([]uint32, len(roots))
				for i, h := range roots {
					counts[i], e = m.GetRefCount(h)
					astTestMust(t, e)
				}
				m.Budget().FailAfter(fault)
				failure := ev.Evaluate(eq, out)
				m.Budget().FailAfter(-1)
				if failure != nil && !errors.Is(failure, arena.ErrInjectedFailure) {
					t.Fatal("unexpected equality refusal", fault, failure)
				}
				if failure == nil {
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if got != expected {
						t.Fatal("successful equality after refusals")
					}
					return
				}
				astTestMust(t, m.SetRef(out, eq))
				astTestMust(t, ev.Reset(nil))
				for i, h := range roots {
					// Source array_rewriter.cpp:173-193 creates raw select/equality
					// intermediates before publishing the enclosing ITE. Refusal
					// can leave those zero-reference nodes (and their child INCs)
					// with the AST manager until its Close, checked by the fixture.
					// The original array operands cannot be such children here.
					if symbolic && i >= 2 {
						continue
					}
					count, e := m.GetRefCount(h)
					astTestMust(t, e)
					if count != counts[i] {
						t.Fatal("equality reference leak", fault, i, count, counts[i])
					}
				}
			}
			t.Fatal("equality refusal campaign never reached success")
		})
	}
}

func TestModelEvaluatorOriginalArrayEqualityFiniteAndInfiniteElseCases(t *testing.T) {
	m, _, ev, _, _, _ := evaluatorFixture(t)
	fid, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	bs := mustBoolSort(t, m)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	constant := func(rangeSort, value ast.ASTHandle) ast.ASTHandle {
		bp, e := m.NewParameterAST(bs)
		astTestMust(t, e)
		rp, e := m.NewParameterAST(rangeSort)
		astTestMust(t, e)
		sort, e := m.MkFamilySort(fid, 0, []ast.ASTParameter{bp, rp})
		astTestMust(t, e)
		astTestMust(t, m.DelParameter(&bp))
		astTestMust(t, m.DelParameter(&rp))
		p, e := m.NewParameterAST(sort)
		astTestMust(t, e)
		h, e := m.MkFamilyApp(fid, ast.ASTArrayOpConst, []ast.ASTParameter{p}, []ast.ASTHandle{value})
		astTestMust(t, e)
		astTestPin(t, m, h)
		astTestMust(t, m.DelParameter(&p))
		return h
	}
	check := func(a, b ast.ASTHandle) {
		eq, e := m.MkEq(a, b)
		astTestMust(t, e)
		astTestPin(t, m, eq)
		out := astTestPin(t, m, ast.ASTHandle{})
		astTestMust(t, ev.Evaluate(eq, out))
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if got != no {
			t.Fatal("distinct else cases", got)
		}
	}
	left, right := constant(bs, no), constant(bs, yes)
	check(left, right)
	// The source retains the differing-default equality even when stores cover
	// the entire finite Boolean index domain. Preserve that source behavior.
	a, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{left, yes, yes})
	astTestMust(t, e)
	astTestPin(t, m, a)
	b, e := m.MkFamilyApp(fid, ast.ASTArrayOpStore, nil, []ast.ASTHandle{right, no, no})
	astTestMust(t, e)
	astTestPin(t, m, b)
	check(a, b)
	infinite, e := m.MkUninterpretedSort(astTestSymbol(t, m, "array-infinite-range"), nil)
	astTestMust(t, e)
	astTestPin(t, m, infinite)
	modelValue := func(index int32) ast.ASTHandle {
		p, e := m.NewParameterInt(index)
		astTestMust(t, e)
		s, e := m.NewParameterAST(infinite)
		astTestMust(t, e)
		h, e := m.MkFamilyApp(3, 0, []ast.ASTParameter{p, s}, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		astTestMust(t, m.DelParameter(&p))
		astTestMust(t, m.DelParameter(&s))
		return h
	}
	check(constant(infinite, modelValue(0)), constant(infinite, modelValue(1)))
}
