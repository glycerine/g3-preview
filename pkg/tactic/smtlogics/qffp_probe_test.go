package smtlogics

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestQFFPSourceStructuralProbeOrdering(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manager.RegisterArithmeticPlugin(); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.RegisterBitVectorPlugin(); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.RegisterFloatingPointPlugin(); err != nil {
		t.Fatal(err)
	}
	probe, err := newIsFPQFNRAProbe(budget)
	if err != nil {
		t.Fatal(err)
	}
	boolean, err := manager.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	name, err := symbols.Intern("qffp-probe-p")
	if err != nil {
		t.Fatal(err)
	}
	constant, err := manager.MkConst(name, boolean)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := tactic.NewGoal(manager, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = plain.Assert(constant); err != nil {
		t.Fatal(err)
	}
	if value, evalErr := probe.Eval(plain); evalErr != nil || value != 1 {
		t.Fatal("uninterpreted Boolean leaf", value, evalErr)
	}
	if err = plain.Close(); err != nil {
		t.Fatal(err)
	}

	constant, err = manager.MkConst(name, boolean)
	if err != nil {
		t.Fatal(err)
	}
	negated, err := manager.MkNot(constant)
	if err != nil {
		t.Fatal(err)
	}
	basic, err := tactic.NewGoal(manager, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = basic.Assert(negated); err != nil {
		t.Fatal(err)
	}
	if value, evalErr := probe.Eval(basic); evalErr != nil || value != 0 {
		t.Fatal("source basic-family early rejection", value, evalErr)
	}
	if err = basic.Close(); err != nil {
		t.Fatal(err)
	}
	if err = probe.Close(); err != nil {
		t.Fatal(err)
	}
	if err = manager.Close(); err != nil {
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
