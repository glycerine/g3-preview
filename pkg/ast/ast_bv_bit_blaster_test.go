package ast

import "testing"

func TestASTBitBlasterBitFactories(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	sort, err := m.BitVectorSort(3)
	astTestMust(t, err)
	value, err := m.MkConst(astTestSymbol(t, m, "bit-blaster-value"), sort)
	astTestMust(t, err)
	bit, err := m.MkBitVectorBit2Bool(1, value)
	astTestMust(t, err)
	if got, err := m.GetDeclKind(bit); err != nil || got != 55 {
		t.Fatal("bit2bool kind", got, err)
	}
	truth, err := m.MkTrue()
	astTestMust(t, err)
	falsehood, err := m.MkFalse()
	astTestMust(t, err)
	assembled, err := m.MkBitVectorFromBits([]ASTHandle{truth, bit, falsehood})
	astTestMust(t, err)
	if got, err := m.GetDeclKind(assembled); err != nil || got != 56 {
		t.Fatal("mkbv kind", got, err)
	}
	if got, err := m.GetFamilyID(assembled); err != nil || got != family {
		t.Fatal("mkbv family", got, family, err)
	}
	if got, err := m.BitVectorSize(assembled); err != nil || got != 3 {
		t.Fatal("mkbv width", got, err)
	}
	if _, err = m.MkBitVectorBit2Bool(3, value); err == nil {
		t.Fatal("out-of-range bit accepted")
	}
}
