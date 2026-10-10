package ast

import (
	"errors"
	"testing"
)

func TestASTProofQuantReaderNativeAppendAndBorrowedReferences(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b, c := proofTestBool(t, m, "a"), proofTestBool(t, m, "b"), proofTestBool(t, m, "c")
	f := proofTestApp(t, m, ASTOpOr, a, b)
	inputs := make([]ASTHandle, 40)
	for i := range inputs {
		inputs[i] = []ASTHandle{a, b, c}[i%3]
	}
	p, e := m.MkProofQuantInstantiation(f, inputs)
	astTestMust(t, e)
	var bindings []ASTHandle
	out := c
	before, e := m.GetRefCount(a)
	astTestMust(t, e)
	used := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	t.Cleanup(func() { m.BudgetField.FailAfter(-1) })
	yes, e := m.MatchProofQuantInstantiation(p, &out, &bindings)
	astTestMust(t, e)
	if !yes || out != f || len(bindings) != len(inputs) || m.BudgetField.Used() != used {
		t.Fatal("native binding append", yes, out, len(bindings), used, m.BudgetField.Used())
	}
	for i, want := range inputs {
		if bindings[i] != want {
			t.Fatal("binding order", i, bindings[i], want)
		}
	}
	yes, e = m.MatchProofQuantInstantiation(p, &out, &bindings)
	if ASTSourceDebug {
		if yes || !errors.Is(e, ErrASTDomain) || out != f || len(bindings) != len(inputs) {
			t.Fatal("binding.empty assertion follows fact assignment", yes, e, out, len(bindings))
		}
	} else {
		astTestMust(t, e)
		if !yes || len(bindings) != 2*len(inputs) {
			t.Fatal("release appends existing bindings", yes, len(bindings))
		}
	}
	after, e := m.GetRefCount(a)
	astTestMust(t, e)
	if before != after {
		t.Fatal("borrowed binding slice changed RC", before, after)
	}
	clear(bindings)
	bindings = bindings[:0]
	yes, e = m.MatchProofQuantInstantiation(p, &out, &bindings)
	astTestMust(t, e)
	if !yes || out != f || len(bindings) != len(inputs) {
		t.Fatal("full decoding after reset")
	}
	m.BudgetField.FailAfter(-1)
	out = c
	yes, e = m.MatchProofQuantInstantiation(a, &out, nil)
	astTestMust(t, e)
	if yes || out != c {
		t.Fatal("unmatched leaves outputs untouched")
	}
}

func TestASTProofQuantReaderMalformedParameterKeepsAppendPrefix(t *testing.T) {
	m := astTestManager(t)
	a, b := proofTestBool(t, m, "prefix-a"), proofTestBool(t, m, "prefix-b")
	f := proofTestApp(t, m, ASTOpOr, a, b)
	pa, e := m.NewParameterAST(a)
	astTestMust(t, e)
	defer m.DelParameter(&pa)
	pi, e := m.NewParameterInt(3)
	astTestMust(t, e)
	defer m.DelParameter(&pi)
	p, e := m.MkFamilyApp(0, ASTPrQuantInst, []ASTParameter{pa, pi}, []ASTHandle{f})
	astTestMust(t, e)
	var bindings []ASTHandle
	out := b
	yes, e := m.MatchProofQuantInstantiation(p, &out, &bindings)
	if yes || !errors.Is(e, ErrASTDomain) || out != f || len(bindings) != 1 || bindings[0] != a {
		t.Fatal("malformed parameter hides completed source output", yes, e, out, bindings)
	}
	out = b
	yes, e = m.MatchProofQuantInstantiation(p, &out, nil)
	if yes || !errors.Is(e, ErrASTHandle) || out != f {
		t.Fatal("nil binding adapter must follow fact assignment", yes, e, out)
	}
}
