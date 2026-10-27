package smtcore

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTStringDisplaySourceLiteralAndContextDispatch(t *testing.T) {
	m := astTestManager(t)
	theory, s := stringTheoryFixture(t, m)
	w := &theoryDisplayFlushWriter{}
	before := m.Stats()
	m.Budget().FailAfter(0)
	err := s.DisplayTheories(w)
	m.Budget().FailAfter(-1)
	astTestMust(t, err)
	const want = "TODO: theory_str display\n"
	if w.output.String() != want || w.flushes != 1 || before != m.Stats() {
		t.Fatal("source literal/endl/empty-owner dispatch", w.output.String(), w.flushes)
	}
	for cut := 0; cut < len(want); cut++ {
		out := &smtLRAOutputPrefix{remaining: cut}
		err := theory.Display(out)
		if !errors.Is(err, io.ErrClosedPipe) || out.text.String() != want[:cut] {
			t.Fatal(cut, err, out.text.String(), want[:cut])
		}
	}
	if err := theory.Display(smtLRAShortOutput{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	refused := &theoryDisplayFlushWriter{refuse: true}
	if err := theory.Display(refused); !errors.Is(err, io.ErrClosedPipe) || refused.flushes != 1 || refused.output.String() != want {
		t.Fatal("source endl refusal", err, refused.output.String(), refused.flushes)
	}
}

func TestSMTStringDisplayOwnerAndWriterGuards(t *testing.T) {
	m := astTestManager(t)
	theory, s := stringTheoryFixture(t, m)
	var out bytes.Buffer
	if err := theory.Display(nil); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	if err := (*SMTStringTheory)(nil).Display(&out); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	if err := (&SMTStringTheory{}).Display(&out); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	s.closed = true
	err := theory.Display(&out)
	s.closed = false
	if !errors.Is(err, ast.ErrASTHandle) || out.Len() != 0 {
		t.Fatal("closed context display", err, out.String())
	}
}
