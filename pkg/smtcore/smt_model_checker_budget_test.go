package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTModelCheckerCounterexampleCapAppliesToEveryQuantifier(t *testing.T) {
	for _, cap := range []uint32{0, 1, 2} {
		m, config := smtModelGeneratorFixture(t)
		solver, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, solver.Close()) })
		state := solver.kernel.internalizer
		boolean, err := m.MkBoolSort()
		astTestMust(t, err)
		x, err := m.MkVar(0, boolean)
		astTestMust(t, err)
		y, err := m.MkVar(1, boolean)
		astTestMust(t, err)
		positive, err := m.MkAnd(x, y)
		astTestMust(t, err)
		negative, err := m.MkNot(positive)
		astTestMust(t, err)
		var quantifiers []ast.ASTHandle
		for _, body := range []ast.ASTHandle{positive, negative} {
			q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean, boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "budget-y"), astTestSymbol(t, m, "budget-x")}, Body: body})
			astTestMust(t, err)
			astTestMust(t, state.Internalize(q, true))
			quantifiers = append(quantifiers, q)
		}
		proto, err := protomodel.NewProtoModel(m, config)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, proto.Close()) })
		checker, err := NewSMTModelChecker(state, cap)
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, checker.Close()) })
		satisfied, err := checker.Check(proto, quantifiers)
		astTestMust(t, err)
		if satisfied {
			t.Fatal("invalid finite model was accepted", cap)
		}
		counts := make(map[ast.ASTHandle]int)
		for _, entry := range state.quantifiers.schedule.newEntries {
			counts[entry.Fingerprint.Data()]++
		}
		first := 1
		if cap == 2 {
			first = 2
		}
		if counts[quantifiers[0]] != first || counts[quantifiers[1]] != 1 {
			t.Fatal("counterexample budget was shared between quantifiers", cap, counts)
		}
		before := len(state.quantifiers.schedule.newEntries)
		astTestMust(t, checker.InitSearch())
		satisfied, err = checker.Check(proto, quantifiers)
		astTestMust(t, err)
		if satisfied || len(state.quantifiers.schedule.newEntries) != before {
			t.Fatal("repeated counterexamples validated the model or escaped the cap", cap, satisfied, len(state.quantifiers.schedule.newEntries), before)
		}
	}
}
