package tactic

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

type selectingProofConverter struct {
	proofConverterBase
	manager *ast.ASTManager
	index   int
	calls   []int
}

func newSelectingProofConverter(t *testing.T, m *ast.ASTManager, index int) *selectingProofConverter {
	t.Helper()
	c := &selectingProofConverter{manager: m, index: index}
	astTestMust(t, c.init(c, m.Budget(), 1))
	return c
}

func (c *selectingProofConverter) Convert(m *ast.ASTManager, sources []ast.ASTHandle, out ast.ASTRef) error {
	if err := c.open(); err != nil {
		return err
	}
	if m != c.manager || c.index < 0 || c.index >= len(sources) {
		return ast.ErrASTDomain
	}
	c.calls = append(c.calls, len(sources))
	return m.SetRef(out, sources[c.index])
}
func (c *selectingProofConverter) Translate(*ast.ASTTranslation) (ProofConverter, error) {
	return nil, ast.ErrASTDomain
}
func (c *selectingProofConverter) Display(w io.Writer) error {
	if err := c.open(); err != nil {
		return err
	}
	_, err := io.WriteString(w, "selecting-proof-converter")
	return err
}
func (c *selectingProofConverter) Close() error {
	if c == nil || c.self != &c.proofConverterBase {
		return ast.ErrASTHandle
	}
	if c.closed {
		return nil
	}
	c.release()
	return nil
}

func proofConverterProof(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	fact := proofTestBool(t, m, name)
	astTestPin(t, m, fact)
	proof, err := m.MkProofAsserted(fact)
	astTestMust(t, err)
	return proof
}

func TestStoredProofConverterConvertDisplayAndLifetime(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	proof := proofConverterProof(t, m, "stored-proof")
	c, err := NewStoredProofConverter(m, proof)
	astTestMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()

	astTestMust(t, c.Convert(m, nil, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != proof {
		t.Fatalf("stored proof = %v, want %v", got, proof)
	}
	if err = c.Convert(m, []ast.ASTHandle{proof}, out); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatalf("nonempty source error = %v", err)
	}
	var text bytes.Buffer
	astTestMust(t, c.Display(&text))
	if !strings.Contains(text.String(), "proof->proof-converter-wrapper") {
		t.Fatalf("display = %q", text.String())
	}
	astTestMust(t, c.IncRef())
	astTestMust(t, c.DecRef())
	if err = c.Cancel(); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("retired converter error = %v", err)
	}
}

func TestConcatProofConvertersApplyInSourceOrder(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	p0 := proofConverterProof(t, m, "concat-0")
	p1 := proofConverterProof(t, m, "concat-1")
	first := newSelectingProofConverter(t, m, 0)
	second := newSelectingProofConverter(t, m, 1)
	c, err := ConcatProofConverters(m.Budget(), first, second)
	astTestMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()

	astTestMust(t, c.Convert(m, []ast.ASTHandle{p0, p1}, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != p1 || len(second.calls) != 1 || second.calls[0] != 2 || len(first.calls) != 1 || first.calls[0] != 1 {
		t.Fatalf("got=%v first calls=%v second calls=%v", got, first.calls, second.calls)
	}
	astTestMust(t, c.Close())
	if !first.closed || !second.closed {
		t.Fatal("concat did not release retained converters")
	}
}

func TestConcatStarProofConvertersPartitionsSources(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	p0 := proofConverterProof(t, m, "star-0")
	p1 := proofConverterProof(t, m, "star-1")
	p2 := proofConverterProof(t, m, "star-2")
	first := newSelectingProofConverter(t, m, 1)
	second := newSelectingProofConverter(t, m, 1)
	c, err := ConcatStarProofConverters(m.Budget(), first, []ProofConverter{nil, second}, []uint32{1, 2})
	astTestMust(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()

	astTestMust(t, c.Convert(m, []ast.ASTHandle{p0, p1, p2}, out))
	got, err := m.RefGet(out)
	astTestMust(t, err)
	if got != p2 || len(second.calls) != 1 || second.calls[0] != 2 || len(first.calls) != 1 || first.calls[0] != 2 {
		t.Fatalf("got=%v first calls=%v second calls=%v", got, first.calls, second.calls)
	}
	if err = c.Convert(m, []ast.ASTHandle{p0}, out); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatalf("bad partition error = %v", err)
	}
	astTestMust(t, c.Close())
	if !first.closed || !second.closed {
		t.Fatal("concat-star did not release retained converters")
	}
}

func TestApplyProofConvertersAndStoredTranslation(t *testing.T) {
	from := astTestManager(t)
	to := astTestManager(t)
	astTestMust(t, from.ToggleProofMode(ast.ASTProofEnabled))
	astTestMust(t, to.ToggleProofMode(ast.ASTProofEnabled))
	p0 := proofConverterProof(t, from, "apply-0")
	p1 := proofConverterProof(t, from, "apply-1")
	c0, err := NewStoredProofConverter(from, p0)
	astTestMust(t, err)
	c1, err := NewStoredProofConverter(from, p1)
	astTestMust(t, err)
	first := newSelectingProofConverter(t, from, 1)
	out, err := from.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, from.DelRef(&out)) }()
	astTestMust(t, ApplyProofConverters(from, first, []ProofConverter{c0, c1}, out))
	got, err := from.RefGet(out)
	astTestMust(t, err)
	if got != p1 {
		t.Fatalf("applied proof = %v, want %v", got, p1)
	}

	tr, err := ast.NewASTTranslation(from, to, true)
	astTestMust(t, err)
	defer func() { astTestMust(t, tr.Close()) }()
	translated, err := c1.Translate(tr)
	astTestMust(t, err)
	target, err := to.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, to.DelRef(&target)) }()
	astTestMust(t, translated.Convert(to, nil, target))
	targetProof, err := to.RefGet(target)
	astTestMust(t, err)
	targetFact, err := to.GetProofFact(targetProof)
	astTestMust(t, err)
	sourceFact, err := from.GetProofFact(p1)
	astTestMust(t, err)
	wantFact, err := tr.Translate(sourceFact)
	astTestMust(t, err)
	if targetFact != wantFact {
		t.Fatalf("translated fact = %v, want %v", targetFact, wantFact)
	}

	astTestMust(t, translated.Close())
	astTestMust(t, first.Close())
	astTestMust(t, c1.Close())
	astTestMust(t, c0.Close())
}
