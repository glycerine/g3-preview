package ast

// Source: src/ast/ast.cpp:734-745; src/util/buffer.h.

import "testing"

func TestASTProofDeclarationNativeDomainSliceOrderAndGrowth(t *testing.T) {
	m := astTestManager(t)
	proofSort := m.external(m.basic.proofSort)
	boolSort := m.external(m.basic.boolSort)
	for _, parents := range []uint32{14, 15, 16, 32} {
		m.mu.Lock()
		r, e := m.proofDecl(ASTPrTheoryLemma, nil, parents)
		m.mu.Unlock()
		astTestMust(t, e)
		h := m.external(r)
		astTestPin(t, m, h)
		arity, e := m.GetArity(h)
		astTestMust(t, e)
		if arity != parents+1 {
			t.Fatal("proof arity", parents, arity)
		}
		for i := uint32(0); i < arity; i++ {
			got, err := m.GetDomain(h, i)
			astTestMust(t, err)
			want := proofSort
			if i == parents {
				want = boolSort
			}
			if got != want {
				t.Fatal("proof domain order", parents, i, got, want)
			}
		}
	}
}
