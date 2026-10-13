package subpaving

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type arbitraryShortWriter struct {
	buf   bytes.Buffer
	limit int
}

func TestArbitraryDisplayConstraintsDefinitionsUnitsAndClauses(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	monomial, err := c.DefineMonomial([]ArbitraryPower{{Variable: f.y, Degree: 1}, {Variable: f.x, Degree: 2}})
	if err != nil || monomial != 2 {
		t.Fatal("monomial declaration", monomial, err)
	}
	polynomial, err := c.DefinePolynomial(&f.four, []numeral.ArbitraryFloat{f.five, f.four}, []uint32{f.x, f.y})
	if err != nil || polynomial != 3 {
		t.Fatal("polynomial declaration", polynomial, err)
	}
	lower, err := c.NewInequality(f.x, &f.four, true, false)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := c.NewInequality(f.y, &f.five, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(lower, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddClause([]*ArbitraryInequality{upper, lower}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := c.DisplayConstraints(&out, false); err != nil {
		t.Fatal(err)
	}
	want := "x2 = x0^2 x1\nx3 = 4 + 5 x0 + 4 x1\n4 <= x0\n4 <= x0 or x1 <= 5\n"
	if out.String() != want {
		t.Fatalf("source constraint order: %q", out.String())
	}
	w, err := NewArbitraryShellCoefficients(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	out.Reset()
	if err := w.DisplayConstraints(&out, true); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "x2 = x0^2*x1\nx3 = 4 + 5*x0 + 4*x1\n4 <= x0\n4 <= x0 or x1 <= 5\n" {
		t.Fatalf("star spelling: %q", got)
	}
	for limit := 0; limit <= len(want); limit++ {
		short := &arbitraryShortWriter{limit: limit}
		err := c.DisplayConstraints(short, false)
		if short.buf.String() != want[:limit] {
			t.Fatalf("write limit %d: %q", limit, short.buf.String())
		}
		if limit == len(want) && err != nil || limit < len(want) && !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("write limit %d: %v", limit, err)
		}
	}
}

func (w *arbitraryShortWriter) Write(p []byte) (int, error) {
	available := w.limit - w.buf.Len()
	if available < 0 {
		available = 0
	}
	if available < len(p) {
		return w.buf.Write(p[:available])
	}
	return w.buf.Write(p)
}

func TestArbitraryDisplayBoundsTreeOrderConflictAndWriterPrefixes(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	c := f.context
	var empty bytes.Buffer
	if err := c.DisplayBounds(&empty); err != nil || empty.Len() != 0 {
		t.Fatal("uninitialized source tree", err, empty.String())
	}
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(root, f.y, &f.five, false, false, true); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	left, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	right, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DecideBound(left, f.x, &f.four, false, true); err != nil {
		t.Fatal(err)
	}
	if err := c.DecideBound(right, f.x, &f.four, true, false); err != nil {
		t.Fatal(err)
	}
	want := "x0 < 4\nx1 <= 5\n=========\n4 <= x0 \nx1 <= 5\n"
	var out bytes.Buffer
	if err := c.DisplayBounds(&out); err != nil || out.String() != want {
		t.Fatalf("source leaf order: %v, %q", err, out.String())
	}
	if err := c.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		return arbitraryWrite(w, "v"+string(rune('0'+variable)))
	}); err != nil {
		t.Fatal(err)
	}
	w, err := NewArbitraryShellCoefficients(c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	out.Reset()
	if err := w.DisplayBounds(&out); err != nil || out.String() != "v0 < 4\nv1 <= 5\n=========\n4 <= v0 \nv1 <= 5\n" {
		t.Fatalf("borrowed shell printer: %v, %q", err, out.String())
	}
	if err := w.SetVariableDisplay(nil); err != nil {
		t.Fatal(err)
	}
	for limit := 0; limit <= len(want); limit++ {
		short := &arbitraryShortWriter{limit: limit}
		err := c.DisplayBounds(short)
		if short.buf.String() != want[:limit] {
			t.Fatalf("write limit %d: %q", limit, short.buf.String())
		}
		if limit == len(want) && err != nil || limit < len(want) && !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("write limit %d: %v", limit, err)
		}
	}
	if err := c.PublishBound(left, f.x, &f.five, true, false, true); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := c.DisplayBounds(&out); err != nil || out.String() != "4 <= x0 \nx1 <= 5\n" {
		t.Fatalf("conflicting leaf omitted: %v, %q", err, out.String())
	}
	if err := c.DisplayBounds(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil || c.displayVariable != nil {
		t.Fatal("closed printer owner", err)
	}
	if err := c.SetVariableDisplay(nil); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed context printer", err)
	}
	if err := w.DisplayBounds(&out); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed shell display", err)
	}
}
