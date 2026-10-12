package arith

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func polynomialASTManager(t *testing.T) (*ast.ASTManager, *symbol.Manager, *numeral.RationalService, *arena.Budget) {
	t.Helper()
	budget := arena.NewBudget(1 << 30)
	sy, err := symbol.NewManager(symbol.Config{Budget: budget})
	equationMust(t, err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	equationMust(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
	equationMust(t, err)
	t.Cleanup(func() {
		equationMust(t, m.Close())
		equationMust(t, q.Close())
		equationMust(t, sy.Close())
		if used := budget.Used(); used != 0 {
			t.Fatalf("AST owner retained %d bytes", used)
		}
	})
	return m, sy, q, budget
}

func TestInequalityPolynomialBuilderArithmeticASTAndRationalBuffer(t *testing.T) {
	m, sy, q, _ := polynomialASTManager(t)
	owner, _ := equationManager(t)
	store, err := NewBoundEquationStore(owner)
	equationMust(t, err)
	defer func() { equationMust(t, store.Close()) }()
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	xName, err := sy.Intern("x")
	equationMust(t, err)
	yName, err := sy.Intern("y")
	equationMust(t, err)
	x, err := m.MkConst(xName, realSort)
	equationMust(t, err)
	y, err := m.MkConst(yName, realSort)
	equationMust(t, err)
	value, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&value, "2"))
	coefficient, err := m.ArithmeticNumeral(&value, false)
	equationMust(t, err)
	equationMust(t, q.Del(&value))
	text, integer, matched, err := m.ArithmeticNumeralText(coefficient)
	equationMust(t, err)
	if !matched || integer || text != "2" {
		t.Fatalf("arithmetic numeral text = (%q,%v,%v)", text, integer, matched)
	}
	text, _, matched, err = m.ArithmeticNumeralText(x)
	equationMust(t, err)
	if matched || text != "" {
		t.Fatalf("non-numeral matched as %q", text)
	}
	product, err := m.MkFamilyApp(family, 9, nil, []ast.ASTHandle{coefficient, x})
	equationMust(t, err)
	sum, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{product, y})
	equationMust(t, err)
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	defer func() { equationMust(t, builder.Close()) }()
	whole, err := builder.MkLinearPolynomial(sum)
	equationMust(t, err)
	if whole != 0 || len(store.constraints) != 1 || len(builder.var2expr) != 3 {
		t.Fatalf("polynomial owner state whole=%d constraints=%d vars=%d", whole, len(store.constraints), len(builder.var2expr))
	}
	var out bytes.Buffer
	equationMust(t, builder.store.equations.Display(&out, store.constraints[0].eq))
	if got := out.String(); got != "-1*x0 + 2*x1 + 1*x2 = 0" {
		t.Fatalf("source polynomial equation = %q", got)
	}
	again, err := builder.MkLinearPolynomial(sum)
	equationMust(t, err)
	if again != whole || len(store.constraints) != 1 {
		t.Fatal("cached polynomial created another variable or equation")
	}
	integerSort, err := m.MkFamilySort(family, 1, nil)
	equationMust(t, err)
	iName, err := sy.Intern("i")
	equationMust(t, err)
	i, err := m.MkConst(iName, integerSort)
	equationMust(t, err)
	toReal, err := m.MkFamilyApp(family, 14, nil, []ast.ASTHandle{i})
	equationMust(t, err)
	iVar, err := builder.MkLinearPolynomial(toReal)
	equationMust(t, err)
	iAgain, err := builder.MkLinearPolynomial(i)
	equationMust(t, err)
	if iVar != iAgain || !store.variables[iVar].integer || len(store.constraints) != 1 {
		t.Fatalf("to_real variable normalization = %d, %d", iVar, iAgain)
	}
	fraction, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&fraction, "3/2"))
	threeHalves, err := m.ArithmeticNumeral(&fraction, false)
	equationMust(t, err)
	equationMust(t, q.Del(&fraction))
	fractionalProduct, err := m.MkFamilyApp(family, 9, nil, []ast.ASTHandle{threeHalves, x})
	equationMust(t, err)
	secondSum, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{fractionalProduct, x, y})
	equationMust(t, err)
	second, err := builder.MkLinearPolynomial(secondSum)
	equationMust(t, err)
	if second != 4 || len(store.constraints) != 2 {
		t.Fatalf("fractional polynomial state variable=%d constraints=%d", second, len(store.constraints))
	}
	out.Reset()
	equationMust(t, builder.store.equations.Display(&out, store.constraints[1].eq))
	if got := out.String(); got != "5*x1 + 2*x2 + -2*x4 = 0" {
		t.Fatalf("fractional duplicate equation = %q", got)
	}
}
