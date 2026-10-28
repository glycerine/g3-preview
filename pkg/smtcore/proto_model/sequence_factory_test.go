package protomodel

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSequenceFactoryCanonicalFreshAndGenericValues(t *testing.T) {
	m, p := protoFixture(t)
	family, err := m.RegisterSequencePlugin()
	protoMust(t, err)
	factory, err := NewSequenceFactory(p, family)
	protoMust(t, err)
	protoMust(t, p.RegisterFactory(factory))

	empty, err := m.MkString("")
	protoMust(t, err)
	stringSort, err := m.GetSort(empty)
	protoMust(t, err)
	value, err := p.SomeValue(stringSort)
	protoMust(t, err)
	if text, matched, e := m.SequenceStringValue(value); e != nil || !matched || text != "" {
		t.Fatal("string some value", text, matched, e)
	}
	first, second, ok, err := protoTestSomeValues(t, m, p, stringSort)
	protoMust(t, err)
	firstText, firstString, err := m.SequenceStringValue(first)
	protoMust(t, err)
	secondText, secondString, err := m.SequenceStringValue(second)
	protoMust(t, err)
	if !ok || !firstString || !secondString || firstText != "a" || secondText != "b" {
		t.Fatal("two string values", firstText, secondText, ok)
	}

	collision, err := m.MkString("!0!")
	protoMust(t, err)
	protoMust(t, p.RegisterValue(collision))
	fresh, err := p.FreshValue(stringSort)
	protoMust(t, err)
	if text, matched, e := m.SequenceStringValue(fresh); e != nil || !matched || text != "!!0!!" {
		t.Fatal("collision-free fresh string", text, matched, e)
	}

	boolean, err := m.MkBoolSort()
	protoMust(t, err)
	parameter, err := m.NewParameterAST(boolean)
	protoMust(t, err)
	sequenceSort, err := m.MkFamilySort(family, 0, []ast.ASTParameter{parameter})
	protoMust(t, err)
	protoMust(t, m.DelParameter(&parameter))
	first, second, ok, err = protoTestSomeValues(t, m, p, sequenceSort)
	protoMust(t, err)
	firstUnit, err := m.IsSequenceUnit(first)
	protoMust(t, err)
	secondUnit, err := m.IsSequenceUnit(second)
	protoMust(t, err)
	if !ok || !firstUnit || !secondUnit || first == second {
		t.Fatal("generic sequence values", first, second, ok)
	}

	regex, err := m.MkSequenceToRegex(empty)
	protoMust(t, err)
	regexSort, err := m.GetSort(regex)
	protoMust(t, err)
	regexValue, err := p.SomeValue(regexSort)
	protoMust(t, err)
	if yes, e := m.IsSequenceOperation(regexValue, ast.ASTSequenceOpToRegex); e != nil || !yes {
		t.Fatal("regex some value", regexValue, yes, e)
	}
	freshRegex, err := p.FreshValue(regexSort)
	protoMust(t, err)
	if yes, e := m.IsSequenceOperation(freshRegex, ast.ASTSequenceOpToRegex); e != nil || !yes {
		t.Fatal("regex fresh value", freshRegex, yes, e)
	}
}

func TestSequenceFactorySourceUnsignedWrapAndCharacterValue(t *testing.T) {
	m, p := protoFixture(t)
	family, err := m.RegisterSequencePlugin()
	protoMust(t, err)
	factory, err := NewSequenceFactory(p, family)
	protoMust(t, err)
	protoMust(t, p.RegisterFactory(factory))

	empty, err := m.MkString("")
	protoMust(t, err)
	stringSort, err := m.GetSort(empty)
	protoMust(t, err)
	factory.next = math.MaxUint32
	for _, want := range []string{"!ffffffff!", "!0!"} {
		value, err := factory.FreshValue(stringSort)
		protoMust(t, err)
		text, matched, err := m.SequenceStringValue(value)
		protoMust(t, err)
		if !matched || text != want {
			t.Fatal("source unsigned fresh-string counter", text, matched, want)
		}
	}

	characterSort, matched, err := m.SequenceSortElement(stringSort)
	protoMust(t, err)
	if !matched {
		t.Fatal("string element sort")
	}
	character, err := factory.FreshValue(characterSort)
	protoMust(t, err)
	value, err := m.NumeralService().NewValue()
	protoMust(t, err)
	defer func() { protoMust(t, m.NumeralService().Del(&value)) }()
	var width uint32
	matched, err = m.BitVectorNumeralValue(character, &value, &width)
	protoMust(t, err)
	text, err := m.NumeralService().String(&value)
	protoMust(t, err)
	if !matched || width != 8 || text != "97" {
		t.Fatal("source character fresh value", matched, width, text)
	}

	protoMust(t, factory.Close())
	if _, err = factory.FreshValue(characterSort); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed sequence factory fresh value", err)
	}
}
