package protomodel

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestStringFactoryNativeDefaultsFreshNamesAndIgnoredRegistration(t *testing.T) {
	m, p := protoFixture(t)
	family, err := m.RegisterSequencePlugin()
	protoMust(t, err)
	factory, err := NewStringFactory(p, family)
	protoMust(t, err)
	protoMust(t, p.RegisterFactory(factory))
	empty, err := m.MkString("")
	protoMust(t, err)
	sort, err := m.GetSort(empty)
	protoMust(t, err)
	value, err := p.SomeValue(sort)
	protoMust(t, err)
	first, second, ok, err := protoTestSomeValues(t, m, p, sort)
	protoMust(t, err)
	if !ok {
		t.Fatal("string factory could not provide two values")
	}
	for i, expression := range []ast.ASTHandle{value, first, second} {
		text, constant, err := m.SequenceStringValue(expression)
		protoMust(t, err)
		if !constant || text != []string{"some value", "value 1", "value 2"}[i] {
			t.Fatal("string backend default value", i, text)
		}
	}
	registered, err := m.MkString("!0!")
	protoMust(t, err)
	protoMust(t, p.RegisterValue(registered))
	fresh, err := p.FreshValue(sort)
	protoMust(t, err)
	if fresh != registered {
		t.Fatal("string backend unexpectedly adopted sequence-factory registration/collision policy")
	}
	regex, err := m.MkSequenceToRegex(empty)
	protoMust(t, err)
	regexSort, err := m.GetSort(regex)
	protoMust(t, err)
	freshRegex, err := p.FreshValue(regexSort)
	protoMust(t, err)
	argument, matched, err := m.SequenceUnaryOperationArgument(freshRegex, ast.ASTSequenceOpToRegex)
	protoMust(t, err)
	text, constant, err := m.SequenceStringValue(argument)
	protoMust(t, err)
	if !matched || !constant || text != "!1!" {
		t.Fatal("fresh regex did not share the string generator", text)
	}
	for i := 2; i < 18; i++ {
		fresh, err = p.FreshValue(sort)
		protoMust(t, err)
	}
	text, _, err = m.SequenceStringValue(fresh)
	protoMust(t, err)
	if text != "!11!" {
		t.Fatal("fresh counter is not hexadecimal", text)
	}
	factory.next = ^uint32(0)
	_, err = p.FreshValue(sort)
	protoMust(t, err)
	fresh, err = p.FreshValue(sort)
	protoMust(t, err)
	text, _, err = m.SequenceStringValue(fresh)
	protoMust(t, err)
	if text != "!12!" {
		t.Fatal("wrapped source counter did not skip previously generated names", text)
	}
	protoMust(t, factory.Close())
	protoMust(t, factory.Close())
}
