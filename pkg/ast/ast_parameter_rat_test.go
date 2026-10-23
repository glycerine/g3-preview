package ast

import (
	"math/big"
	"testing"
)

func TestASTParameterRatExactOwningCopy(t *testing.T) {
	m := astTestManager(t)
	value, ok := new(big.Rat).SetString("-1267650600228229401496703205379/17")
	if !ok {
		t.Fatal("invalid test numeral")
	}
	p, err := m.NewParameterRat(value)
	astTestMust(t, err)
	value.SetInt64(0)
	out, err := m.numerals.NewValue()
	astTestMust(t, err)
	matched, err := m.ParameterRationalCopy(p, &out)
	astTestMust(t, err)
	text, err := m.numerals.String(&out)
	astTestMust(t, err)
	if !matched || text != "-1267650600228229401496703205379/17" {
		t.Fatal(matched, text)
	}
	astTestMust(t, m.numerals.Del(&out))
	astTestMust(t, m.DelParameter(&p))
	if _, err := m.NewParameterRat(nil); err != ErrASTDomain {
		t.Fatal("nil rational", err)
	}
}
