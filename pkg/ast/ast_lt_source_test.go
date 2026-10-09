// Source: src/ast/ast_lt.cpp:56-155.
package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"testing"
)

func TestASTLessOriginalFieldOrderAndTailTransitions(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	// Reverse allocation order makes an ID comparator observably wrong.
	z, e := m.MkConst(astTestSymbol(t, m, "z-lt"), b)
	astTestMust(t, e)
	astTestPin(t, m, z)
	a, e := m.MkConst(astTestSymbol(t, m, "a-lt"), b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	less, e := m.ASTLess(a, z)
	astTestMust(t, e)
	if !less {
		t.Fatal("application declaration-name order")
	}
	za, e := m.GetDecl(z)
	astTestMust(t, e)
	less, e = m.ASTLess(z, za)
	astTestMust(t, e)
	if !less {
		t.Fatal("AST kind precedes fields")
	}
	v0, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, v0)
	v1, e := m.MkVar(1, b)
	astTestMust(t, e)
	astTestPin(t, m, v1)
	less, e = m.ASTLess(v0, v1)
	astTestMust(t, e)
	if !less {
		t.Fatal("variable index order")
	}
	f, e := m.MkFuncDecl(astTestSymbol(t, m, "deep-lt"), []ASTHandle{b}, b, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, f)
	left, right := v0, v1
	for i := 0; i < 1000; i++ {
		left, e = m.MkApp(f, []ASTHandle{left})
		astTestMust(t, e)
		astTestPin(t, m, left)
		right, e = m.MkApp(f, []ASTHandle{right})
		astTestMust(t, e)
		astTestPin(t, m, right)
	}
	less, e = m.ASTLess(left, right)
	astTestMust(t, e)
	if !less {
		t.Fatal("iterative child tail transition")
	}
	less, e = m.ASTLess(right, left)
	astTestMust(t, e)
	if less {
		t.Fatal("reverse deep order")
	}
	less, e = m.ASTLess(left, left)
	astTestMust(t, e)
	if less {
		t.Fatal("identity is not less")
	}
}
func TestASTLexLessAndSortedOriginalFirstDifferentSlot(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, a)
	z, e := m.MkVar(1, b)
	astTestMust(t, e)
	astTestPin(t, m, z)
	less, e := m.ASTLexLess(SliceArguments([]ASTHandle{a, z}), SliceArguments([]ASTHandle{z, a}))
	astTestMust(t, e)
	if !less {
		t.Fatal("first different slot must decide")
	}
	sorted, e := m.ASTIsSorted(SliceArguments([]ASTHandle{a, a, z}))
	astTestMust(t, e)
	if !sorted {
		t.Fatal("equal adjacent entries allowed")
	}
	sorted, e = m.ASTIsSorted(SliceArguments([]ASTHandle{z, a}))
	astTestMust(t, e)
	if sorted {
		t.Fatal("inversion accepted")
	}
}

