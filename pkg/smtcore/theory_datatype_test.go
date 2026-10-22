package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func smtDatatypeSolver(t *testing.T) (*ast.ASTManager, ast.FamilyID, *SMTSolver) {
	t.Helper()
	m, solver := smtSolverFixture(t)
	family, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	theory, err := NewSMTDatatypeTheory(family)
	astTestMust(t, err)
	astTestMust(t, solver.kernel.RegisterTheory(theory))
	return m, family, solver
}

func TestSMTDatatypeQFDTSetup(t *testing.T) {
	m := astTestManager(t)
	params, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_DT", SMTSolverConfig{Params: params})
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, solver.Close())
		astTestMust(t, params.Close())
	})
	familySymbol, err := m.Symbols.Intern("datatype")
	astTestMust(t, err)
	family, err := m.FamilyIDForSymbol(familySymbol)
	astTestMust(t, err)
	if _, ok := solver.kernel.internalizer.theories[family].(*SMTDatatypeTheory); !ok {
		t.Fatal("QF_DT did not install the datatype theory")
	}
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
		Name: astTestSymbol(t, m, "smt-qf-dt-color"),
		Constructors: []ast.DatatypeConstructorDefinition{
			{Name: astTestSymbol(t, m, "smt-qf-dt-red"), Recognizer: astTestSymbol(t, m, "smt-qf-dt-is-red")},
			{Name: astTestSymbol(t, m, "smt-qf-dt-blue"), Recognizer: astTestSymbol(t, m, "smt-qf-dt-is-blue")},
		},
	}})
	astTestMust(t, err)
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "smt-qf-dt-x"), sorts[0])
	astTestMust(t, err)
	for _, constructor := range constructors {
		recognizer, recErr := m.DatatypeConstructorRecognizer(constructor)
		astTestMust(t, recErr)
		atom, recErr := m.MkApp(recognizer, []ast.ASTHandle{x})
		astTestMust(t, recErr)
		negated, recErr := m.MkNot(atom)
		astTestMust(t, recErr)
		astTestMust(t, solver.Assert(negated))
	}
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("QF_DT setup ignored datatype exhaustiveness", status)
	}
}

func TestSMTDatatypeRecognizerCases(t *testing.T) {
	m, family, solver := smtDatatypeSolver(t)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
		Name: astTestSymbol(t, m, "smt-color"),
		Constructors: []ast.DatatypeConstructorDefinition{
			{Name: astTestSymbol(t, m, "smt-red"), Recognizer: astTestSymbol(t, m, "smt-is-red")},
			{Name: astTestSymbol(t, m, "smt-blue"), Recognizer: astTestSymbol(t, m, "smt-is-blue")},
		},
	}})
	astTestMust(t, err)
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "smt-color-x"), sorts[0])
	astTestMust(t, err)
	atoms := make([]ast.ASTHandle, 0, len(constructors))
	for _, constructor := range constructors {
		recognizer, recErr := m.DatatypeConstructorRecognizer(constructor)
		astTestMust(t, recErr)
		atom, recErr := m.MkApp(recognizer, []ast.ASTHandle{x})
		astTestMust(t, recErr)
		atoms = append(atoms, atom)
		negated, recErr := m.MkNot(atom)
		astTestMust(t, recErr)
		astTestMust(t, solver.Assert(negated))
	}
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	theory, _ := solver.kernel.internalizer.theories[family].(*SMTDatatypeTheory)
	if theory == nil || len(theory.initialized) == 0 {
		t.Fatal("datatype value did not receive a theory variable")
	}
	if status != lbool.False {
		assignments := make([]lbool.LBool, len(atoms))
		for i, atom := range atoms {
			literal, literalErr := solver.kernel.internalizer.Literal(atom)
			astTestMust(t, literalErr)
			assignments[i], literalErr = solver.kernel.internalizer.Assignment(literal)
			astTestMust(t, literalErr)
		}
		t.Fatal("datatype value escaped every constructor", status, assignments, len(solver.kernel.internalizer.Clauses()))
	}
}

