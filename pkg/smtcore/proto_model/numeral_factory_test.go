package protomodel

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestArithmeticFactoryPreservesRequestedNumeralSort(t *testing.T) {
	m, p := protoFixture(t)
	integerZero, err := m.ArithmeticIntegerNumeralText("0")
	protoMust(t, err)
	realZero, err := m.ArithmeticRealNumeralText("0")
	protoMust(t, err)
	protoPin(t, m, integerZero)
	protoPin(t, m, realZero)
	factory, err := NewArithmeticFactory(m)
	protoMust(t, err)
	protoMust(t, p.RegisterFactory(factory))
	for _, zero := range []ast.ASTHandle{integerZero, realZero} {
		sort, err := m.GetSort(zero)
		protoMust(t, err)
		first, err := p.SomeValue(sort)
		protoMust(t, err)
		fresh, err := p.FreshValue(sort)
		protoMust(t, err)
		left, right, distinct, err := protoTestSomeValues(t, m, p, sort)
		protoMust(t, err)
		if first != zero || first == fresh || !distinct || left == right {
			t.Fatal("arithmetic value generation", first, zero, fresh, left, right, distinct)
		}
		for _, value := range []ast.ASTHandle{first, fresh, left, right} {
			actual, err := m.GetSort(value)
			protoMust(t, err)
			if actual != sort {
				t.Fatal("arithmetic factory returned a value of the wrong sort", actual, sort)
			}
		}
	}
}
