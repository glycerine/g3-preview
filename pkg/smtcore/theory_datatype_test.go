package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
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