func TestSMTDatatypeConstructorAccessor(t *testing.T) {
	m, _, solver := smtDatatypeSolver(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
		Name: astTestSymbol(t, m, "smt-box"),
		Constructors: []ast.DatatypeConstructorDefinition{{
			Name:       astTestSymbol(t, m, "smt-mk-box"),
			Recognizer: astTestSymbol(t, m, "smt-is-box"),
			Accessors: []ast.DatatypeAccessorDefinition{{
				Name: astTestSymbol(t, m, "smt-unbox"), Range: boolean,
			}},
		}},
	}})
	astTestMust(t, err)
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	box, err := m.MkApp(constructors[0], []ast.ASTHandle{truth})
	astTestMust(t, err)
	accessors, err := m.DatatypeConstructorAccessors(constructors[0])
	astTestMust(t, err)
	selected, err := m.MkApp(accessors[0], []ast.ASTHandle{box})
	astTestMust(t, err)
	notSelected, err := m.MkNot(selected)
	astTestMust(t, err)
	astTestMust(t, solver.Assert(notSelected))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("constructor accessor ignored its argument", status)
	}
}

func TestSMTDatatypeRecursiveOccursCheck(t *testing.T) {
	for _, cyclic := range []bool{false, true} {
		t.Run(map[bool]string{false: "finite-value", true: "cycle"}[cyclic], func(t *testing.T) {
			m, _, solver := smtDatatypeSolver(t)
			sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
				Name: astTestSymbol(t, m, "smt-nat"),
				Constructors: []ast.DatatypeConstructorDefinition{
					{Name: astTestSymbol(t, m, "smt-zero"), Recognizer: astTestSymbol(t, m, "smt-is-zero")},
					{Name: astTestSymbol(t, m, "smt-succ"), Recognizer: astTestSymbol(t, m, "smt-is-succ"), Accessors: []ast.DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "smt-pred"), Recursive: true}}},
				},
			}})
			astTestMust(t, err)
			constructors, err := m.DatatypeConstructors(sorts[0])
			astTestMust(t, err)
			x, err := m.MkConst(astTestSymbol(t, m, "smt-nat-x"), sorts[0])
			astTestMust(t, err)
			argument, err := m.MkApp(constructors[0], nil)
			astTestMust(t, err)
			if cyclic {
				argument = x
			}
			successor, err := m.MkApp(constructors[1], []ast.ASTHandle{argument})
			astTestMust(t, err)
			equality, err := m.MkEq(x, successor)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(equality))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			want := lbool.True
			if cyclic {
				want = lbool.False
			}
			if status != want {
				t.Fatal("recursive datatype status", status, "want", want)
			}
		})
	}
}

func TestSMTDatatypeInfiniteRecognizerSplit(t *testing.T) {
	m, _, solver := smtDatatypeSolver(t)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
		Name: astTestSymbol(t, m, "smt-split-nat"),
		Constructors: []ast.DatatypeConstructorDefinition{
			{Name: astTestSymbol(t, m, "smt-split-zero"), Recognizer: astTestSymbol(t, m, "smt-split-is-zero")},
			{Name: astTestSymbol(t, m, "smt-split-succ"), Recognizer: astTestSymbol(t, m, "smt-split-is-succ"), Accessors: []ast.DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "smt-split-pred"), Recursive: true}}},
		},
	}})
	astTestMust(t, err)
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "smt-split-x"), sorts[0])
	astTestMust(t, err)
	for _, constructor := range constructors {
		recognizer, recErr := m.DatatypeConstructorRecognizer(constructor)
		astTestMust(t, recErr)
		atom, recErr := m.MkApp(recognizer, []ast.ASTHandle{x})
		astTestMust(t, recErr)
		negated, recErr := m.MkNot(atom)
		astTestMust(t, recErr)
		astTestMust(t, solver.Assert(negated))
	}
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.False {
		t.Fatal("infinite datatype escaped every constructor", status)
	}
}

