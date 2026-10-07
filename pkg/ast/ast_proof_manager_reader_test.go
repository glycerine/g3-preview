package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTProofQuantReaderAppendAndFailurePrefixes(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b, c := proofTestBool(t, m, "a"), proofTestBool(t, m, "b"), proofTestBool(t, m, "c")
	f := proofTestApp(t, m, ASTOpOr, a, b)
	p, e := m.MkProofQuantInstantiation(f, []ASTHandle{a, b, c})
	astTestMust(t, e)
	v, e := NewASTProofBindings(m)
	astTestMust(t, e)
	defer v.Close()
	out := c
	m.BudgetField.FailAfter(1)
	yes, e := m.MatchProofQuantInstantiation(p, &out, v)
	if yes || !errors.Is(e, arena.ErrInjectedFailure) || out != f || v.Len() != 2 {
		t.Fatal("source output and append prefix", yes, e, out, v.Len())
	}
	for i, want := range []ASTHandle{a, b} {
		x, e := v.At(uint32(i))
		astTestMust(t, e)
		if x != want {
			t.Fatal("binding order", i, x, want)
		}
	}
	// The output is nonowning: app construction retained the actual parameter
	// references, while reader append has no extra inc_ref callbacks.
	before, e := m.GetRefCount(a)
	astTestMust(t, e)
	m.BudgetField.FailAfter(-1)
	yes, e = m.MatchProofQuantInstantiation(p, &out, v)
	if ASTSourceDebug {
		if yes || !errors.Is(e, ErrASTDomain) || out != f || v.Len() != 2 {
			t.Fatal("source binding.empty assertion after fact assignment", yes, e, out, v.Len())
		}
	} else {
		astTestMust(t, e)
		if !yes || v.Len() != 5 {
			t.Fatal("release appends existing bindings", yes, v.Len())
		}
	}
	after, e := m.GetRefCount(a)
	astTestMust(t, e)
	if before != after {
		t.Fatal("nonowning pointer vector changed RC", before, after)
	}
	astTestMust(t, v.Reset())
	yes, e = m.MatchProofQuantInstantiation(p, &out, v)
	astTestMust(t, e)
	if !yes || out != f || v.Len() != 3 {
		t.Fatal("full decoding")
	}
	x, e := v.At(2)
	astTestMust(t, e)
	if x != c {
		t.Fatal("third binding")
	}
	out = c
	yes, e = m.MatchProofQuantInstantiation(a, &out, nil)
	astTestMust(t, e)
	if yes || out != c {
		t.Fatal("unmatched leaves outputs untouched")
	}
}
