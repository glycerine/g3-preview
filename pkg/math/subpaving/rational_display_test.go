package subpaving

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalDisplayDefinitionsClausesLeavesAndWriterPrefix(t *testing.T) {
	c, m, value, x, y := rationalClauseFixture(t)
	if _, err := c.DefineMonomial([]RationalPower{{Variable: y, Degree: 1}, {Variable: x, Degree: 2}}); err != nil {
		t.Fatal(err)
	}
	constant := numeral.NewRational()
	t.Cleanup(func() {
		if err := m.Del(&constant); err != nil {
			t.Error(err)
		}
	})
	if err := m.SetInt32(&constant, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DefinePolynomial(&constant, []numeral.Rational{constant, constant}, []uint32{x, y}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 1); err != nil {
		t.Fatal(err)
	}
	lower, err := c.NewInequality(x, value, true, false)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := c.NewInequality(y, value, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(lower, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddClause([]*RationalInequality{upper, lower}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := c.DisplayConstraints(&out, false); err != nil {
		t.Fatal(err)
	}
	want := "x2 = x0^2 x1\nx3 = 1 + x0 + x1\n1 <= x0\n1 <= x0 or x1 <= 1\n"
	if out.String() != want {
		t.Fatal("source constraint order", out.String())
	}
	out.Reset()
	if err := c.DisplayConstraints(&out, true); err != nil || out.String() != "x2 = x0^2*x1\nx3 = 1 + x0 + x1\n1 <= x0\n1 <= x0 or x1 <= 1\n" {
		t.Fatal("star spelling", out.String(), err)
	}
	for limit := 0; limit <= len(want); limit++ {
		short := &arbitraryShortWriter{limit: limit}
		err := c.DisplayConstraints(short, false)
		if short.buf.String() != want[:limit] || limit < len(want) && !errors.Is(err, io.ErrShortWrite) || limit == len(want) && err != nil {
			t.Fatal("writer prefix", limit, short.buf.String(), err)
		}
	}
	if err := c.DisplayBounds(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil bounds writer", err)
	}
	if err := c.InitializeRoot(); err != nil {
		t.Fatal(err)
	}
	left, err := c.NewChild(c.root)
	if err != nil {
		t.Fatal(err)
	}
	right, err := c.NewChild(c.root)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 2); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(left, x, value, false, false); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 3); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishBound(right, x, value, true, false); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := c.DisplayBounds(&out); err != nil || out.String() != "1 <= x0 x0 <= 2\n=========\n3 <= x0 \n" {
		t.Fatal("source leaf tree order", out.String(), err)
	}
}
