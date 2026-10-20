package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func gateManager(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	paramsMust(t, err)
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	paramsMust(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	paramsMust(t, err)
	t.Cleanup(func() {
		paramsMust(t, m.Close())
		paramsMust(t, q.Close())
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("owner graph retained %d bytes", b.Used())
		}
	})
	return m
}

func gatePin(t *testing.T, m *ast.ASTManager, h ast.ASTHandle) ast.ASTRef {
	t.Helper()
	r, err := m.NewRef(h)
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, m.DelRef(&r)) })
	return r
}

func TestBitBlasterSourceXor3AndCarryBranches(t *testing.T) {
	m := gateManager(t)
	_, err := m.RegisterArithmeticPlugin()
	paramsMust(t, err)
	family, err := m.RegisterBitVectorPlugin()
	paramsMust(t, err)
	b, err := m.MkBoolSort()
	paramsMust(t, err)
	a, err := m.MkConst(usymbol.Numerical(701), b)
	paramsMust(t, err)
	c, err := m.MkConst(usymbol.Numerical(702), b)
	paramsMust(t, err)
	d, err := m.MkConst(usymbol.Numerical(703), b)
	paramsMust(t, err)
	gatePin(t, m, a)
	gatePin(t, m, c)
	gatePin(t, m, d)
	notA, err := m.MkNot(a)
	paramsMust(t, err)
	gatePin(t, m, notA)
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()

	extended, err := NewConfig(m, family, SMTBitBlasterParams{ExtGates: true})
	paramsMust(t, err)
	paramsMust(t, extended.MkXor3(a, a, c, out))
	got, err := m.RefGet(out)
	paramsMust(t, err)
	if got != c {
		t.Fatal("xor3 duplicate", got, c)
	}
	paramsMust(t, extended.MkXor3(a, notA, c, out))
	got, err = m.RefGet(out)
	paramsMust(t, err)
	want, err := m.MkNotSimplified(c)
	paramsMust(t, err)
	if got != want {
		t.Fatal("xor3 complement", got, want)
	}
	paramsMust(t, extended.MkCarry(a, notA, c, out))
	got, err = m.RefGet(out)
	paramsMust(t, err)
	if got != c {
		t.Fatal("carry complement", got, c)
	}
	paramsMust(t, extended.MkXor3(a, c, d, out))
	got, err = m.RefGet(out)
	paramsMust(t, err)
	kind, err := m.GetDeclKind(got)
	paramsMust(t, err)
	if kind != 60 {
		t.Fatal("xor3 extended gate", kind)
	}
	for i := uint32(1); i < 3; i++ {
		left, err := m.GetArg(got, i-1)
		paramsMust(t, err)
		right, err := m.GetArg(got, i)
		paramsMust(t, err)
		leftID, err := m.GetASTID(left)
		paramsMust(t, err)
		rightID, err := m.GetASTID(right)
		paramsMust(t, err)
		if leftID >= rightID {
			t.Fatal("source ID ordering", leftID, rightID)
		}
	}

	expanded, err := NewConfig(m, family, SMTBitBlasterParams{})
	paramsMust(t, err)
	paramsMust(t, expanded.MkXor3(a, a, c, out))
	got, err = m.RefGet(out)
	paramsMust(t, err)
	if got != c {
		t.Fatal("expanded xor chain", got, c)
	}
	paramsMust(t, expanded.MkCarry(a, a, a, out))
	got, err = m.RefGet(out)
	paramsMust(t, err)
	if got != a {
		t.Fatal("expanded majority", got, a)
	}
}
