package ast

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type proofDOTRefusingWriter struct{ writes int }

func (w *proofDOTRefusingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == 2 {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func TestASTProofDOTSourceTraversalAndSpelling(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a, b := proofTestBool(t, m, "dot-a"), proofTestBool(t, m, "dot-b")
	pa, err := m.MkProofHypothesis(a)
	astTestMust(t, err)
	implication := proofTestApp(t, m, ASTOpImplies, a, b)
	pi, err := m.MkProofHypothesis(implication)
	astTestMust(t, err)
	root, err := m.MkProofModusPonens(pa, pi)
	astTestMust(t, err)

	var out bytes.Buffer
	astTestMust(t, m.WriteProofDOT(&out, root))
	want := "digraph proof { \n" +
		"node_0 [shape=box,style=\"filled\",label=\"dot-b\"color=\"red\"]\n" +
		"node_0 -> node_1[label=\"mp\"];\n" +
		"node_0 -> node_2[label=\"mp\"];\n" +
		"node_2 [shape=box,style=\"filled\",label=\"(=> dot-a dot-b)\"color=\"yellow\"]\n" +
		"node_1 [shape=box,style=\"filled\",label=\"dot-a\"color=\"yellow\"]\n" +
		"\n } \n"
	if got := out.String(); got != want {
		t.Fatalf("source DOT: got %q, want %q", got, want)
	}
}

func TestASTProofDOTEscapeAndGuards(t *testing.T) {
	if got := escapeDOT("a\nb\n"); got != `a\lb\l` {
		t.Fatalf("escape: %q", got)
	}
	m := astTestManager(t)
	a := proofTestBool(t, m, "dot-guard")
	if err := m.WriteProofDOT(io.Discard, a); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("non-proof guard: %v", err)
	}
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	p, err := m.MkProofAsserted(a)
	astTestMust(t, err)
	if err = m.WriteProofDOT(nil, p); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("nil output guard: %v", err)
	}
	refusing := new(proofDOTRefusingWriter)
	if err = m.WriteProofDOT(refusing, p); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("writer refusal: %v", err)
	}
}