func TestASTLessOriginalParameterKindsAndValues(t *testing.T) {
	m := astTestManager(t)
	var low, high [6]ASTParameter
	var e error
	low[0], e = m.NewParameterInt(2)
	astTestMust(t, e)
	high[0], e = m.NewParameterInt(3)
	astTestMust(t, e)
	a, e := m.MkUninterpretedSort(astTestSymbol(t, m, "a-parameter-sort"), nil)
	astTestMust(t, e)
	astTestPin(t, m, a)
	z, e := m.MkUninterpretedSort(astTestSymbol(t, m, "z-parameter-sort"), nil)
	astTestMust(t, e)
	astTestPin(t, m, z)
	low[1], e = m.NewParameterAST(a)
	astTestMust(t, e)
	high[1], e = m.NewParameterAST(z)
	astTestMust(t, e)
	low[2], e = m.NewParameterSymbol(astTestSymbol(t, m, "a-parameter-symbol"))
	astTestMust(t, e)
	high[2], e = m.NewParameterSymbol(astTestSymbol(t, m, "z-parameter-symbol"))
	astTestMust(t, e)
	rational, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&rational, "2/3"))
	low[3], e = m.NewParameterRational(&rational)
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&rational, "3/4"))
	high[3], e = m.NewParameterRational(&rational)
	astTestMust(t, e)
	astTestMust(t, m.numerals.Del(&rational))
	low[4], e = m.NewParameterDouble(2.5)
	astTestMust(t, e)
	high[4], e = m.NewParameterDouble(3.5)
	astTestMust(t, e)
	low[5], e = m.NewParameterExternal(2)
	astTestMust(t, e)
	high[5], e = m.NewParameterExternal(3)
	astTestMust(t, e)
	var sorts [6]ASTHandle
	for i := range low {
		left, e := m.MkUninterpretedSort(astTestSymbol(t, m, "parameter-lt"), []ASTParameter{low[i]})
		astTestMust(t, e)
		astTestPin(t, m, left)
		right, e := m.MkUninterpretedSort(astTestSymbol(t, m, "parameter-lt"), []ASTParameter{high[i]})
		astTestMust(t, e)
		astTestPin(t, m, right)
		less, e := m.ASTLess(left, right)
		astTestMust(t, e)
		if !less {
			t.Fatal("parameter value order", i)
		}
		less, e = m.ASTLess(right, left)
		astTestMust(t, e)
		if less {
			t.Fatal("reverse parameter value order", i)
		}
		sorts[i] = left
	}
	for i := 1; i < len(sorts); i++ {
		less, e := m.ASTLess(sorts[i-1], sorts[i])
		astTestMust(t, e)
		if !less {
			t.Fatal("parameter kind precedes payload", i)
		}
	}
	for i := range low {
		astTestMust(t, m.DelParameter(&high[i]))
		astTestMust(t, m.DelParameter(&low[i]))
	}
}
func TestASTLessOriginalQuantifierPriority(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, v)
	makeQuant := func(forall bool, weight int32, n string) ASTHandle {
		h, e := m.MkQuantifier(ASTQuantifierSpec{Forall: forall, Sorts: []ASTHandle{b}, Names: []usymbol.Symbol{astTestSymbol(t, m, n)}, Body: v, Weight: weight})
		astTestMust(t, e)
		astTestPin(t, m, h)
		return h
	}
	exists := makeQuant(false, 100, "z-lt-bound")
	forall := makeQuant(true, 0, "a-lt-bound")
	less, e := m.ASTLess(exists, forall)
	astTestMust(t, e)
	if !less {
		t.Fatal("quantifier kind precedes weight and binder name")
	}
	weighted := makeQuant(true, 1, "a-lt-bound")
	less, e = m.ASTLess(forall, weighted)
	astTestMust(t, e)
	if !less {
		t.Fatal("weight precedes binder name")
	}
	named := makeQuant(true, 0, "z-lt-bound")
	less, e = m.ASTLess(forall, named)
	astTestMust(t, e)
	if !less {
		t.Fatal("binder symbol order")
	}
}

func TestASTLessRationalParameterCopyRefusalCleanup(t *testing.T) {
	m := astTestManager(t)
	value, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&value, "123456789012345678901234567890123456789012345678901234567890/7"))
	p, e := m.NewParameterRational(&value)
	astTestMust(t, e)
	left, e := m.MkUninterpretedSort(astTestSymbol(t, m, "refused-rational-lt"), []ASTParameter{p})
	astTestMust(t, e)
	astTestPin(t, m, left)
	astTestMust(t, m.DelParameter(&p))
	astTestMust(t, m.numerals.SetString(&value, "123456789012345678901234567890123456789012345678901234567891/7"))
	p, e = m.NewParameterRational(&value)
	astTestMust(t, e)
	right, e := m.MkUninterpretedSort(astTestSymbol(t, m, "refused-rational-lt"), []ASTParameter{p})
	astTestMust(t, e)
	astTestPin(t, m, right)
	astTestMust(t, m.DelParameter(&p))
	astTestMust(t, m.numerals.Del(&value))
	less, e := m.ASTLess(left, right)
	astTestMust(t, e)
	if !less {
		t.Fatal("rational setup order")
	}
	// Exhaust reusable parameter slots so the reached first lexical copy must
	// grow its actual descriptor page. Warm numeric storage is deliberately reused.
	var fillers []ASTParameter
	for m.freeParameter != 0 || m.nextParameter <= uint32(len(m.Parameters)*astPageSize) {
		p, e := m.NewParameterInt(0)
		astTestMust(t, e)
		fillers = append(fillers, p)
	}
	defer func() {
		for i := range fillers {
			astTestMust(t, m.DelParameter(&fillers[i]))
		}
	}()
	baseline := m.Budget().Used()
	parameters := m.liveParameters
	m.Budget().FailAfter(0)
	_, e = m.ASTLess(left, right)
	m.Budget().FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("parameter copy did not reach source allocation", e)
	}
	if m.liveParameters != parameters || m.Budget().Used() != baseline {
		t.Fatal("lexical parameter copy leaked", m.liveParameters, parameters, m.Budget().Used(), baseline)
	}
	less, e = m.ASTLess(left, right)
	astTestMust(t, e)
	if !less {
		t.Fatal("comparison after refusal")
	}
}
