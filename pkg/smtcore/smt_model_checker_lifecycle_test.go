package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTModelCheckerCandidateCompressionAndSuccessfulCleanup(t *testing.T) {
	for _, scenario := range []string{"accepted", "rejected", "no-quantifiers", "iteration-limit"} {
		t.Run(scenario, func(t *testing.T) {
			m, config := smtModelGeneratorFixture(t)
			solver, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, solver.Close()) })
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			yes, err := m.MkTrue()
			astTestMust(t, err)
			x, err := m.MkVar(0, boolean)
			astTestMust(t, err)
			body := x
			if scenario != "rejected" {
				body, err = m.MkIff(x, x)
				astTestMust(t, err)
			}
			q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "lifecycle-x")}, Body: body})
			astTestMust(t, err)
			state := solver.kernel.internalizer
			astTestMust(t, state.Internalize(q, true))
			proto, err := protomodel.NewProtoModel(m, config)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, proto.Close()) })
			declare := func(name string) ast.ASTHandle {
				d, err := m.MkFuncDecl(astTestSymbol(t, m, name), []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
				astTestMust(t, err)
				ref, err := m.NewRef(d)
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, m.DelRef(&ref)) })
				return d
			}
			unused := declare("lifecycle-unused-aux")
			aux, err := model.NewFunctionInterpretation(m, 1)
			astTestMust(t, err)
			astTestMust(t, aux.SetElse(yes))
			astTestMust(t, proto.RegisterAuxFunction(unused, aux))
			f := declare("lifecycle-compress")
			fi, err := model.NewFunctionInterpretation(m, 1)
			astTestMust(t, err)
			astTestMust(t, fi.InsertEntry([]ast.ASTHandle{yes}, yes))
			astTestMust(t, fi.SetElse(yes))
			astTestMust(t, proto.Core().RegisterFunction(f, fi))
			late, err := m.MkConst(astTestSymbol(t, m, "lifecycle-late"), boolean)
			astTestMust(t, err)
			lateDecl, err := m.GetDecl(late)
			astTestMust(t, err)
			lateFI, err := model.NewFunctionInterpretation(m, 1)
			astTestMust(t, err)
			astTestMust(t, lateFI.SetElse(late))
			astTestMust(t, proto.Core().RegisterFunction(declare("lifecycle-cleanup"), lateFI))
			checker, err := NewSMTModelChecker(state, 1)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, checker.Close()) })
			quantifiers := []ast.ASTHandle{q}
			if scenario == "no-quantifiers" {
				quantifiers = nil
			}
			if scenario == "iteration-limit" {
				astTestMust(t, checker.ConfigureLifecycle(1, 0))
			}
			accepted, err := checker.Check(proto, quantifiers)
			astTestMust(t, err)
			if accepted != (scenario == "accepted" || scenario == "no-quantifiers") {
				t.Fatal("unexpected model check result", accepted)
			}
			expectedEntries := uint32(0)
			if scenario == "no-quantifiers" || scenario == "iteration-limit" {
				expectedEntries = 1
			}
			if fi.NumEntries() != expectedEntries {
				t.Fatal("candidate compression lifecycle", fi.NumEntries(), expectedEntries)
			}
			remaining, err := proto.Core().FunctionInterpretation(unused)
			astTestMust(t, err)
			value, err := proto.Core().ConstantInterpretation(lateDecl)
			astTestMust(t, err)
			fallback, err := lateFI.Else()
			astTestMust(t, err)
			if scenario == "accepted" {
				if remaining != nil || value.IsNull() || value != fallback || !m.IsFalse(fallback) {
					t.Fatal("successful candidate was not cleaned", remaining, value, fallback)
				}
			} else if remaining == nil || !value.IsNull() || fallback != late {
				t.Fatal("unchecked or rejected candidate was cleaned", remaining, value, fallback)
			}
		})
	}
}

func TestSMTModelCheckerResidualQuantifierDoesNotValidateCandidate(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	solver, err := NewSMTSolver(m, nil, "AUFLIA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	y, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	inner, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: false, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "residual-y")}, Body: y})
	astTestMust(t, err)
	outer, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{astTestSymbol(t, m, "residual-x")}, Body: inner})
	astTestMust(t, err)
	outerRef, err := m.NewRef(outer)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&outerRef)) })
	state := solver.kernel.internalizer
	proto, err := protomodel.NewProtoModel(m, config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, proto.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	_, err = proto.Eval(inner, out, true)
	astTestMust(t, err)
	value, err := m.RefGet(out)
	astTestMust(t, err)
	if m.IsTrue(value) || m.IsFalse(value) {
		t.Fatal("fixture did not retain an unresolved quantifier", value)
	}
	checker, err := NewSMTModelChecker(state, 1)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, checker.Close()) })
	accepted, err := checker.Check(proto, []ast.ASTHandle{outer})
	astTestMust(t, err)
	if accepted || !checker.Exhausted() || len(state.quantifiers.schedule.newEntries) != 0 {
		t.Fatal("unresolved evaluation validated the model or queued a false counterexample", accepted, checker.Exhausted())
	}
}
