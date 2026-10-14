package arith

import (
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/text"
)

// Source: propagate_ineqs_tactic.cpp:483-513; tactic.cpp:64-69.
func TestInequalityBuilderPublishesRetainedProgressOnRealGoal(t *testing.T) {
	m, symbols, q, _ := polynomialASTManager(t)
	numbers, _ := equationManager(t)
	store, err := NewBoundEquationStore(numbers)
	equationMust(t, err)
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	input, err := goaltactic.NewGoal(m, false, false, false)
	equationMust(t, err)
	t.Cleanup(func() {
		equationMust(t, input.Close())
		equationMust(t, builder.Close())
		equationMust(t, store.Close())
	})
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	name, err := symbols.Intern("progress-x")
	equationMust(t, err)
	x, err := m.MkConst(name, realSort)
	equationMust(t, err)
	yName, err := symbols.Intern("progress-y")
	equationMust(t, err)
	y, err := m.MkConst(yName, realSort)
	equationMust(t, err)
	v, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&v, "2"))
	two, err := m.ArithmeticNumeral(&v, false)
	equationMust(t, err)
	lower, err := m.MkFamilyApp(family, 3, nil, []ast.ASTHandle{x, two})
	equationMust(t, err)
	equationMust(t, q.SetString(&v, "3"))
	three, err := m.ArithmeticNumeral(&v, false)
	equationMust(t, err)
	equationMust(t, q.Del(&v))
	sum, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{x, y})
	equationMust(t, err)
	upperSum, err := m.MkFamilyApp(family, 2, nil, []ast.ASTHandle{sum, three})
	equationMust(t, err)
	equationMust(t, input.Assert(lower))
	equationMust(t, input.Assert(upperSum))
	b := arena.NewBudget(2048)
	out, err := text.NewNumericTextStream(text.NumericTextStreamConfig{Budget: b})
	equationMust(t, err)
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	equationMust(t, err)
	oldStream, err := diagnostic.BorrowProcessVerboseStream()
	equationMust(t, err)
	equationMust(t, diagnostic.SetProcessVerboseStream(out))
	t.Cleanup(func() {
		equationMust(t, diagnostic.SetProcessVerboseStream(oldStream))
		equationMust(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
		equationMust(t, out.Close())
	})
	equationMust(t, diagnostic.SetProcessVerbosityLevel(10))
	result, err := builder.Apply(input)
	equationMust(t, err)
	if result == nil || result == input {
		t.Fatal("progress goal did not produce owned output")
	}
	t.Cleanup(func() { equationMust(t, result.Close()) })
	message, err := out.String()
	equationMust(t, err)
	if !strings.Contains(message, "(:bound-propagations 1)\n") || !strings.Contains(message, "(propagate-ineqs :num-exprs ") || strings.Contains(message, ":bound-false-alarms") {
		t.Fatal("source progress or tactic report", message)
	}
}
