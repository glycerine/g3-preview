// Source: src/ast/arith_decl_plugin.cpp:342-399,451-509.
package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTArithmeticOriginalOperatorDeclarationSelection(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	i, r := m.external(m.arithmeticPlugin.fields[1]), m.external(m.arithmeticPlugin.fields[0])
	for _, enabled := range []bool{true, false} {
		astTestMust(t, m.EnableIntRealCoercions(enabled))
		for _, row := range []struct {
			kind        int32
			ints, reals int
			eligible    bool
		}{
			{2, 11, 2, true}, {3, 12, 3, true}, {4, 13, 4, true}, {5, 14, 5, true}, {6, 15, 6, true}, {7, 16, 7, true}, {8, 17, 8, true}, {9, 18, 9, true},
			{10, 10, 10, false}, {11, 19, 19, false}, {12, 21, 21, false}, {13, 20, 20, false}, {14, 22, 22, false}, {15, 23, 23, false}, {16, 24, 24, false},
			{17, 27, 28, false}, {18, 25, 26, true}, {19, 29, 29, false}, {20, 30, 30, false}, {21, 31, 31, false}, {22, 32, 32, false}, {23, 33, 33, false}, {24, 34, 34, false},
			{25, 35, 35, false}, {26, 36, 36, false}, {27, 37, 37, false}, {28, 38, 38, false}, {29, 39, 39, false}, {30, 40, 40, false}, {33, 43, 43, false}, {34, 44, 44, false}, {35, 45, 45, false},
		} {
			for _, domains := range [][]ASTHandle{{i, i}, {r, i}, {i, r}, {r, r}} {
				want := row.ints
				if domains[0] == r || enabled && row.eligible && domains[1] == r {
					want = row.reals
				}
				d, e := m.MkFamilyFuncDecl(fid, row.kind, nil, domains, ASTHandle{})
				astTestMust(t, e)
				if d != m.external(m.arithmeticPlugin.fields[want]) {
					t.Fatal("original cached declaration selection", enabled, row.kind, domains, d, want)
				}
			}
		}
	}
	d, e := m.MkFamilyFuncDecl(fid, 7, nil, []ASTHandle{i}, r)
	astTestMust(t, e)
	if d != m.external(m.arithmeticPlugin.fields[17]) {
		t.Fatal("unary SUB must select UMINUS")
	}
	for _, row := range []struct {
		kind  int32
		index int
	}{{31, 41}, {32, 42}} {
		h, e := m.MkFamilyApp(fid, row.kind, nil, nil)
		astTestMust(t, e)
		if h != m.external(m.arithmeticPlugin.fields[row.index]) {
			t.Fatal("original PI/E canonical constant", row.kind)
		}
	}
	if _, e := m.MkFamilyFuncDecl(fid, 6, nil, nil, r); e == nil {
		t.Fatal("zero-arity ADD accepted")
	}
	d, e = m.MkFamilyFuncDecl(fid, 999, nil, []ASTHandle{i}, ASTHandle{})
	astTestMust(t, e)
	if !d.IsNull() {
		t.Fatal("unknown operator must return null")
	}
}

