package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
)

func TestSMTModelFinderInstantiationSetOwnershipInverseAndRefusals(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	zero, err := m.ArithmeticIntegerNumeralText("0")
	astTestMust(t, err)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	integer, err := m.GetSort(zero)
	astTestMust(t, err)
	set, err := newSMTModelFinderInstantiationSet(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, set.Close()) })
	inserted, err := set.Insert(zero, 4)
	astTestMust(t, err)
	if !inserted {
		t.Fatal("first instantiation term was not inserted")
	}
	inserted, err = set.Insert(zero, 1)
	astTestMust(t, err)
	if inserted || set.elements[zero] != 4 {
		t.Fatal("duplicate changed source generation", inserted, set.elements[zero])
	}
	modelValue, err := m.MkModelValue(0, integer)
	astTestMust(t, err)
	inserted, err = set.Insert(modelValue, 0)
	astTestMust(t, err)
	if inserted {
		t.Fatal("private model value entered instantiation set")
	}
	inserted, err = set.Insert(one, 2)
	astTestMust(t, err)
	if !inserted {
		t.Fatal("second instantiation term was not inserted")
	}
	proto, err := protomodel.NewProtoModel(m, config)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, proto.Close()) })
	astTestMust(t, set.MakeInverse(proto))
	term, generation, found, err := set.Inverse(one)
	astTestMust(t, err)
	if !found || term != one || generation != 2 {
		t.Fatal("inverse lookup", term, generation, found)
	}
	if err = set.Remove(one); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("term removed after inverse construction", err)
	}
	astTestMust(t, set.MakeInverse(proto))
}
