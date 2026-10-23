package ast

import "testing"

func TestASTBasicEqualityAndITEJoinMixedArithmeticAsReal(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	integer, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	real, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "join-int"), integer)
	astTestMust(t, err)
	astTestPin(t, m, x)
	y, err := m.MkConst(astTestSymbol(t, m, "join-real"), real)
	astTestMust(t, err)
	astTestPin(t, m, y)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "join-condition"), boolean)
	astTestMust(t, err)
	astTestPin(t, m, p)
	toReal, err := m.MkFamilyApp(family, 14, nil, []ASTHandle{x})
	astTestMust(t, err)
	astTestPin(t, m, toReal)
	for _, reverse := range []bool{false, true} {
		left, right := x, y
		if reverse {
			left, right = y, x
		}
		eq, err := m.MkEq(left, right)
		astTestMust(t, err)
		astTestPin(t, m, eq)
		for index := uint32(0); index < 2; index++ {
			arg, err := m.GetArg(eq, index)
			astTestMust(t, err)
			sort, err := m.GetSort(arg)
			astTestMust(t, err)
			if sort != real || arg != y && arg != toReal {
				t.Fatal("mixed equality narrowed a Real operand", reverse, index, arg)
			}
		}
		ite, err := m.MkIte(p, left, right)
		astTestMust(t, err)
		astTestPin(t, m, ite)
		sort, err := m.GetSort(ite)
		astTestMust(t, err)
		if sort != real {
			t.Fatal("mixed ITE has Int result", reverse)
		}
		for index := uint32(1); index < 3; index++ {
			arg, err := m.GetArg(ite, index)
			astTestMust(t, err)
			if arg != y && arg != toReal {
				t.Fatal("mixed ITE narrowed a Real operand", reverse, index, arg)
			}
		}
	}
}
