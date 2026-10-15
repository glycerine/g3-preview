package subpaving

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func fixedDisplayFixture(t *testing.T) (*numeral.FixedPointManager, *numeral.RationalManager, *FixedContext, *FixedShellCoefficients) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, nil, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewFixedShellCoefficients(c, qm)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{w.Close(), c.Close(), qm.CloseChecked(), m.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("fixed display owner leak", b.Used())
		}
	})
	return m, qm, c, w
}

func TestFixedDisplayConstraintsSourceDefinitionsAndClause(t *testing.T) {
	_, qm, c, w := fixedDisplayFixture(t)
	x, err := w.MkVar(false)
	if err != nil {
		t.Fatal(err)
	}
	y, err := w.MkVar(true)
	if err != nil {
		t.Fatal(err)
	}
	one, two, three := numeral.NewInt(1), numeral.NewInt(2), numeral.NewInt(3)
	if _, err := w.MkSum(&three, []*numeral.Int{&one, &two}, []uint32{x, y}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.MkMonomial([]FixedPower{{Variable: x, Degree: 2}, {Variable: y, Degree: 1}}); err != nil {
		t.Fatal(err)
	}
	bound := numeral.NewRational()
	defer func() {
		if err := qm.Del(&bound); err != nil {
			t.Error(err)
		}
	}()
	if err := qm.SetInt32(&bound, 1); err != nil {
		t.Fatal(err)
	}
	lower, err := w.MkInequality(x, &bound, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := qm.SetInt32(&bound, 2); err != nil {
		t.Fatal(err)
	}
	upper, err := w.MkInequality(y, &bound, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.AddClause([]*FixedInequality{lower, upper}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := w.DisplayConstraints(&out, true); err != nil {
		t.Fatal(err)
	}
	if want := "x2 = 3 + x0 + 2*x1\nx3 = x0^2*x1\n1 <= x0 or x1 < 2\n"; out.String() != want {
		t.Fatal("source constraints", out.String())
	}
	if err := c.SetVariableDisplay(func(out io.Writer, variable uint32) error {
		_, err := io.WriteString(out, "v")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := w.DisplayConstraints(&out, false); err != nil || out.String() != "v = 3 + v + 2 v\nv = v^2 v\n1 <= v or v < 2\n" {
		t.Fatal("borrowed variable printer and space separator", out.String(), err)
	}
}

func TestFixedDisplayBoundsSearchOrderAndWriterPrefixes(t *testing.T) {
	_, _, c, w := fixedDisplayFixture(t)
	if _, err := w.MkVar(false); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := w.DisplayBounds(&out); err != nil || out.Len() != 0 {
		t.Fatal("empty root bounds", out.String(), err)
	}
	if err := c.displayRelation(&out, 0, nil, false, false); !errors.Is(err, numeral.ErrFixedPointOwner) || out.String() != "x0 <= " {
		t.Fatal("upper relation prefix on numeral refusal", out.String(), err)
	}
	out.Reset()
	if err := c.displayRelation(&out, 0, nil, true, false); !errors.Is(err, numeral.ErrFixedPointOwner) || out.Len() != 0 {
		t.Fatal("lower numeral precedes output", out.String(), err)
	}
	short := &fastPrefixWriter{limit: 2}
	if err := w.DisplayBounds(short); err != nil || short.buf.Len() != 0 {
		t.Fatal("empty bounds writer", err)
	}
	c.maxDepth = 1
	if err := w.Search(); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := w.DisplayBounds(&out); err != nil || out.String() != "x0 < 0\n=========\n0 <= x0 \n" {
		t.Fatal("searched leaf display order", out.String(), err)
	}
	short = &fastPrefixWriter{limit: 2}
	if err := w.DisplayBounds(short); !errors.Is(err, io.ErrShortWrite) || short.buf.String() != "x0" {
		t.Fatal("writer prefix", short.buf.String(), err)
	}
}
