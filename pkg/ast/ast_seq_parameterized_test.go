package ast

import (
	"errors"
	"testing"
)

func TestASTSequenceParameterizedLoopAndSkolem(t *testing.T) {
	m := astTestManager(t)
	stringValue, err := m.MkString("a")
	astTestMust(t, err)
	regex, err := m.MkSequenceToRegex(stringValue)
	astTestMust(t, err)
	checkLoop := func(h ASTHandle, want ...int32) {
		t.Helper()
		kind, e := m.GetDeclKind(h)
		astTestMust(t, e)
		if kind != 20 {
			t.Fatalf("loop kind: %d", kind)
		}
		decl, e := m.GetDecl(h)
		astTestMust(t, e)
		parameters, e := m.GetDeclParametersCopy(decl)
		astTestMust(t, e)
		defer func() {
			for i := range parameters {
				astTestMust(t, m.DelParameter(&parameters[i]))
			}
		}()
		if len(parameters) != len(want) {
			t.Fatalf("loop parameter count: %d, want %d", len(parameters), len(want))
		}
		for i, parameter := range parameters {
			value, e := m.ParameterInt(parameter)
			astTestMust(t, e)
			if value != want[i] {
				t.Fatalf("loop parameter %d: %d, want %d", i, value, want[i])
			}
		}
	}
	one, err := m.MkRegexLoop(regex, 3)
	astTestMust(t, err)
	checkLoop(one, 3)
	two, err := m.MkRegexLoop(regex, 2, 7)
	astTestMust(t, err)
	checkLoop(two, 2, 7)
	if _, err = m.MkRegexLoop(regex, 1, 2, 3); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("too many loop parameters: %v", err)
	}
	truth, err := m.MkTrue()
	astTestMust(t, err)
	if _, err = m.MkRegexLoop(truth, 1); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("non-regex loop: %v", err)
	}

	name := astTestSymbol(t, m, "seq-skolem")
	rangeSort, err := m.GetSort(stringValue)
	astTestMust(t, err)
	skolem, err := m.MkSequenceSkolem(name, []ASTHandle{stringValue, truth}, rangeSort)
	astTestMust(t, err)
	kind, err := m.GetDeclKind(skolem)
	astTestMust(t, err)
	actualRange, err := m.GetSort(skolem)
	astTestMust(t, err)
	decl, err := m.GetDecl(skolem)
	astTestMust(t, err)
	parameters, err := m.GetDeclParametersCopy(decl)
	astTestMust(t, err)
	if kind != 42 || actualRange != rangeSort || len(parameters) != 1 {
		t.Fatalf("sequence skolem: kind=%d range=%v parameters=%d", kind, actualRange, len(parameters))
	}
	gotName, err := m.ParameterSymbol(parameters[0])
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&parameters[0]))
	if gotName != name {
		t.Fatalf("sequence skolem parameter: %v, want %v", gotName, name)
	}
}