func TestSMTDatatypeUpdateFieldAxioms(t *testing.T) {
	for _, matching := range []bool{true, false} {
		t.Run(map[bool]string{true: "matching-constructor", false: "identity"}[matching], func(t *testing.T) {
			m, family, solver := smtDatatypeSolver(t)
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
				Name: astTestSymbol(t, m, "smt-update-value"),
				Constructors: []ast.DatatypeConstructorDefinition{
					{Name: astTestSymbol(t, m, "smt-update-empty"), Recognizer: astTestSymbol(t, m, "smt-update-is-empty")},
					{Name: astTestSymbol(t, m, "smt-update-pair"), Recognizer: astTestSymbol(t, m, "smt-update-is-pair"), Accessors: []ast.DatatypeAccessorDefinition{
						{Name: astTestSymbol(t, m, "smt-update-first"), Range: boolean},
						{Name: astTestSymbol(t, m, "smt-update-second"), Range: boolean},
					}},
				},
			}})
			astTestMust(t, err)
			constructors, err := m.DatatypeConstructors(sorts[0])
			astTestMust(t, err)
			accessors, err := m.DatatypeConstructorAccessors(constructors[1])
			astTestMust(t, err)
			parameter, err := m.NewParameterAST(accessors[0])
			astTestMust(t, err)
			updateDecl, err := m.MkFamilyFuncDecl(family, ast.ASTDatatypeUpdateField, []ast.ASTParameter{parameter}, []ast.ASTHandle{sorts[0], boolean}, ast.ASTHandle{})
			astTestMust(t, err)
			astTestPin(t, m, updateDecl)
			astTestMust(t, m.DelParameter(&parameter))
			x, err := m.MkConst(astTestSymbol(t, m, "smt-update-x"), sorts[0])
			astTestMust(t, err)
			truth, err := m.MkTrue()
			astTestMust(t, err)
			updated, err := m.MkApp(updateDecl, []ast.ASTHandle{x, truth})
			astTestMust(t, err)
			recognizerDecl, err := m.DatatypeConstructorRecognizer(constructors[1])
			astTestMust(t, err)
			isPair, err := m.MkApp(recognizerDecl, []ast.ASTHandle{x})
			astTestMust(t, err)
			if matching {
				astTestMust(t, solver.Assert(isPair))
				selected, selectErr := m.MkApp(accessors[0], []ast.ASTHandle{updated})
				astTestMust(t, selectErr)
				negated, selectErr := m.MkNot(selected)
				astTestMust(t, selectErr)
				astTestMust(t, solver.Assert(negated))
			} else {
				notPair, assertionErr := m.MkNot(isPair)
				astTestMust(t, assertionErr)
				astTestMust(t, solver.Assert(notPair))
				equal, assertionErr := m.MkEq(updated, x)
				astTestMust(t, assertionErr)
				different, assertionErr := m.MkNot(equal)
				astTestMust(t, assertionErr)
				astTestMust(t, solver.Assert(different))
			}
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("update-field axiom did not conflict", status)
			}
		})
	}
}

func TestSMTDatatypeConstructorModelValue(t *testing.T) {
	m, modelConfig := smtModelGeneratorFixture(t)
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
		Name: astTestSymbol(t, m, "smt-model-box"),
		Constructors: []ast.DatatypeConstructorDefinition{{
			Name:       astTestSymbol(t, m, "smt-model-mk-box"),
			Recognizer: astTestSymbol(t, m, "smt-model-is-box"),
			Accessors:  []ast.DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "smt-model-unbox"), Range: boolean}},
		}},
	}})
	astTestMust(t, err)
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	box, err := m.MkApp(constructors[0], []ast.ASTHandle{truth})
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "smt-model-box-x"), sorts[0])
	astTestMust(t, err)
	equality, err := m.MkEq(box, x)
	astTestMust(t, err)
	solver, err := NewSMTSolver(m, nil, "QF_DT", SMTSolverConfig{Params: modelConfig.Context, ModelEvaluator: modelConfig})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, solver.Close()) })
	astTestMust(t, solver.SetProduceModels(true))
	astTestMust(t, solver.Assert(equality))
	status, err := solver.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("datatype model status", status)
	}
	generated, err := solver.AcquireModel()
	astTestMust(t, err)
	if generated == nil {
		t.Fatal("datatype solver omitted model")
	}
	t.Cleanup(func() { astTestMust(t, generated.DecRef()) })
	declaration, err := m.GetDecl(x)
	astTestMust(t, err)
	value, err := generated.ConstantInterpretation(declaration)
	astTestMust(t, err)
	if value.IsNull() {
		t.Fatal("datatype model omitted constant interpretation")
	}
	constructor, matched, err := m.DatatypeConstructorApp(value)
	astTestMust(t, err)
	if !matched || constructor != constructors[0] {
		t.Fatal("datatype model omitted constructor", value)
	}
	argument, err := m.GetArg(value, 0)
	astTestMust(t, err)
	if !m.IsTrue(argument) {
		t.Fatal("datatype model omitted constructor argument", argument)
	}
}
