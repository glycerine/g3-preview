package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTUnsupportedArithmeticFinishesSharedAssignmentsBeforeGiveUp(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	params, err := config.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, config.Context.Del(&params)) })
	key := "preprocess"
	astTestMust(t, config.Context.SetBoolName(&params, &key, false))
	key = "compact"
	astTestMust(t, config.Context.SetBoolName(&params, &key, false))
	s, err := NewSMTSolver(m, &params, "QF_UFLRA", SMTSolverConfig{Params: config.Context, ModelEvaluator: config})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	astTestMust(t, s.SetProduceModels(true))
	zero, err := m.ArithmeticRealNumeralText("0")
	astTestMust(t, err)
	real, err := m.GetSort(zero)
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "incomplete-function"), []ast.ASTHandle{real}, boolean, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	var arguments []ast.ASTHandle
	for _, name := range []string{"incomplete-a", "incomplete-b"} {
		argument, err := m.MkConst(astTestSymbol(t, m, name), real)
		astTestMust(t, err)
		arguments = append(arguments, argument)
		application, err := m.MkApp(f, []ast.ASTHandle{argument})
		astTestMust(t, err)
		if len(arguments) == 2 {
			application, err = m.MkNot(application)
			astTestMust(t, err)
		}
		astTestMust(t, s.Assert(application))
	}
	// Symbolic exponentiation is unsupported, but native still completes its
	// feasibility/equality/nonlinear round before reporting incompleteness.
	power, err := m.MkArithmeticPower(arguments[0], arguments[1])
	astTestMust(t, err)
	application, err := m.MkApp(f, []ast.ASTHandle{power})
	astTestMust(t, err)
	astTestMust(t, s.Assert(application))
	one, err := m.ArithmeticRealNumeralText("1")
	astTestMust(t, err)
	distinct, err := m.MkApp(f, []ast.ASTHandle{one})
	astTestMust(t, err)
	distinct, err = m.MkNot(distinct)
	astTestMust(t, err)
	astTestMust(t, s.Assert(distinct))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.Undef || s.kernel.lastFailure != SMTFailureTheory {
		t.Fatal("incomplete-theory search", status, s.kernel.lastFailure)
	}
	ok, err := s.PrepareCandidateModel()
	astTestMust(t, err)
	if !ok {
		t.Fatal("candidate model unavailable")
	}
	generated, err := s.AcquireModel()
	astTestMust(t, err)
	if generated == nil {
		t.Fatal("candidate model missing")
	}
	defer func() { astTestMust(t, generated.DecRef()) }()
	interpretation, err := generated.FunctionInterpretation(f)
	astTestMust(t, err)
	if interpretation == nil {
		t.Fatal("missing candidate function interpretation")
	}
	var values [2]ast.ASTHandle
	for index, argument := range arguments {
		declaration, err := m.GetDecl(argument)
		astTestMust(t, err)
		values[index], err = generated.ConstantInterpretation(declaration)
		astTestMust(t, err)
		entry, err := interpretation.FindEntry([]ast.ASTHandle{values[index]})
		astTestMust(t, err)
		if entry == nil {
			t.Fatal("missing candidate argument", index)
		}
		value, err := entry.Result()
		astTestMust(t, err)
		if (index == 0 && !m.IsTrue(value)) || (index == 1 && !m.IsFalse(value)) {
			t.Fatal("unsupported operation bypassed shared assignment separation", index, value)
		}
	}
	if values[0] == values[1] {
		t.Fatal("opposite function results have equal candidate arguments")
	}
}

// Exercise mk_func_interps independently of arithmetic's choice of candidate
// assignments. These concrete wrapper procedures deliberately give distinct
// congruence roots the same model argument, as permitted for incomplete theories.
func TestSMTIncompleteTheoryCandidateKeepsFirstFunctionEntry(t *testing.T) {
	for _, failure := range []SMTFailure{SMTFailureTheory, SMTFailureOK, SMTFailureConflictLimit, SMTFailureQuantifiers} {
		t.Run(failure.String(), func(t *testing.T) {
			m, config := smtModelGeneratorFixture(t)
			context, _, state := smtInternalizerFixture(t, m, nil)
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			yes, err := m.MkTrue()
			astTestMust(t, err)
			no, err := m.MkFalse()
			astTestMust(t, err)
			f, err := m.MkFuncDecl(astTestSymbol(t, m, "duplicate-function"), []ast.ASTHandle{boolean}, boolean, ast.ASTFuncDeclInfo{})
			astTestMust(t, err)
			var roots []*SMTEnode
			procedures := make(map[*SMTEnode]SMTModelValueProcedure)
			for index, name := range []string{"duplicate-a", "duplicate-b", "distinct-c"} {
				argument, err := m.MkConst(astTestSymbol(t, m, name), boolean)
				astTestMust(t, err)
				argNode, err := state.MkEnode(argument, false, false, true)
				astTestMust(t, err)
				application, err := m.MkApp(f, []ast.ASTHandle{argument})
				astTestMust(t, err)
				node, err := state.MkEnode(application, false, false, true)
				astTestMust(t, err)
				astTestMust(t, context.SetRelevant(application, true))
				argValue, result := no, no
				if index == 0 {
					result = yes
				}
				if index == 2 {
					argValue = yes
				}
				procedures[argNode], err = NewSMTExpressionValueProcedure(argValue)
				astTestMust(t, err)
				procedures[node], err = NewSMTExpressionValueProcedure(result)
				astTestMust(t, err)
				roots = append(roots, argNode, node)
			}
			g, err := NewSMTModelGenerator(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, g.Close()) })
			astTestMust(t, g.SetContext(context))
			g.searchFailure = failure
			astTestMust(t, g.InitializeModel(config))
			astTestMust(t, g.BuildValues(roots, procedures))
			err = g.publishFunctionInterpretations(state)
			if ast.ASTSourceDebug && failure != SMTFailureTheory {
				if !errors.Is(err, ast.ErrASTDomain) {
					t.Fatal("non-theory duplicate assertion", err)
				}
				return
			}
			astTestMust(t, err)
			interpretation, err := g.model.Core().FunctionInterpretation(f)
			astTestMust(t, err)
			want := uint32(3)
			if failure == SMTFailureTheory {
				want = 2
			}
			if interpretation == nil || interpretation.NumEntries() != want {
				t.Fatal("candidate entry count", interpretation, want)
			}
			for index, arg := range []ast.ASTHandle{no, yes} {
				entry, err := interpretation.FindEntry([]ast.ASTHandle{arg})
				astTestMust(t, err)
				if entry == nil {
					t.Fatal("missing candidate argument", index)
				}
				value, err := entry.Result()
				astTestMust(t, err)
				if (index == 0 && !m.IsTrue(value)) || (index == 1 && !m.IsFalse(value)) {
					t.Fatal("first result or distinct entry lost", index, value)
				}
			}
		})
	}
}
