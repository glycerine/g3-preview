package smtlogics

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestQFFPLRASourceStructuralProbe(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.RegisterArithmeticPlugin(); err != nil {
		t.Fatal(err)
	}
	if _, err = m.RegisterBitVectorPlugin(); err != nil {
		t.Fatal(err)
	}
	if _, err = m.RegisterFloatingPointPlugin(); err != nil {
		t.Fatal(err)
	}
	probe, err := newIsQFFPLRAProbe(budget)
	if err != nil {
		t.Fatal(err)
	}
	boolean, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	name, err := symbols.Intern("qffplra-p")
	if err != nil {
		t.Fatal(err)
	}
	constant, err := m.MkConst(name, boolean)
	if err != nil {
		t.Fatal(err)
	}
	negated, err := m.MkNot(constant)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := tactic.NewGoal(m, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = accepted.Assert(negated); err != nil {
		t.Fatal(err)
	}
	if value, evalErr := probe.Eval(accepted); evalErr != nil || value != 1 {
		t.Fatal("basic-family Boolean application", value, evalErr)
	}
	if err = accepted.Close(); err != nil {
		t.Fatal(err)
	}
	constant, err = m.MkConst(name, boolean)
	if err != nil {
		t.Fatal(err)
	}

	functionName, err := symbols.Intern("qffplra-f")
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := m.MkFuncDecl(functionName, []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	application, err := m.MkApp(declaration, []ast.ASTHandle{constant})
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := tactic.NewGoal(m, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = rejected.Assert(application); err != nil {
		t.Fatal(err)
	}
	if value, evalErr := probe.Eval(rejected); evalErr != nil || value != 0 {
		t.Fatal("nonconstant uninterpreted Boolean application", value, evalErr)
	}
	if err = rejected.Close(); err != nil {
		t.Fatal(err)
	}
	if err = probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	if err = numerals.Close(); err != nil {
		t.Fatal(err)
	}
	if err = symbols.Close(); err != nil {
		t.Fatal(err)
	}
	if used := budget.Used(); used != 0 {
		t.Fatal("retained budget", used)
	}
}
