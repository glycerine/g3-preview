package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTValueSortScalarsRecursiveDatatypesAndExpressions(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterDatatypePlugin()
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	integer, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	integerSort, err := m.GetSort(integer)
	astTestMust(t, err)
	bits, err := m.BitVectorSort(9)
	astTestMust(t, err)
	name, err := m.Symbols.Intern("SMTValueList")
	astTestMust(t, err)
	nilName, err := m.Symbols.Intern("smt-value-nil")
	astTestMust(t, err)
	isNil, err := m.Symbols.Intern("is-smt-value-nil")
	astTestMust(t, err)
	consName, err := m.Symbols.Intern("smt-value-cons")
	astTestMust(t, err)
	isCons, err := m.Symbols.Intern("is-smt-value-cons")
	astTestMust(t, err)
	headName, err := m.Symbols.Intern("smt-value-head")
	astTestMust(t, err)
	tailName, err := m.Symbols.Intern("smt-value-tail")
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
		Name: name,
		Constructors: []ast.DatatypeConstructorDefinition{
			{Name: nilName, Recognizer: isNil},
			{Name: consName, Recognizer: isCons, Accessors: []ast.DatatypeAccessorDefinition{
				{Name: headName, Range: bits},
				{Name: tailName, Recursive: true, RecursiveIndex: 0},
			}},
		},
	}})
	astTestMust(t, err)
	for _, valueSort := range []ast.ASTHandle{boolean, integerSort, bits, sorts[0]} {
		yes, err := SMTIsValueSort(m, valueSort)
		astTestMust(t, err)
		if !yes {
			t.Fatal("value sort rejected", valueSort)
		}
	}
	variable, err := m.MkVar(0, sorts[0])
	astTestMust(t, err)
	yes, err := SMTExprHasValueSort(m, variable)
	astTestMust(t, err)
	if !yes {
		t.Fatal("expression overload did not inspect its sort")
	}
}

func TestSMTValueSortRejectsUninterpretedAndContainingDatatype(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterDatatypePlugin()
	astTestMust(t, err)
	uninterpreted, err := m.MkSort(symbol.Numerical(1701), ast.ASTSortInfo{})
	astTestMust(t, err)
	name, err := m.Symbols.Intern("SMTNonValueBox")
	astTestMust(t, err)
	ctor, err := m.Symbols.Intern("smt-non-value-box")
	astTestMust(t, err)
	recognizer, err := m.Symbols.Intern("is-smt-non-value-box")
	astTestMust(t, err)
	field, err := m.Symbols.Intern("smt-non-value-field")
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: name, Constructors: []ast.DatatypeConstructorDefinition{{
		Name: ctor, Recognizer: recognizer, Accessors: []ast.DatatypeAccessorDefinition{{Name: field, Range: uninterpreted}},
	}}}})
	astTestMust(t, err)
	for _, nonValue := range []ast.ASTHandle{uninterpreted, sorts[0]} {
		yes, err := SMTIsValueSort(m, nonValue)
		astTestMust(t, err)
		if yes {
			t.Fatal("non-value sort accepted", nonValue)
		}
	}
	if _, err := SMTIsValueSort(m, ast.ASTHandle{}); err == nil {
		t.Fatal("invalid sort handle accepted")
	}
}