func TestASTArithmeticOriginalMixedApplications(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	i, r := m.external(m.arithmeticPlugin.fields[1]), m.external(m.arithmeticPlugin.fields[0])
	x, e := m.MkConst(astTestSymbol(t, m, "arith-int"), i)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkConst(astTestSymbol(t, m, "arith-real"), r)
	astTestMust(t, e)
	astTestPin(t, m, y)
	h, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{x, y})
	astTestMust(t, e)
	astTestPin(t, m, h)
	want, e := m.MkFamilyApp(fid, 14, nil, []ASTHandle{x})
	astTestMust(t, e)
	a, e := m.GetArg(h, 0)
	astTestMust(t, e)
	if a != want {
		t.Fatal("mixed ADD requires original to_real conversion")
	}
	// Coercion follows the declaration's domain, including non-associative
	// operators and declarations from other families with arithmetic domains.
	convertedInt, e := m.MkFamilyApp(fid, 15, nil, []ASTHandle{y})
	astTestMust(t, e)
	astTestPin(t, m, convertedInt)
	for _, row := range []struct {
		kind     int32
		args     []ASTHandle
		expected []ASTHandle
	}{
		{10, []ASTHandle{x, y}, []ASTHandle{want, y}},
		{11, []ASTHandle{x, y}, []ASTHandle{x, convertedInt}},
		{13, []ASTHandle{y, x}, []ASTHandle{convertedInt, x}},
		{18, []ASTHandle{x, y}, []ASTHandle{want, y}},
	} {
		h, e := m.MkFamilyApp(fid, row.kind, nil, row.args)
		astTestMust(t, e)
		astTestPin(t, m, h)
		for i, want := range row.expected {
			got, e := m.GetArg(h, uint32(i))
			astTestMust(t, e)
			if got != want {
				t.Fatal("original domain-selected cast", row.kind, i, got, want)
			}
		}
	}
	af, e := m.RegisterArrayPlugin()
	astTestMust(t, e)
	as, e := m.MkArraySort(SliceArguments([]ASTHandle{r}), r)
	astTestMust(t, e)
	array, e := m.MkConst(astTestSymbol(t, m, "arithmetic-domain-array"), as)
	astTestMust(t, e)
	astTestPin(t, m, array)
	selected, e := m.MkArrayAppArguments(af, ASTArrayOpSelect, SliceArguments([]ASTHandle{array, x}))
	astTestMust(t, e)
	astTestPin(t, m, selected)
	index, e := m.GetArg(selected, 1)
	astTestMust(t, e)
	if index != want {
		t.Fatal("array-domain arithmetic coercion")
	}
	astTestMust(t, m.EnableIntRealCoercions(false))
	if _, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{x, y}); e == nil {
		t.Fatal("disabled coercion accepted mixed ADD")
	}
}

func TestASTArithmeticExpressionOverloadReadErasure(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	r := m.external(m.arithmeticPlugin.fields[0])
	y, e := m.MkConst(astTestSymbol(t, m, "arith-real-prefix"), r)
	astTestMust(t, e)
	args := make([]ASTHandle, 18)
	args[0] = y
	// Unknown operator returns null after the first argument in the original
	// non-coercing expression overload. It does not build a domain buffer or
	// read invalid tail handles. Non-numeral parameters/range are ignored.
	m.Budget().FailAfter(0)
	h, e := m.MkFamilyApp(fid, 999, []ASTParameter{{}}, args)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if !h.IsNull() {
		t.Fatal("unknown operator fallback")
	}
	if _, e := m.MkFamilyApp(fid, 0, nil, []ASTHandle{{}}); e == nil {
		t.Fatal("NUM validates parameter/arity before argument reads")
	}
}

func TestASTArithmeticAppCoercionEveryWarmReservation(t *testing.T) {
	succeeded, sawPending := false, false
	for fault := int64(0); fault < 128; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			m := astTestManager(t)
			fid, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			x, e := m.MkConst(astTestSymbol(t, m, "coercion-int"), m.external(m.arithmeticPlugin.fields[1]))
			astTestMust(t, e)
			astTestPin(t, m, x)
			y, e := m.MkConst(astTestSymbol(t, m, "coercion-real"), m.external(m.arithmeticPlugin.fields[0]))
			astTestMust(t, e)
			astTestPin(t, m, y)
			args := make([]ASTHandle, 18)
			for i := range args {
				args[i] = x
			}
			args[17] = y
			want, e := m.MkFamilyApp(fid, 6, nil, args)
			astTestMust(t, e)
			astTestPin(t, m, want)
			m.Budget().FailAfter(fault)
			got, e := m.MkFamilyApp(fid, 6, nil, args)
			m.Budget().FailAfter(-1)
			if m.formatExecuting {
				t.Fatal("arithmetic expression execution guard retained")
			}
			if e == nil {
				if got != want || m.appCoercion.active {
					t.Fatal("canonical variadic result/temporary cleanup", fault)
				}
				succeeded = true
			} else {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal("warm coercion refusal", fault, e)
				}
				if m.appCoercion.pending.slot != 0 {
					sawPending = true
					if e := m.cleanupAppCoercion(false); !errors.Is(e, ErrASTPending) {
						t.Fatal("source guard for unjoined buffer INC", e)
					}
				}
			}
		})
		if succeeded {
			if !sawPending {
				t.Fatal("owning buffer overflow was not reached")
			}
			return
		}
	}
	t.Fatal("coercion campaign never succeeded")
}
