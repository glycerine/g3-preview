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

type fastPrefixWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *fastPrefixWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.buf.Len()
	if remaining < 0 {
		remaining = 0
	}
	if remaining < len(p) {
		p = p[:remaining]
	}
	return w.buf.Write(p)
}

func TestFastDisplayBoundsLeavesConflictAndWriterPrefixes(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFastContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewFastShellCoefficients(c, qm)
	if err != nil {
		t.Fatal(err)
	}
	valueOwner, err := numeral.NewScopedFastFloat(m)
	if err != nil {
		t.Fatal(err)
	}
	value, err := valueOwner.Get()
	if err != nil {
		t.Fatal(err)
	}
	var left, right *FastNode
	t.Cleanup(func() {
		if right != nil {
			if err := right.Close(); err != nil {
				t.Error(err)
			}
		}
		if left != nil {
			if err := left.Close(); err != nil {
				t.Error(err)
			}
		}
		for _, err := range []error{w.Close(), c.Close(), valueOwner.Close(), qm.CloseChecked(), m.CloseChecked()} {
			if err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("retained fast display owners", b.Used())
		}
	})
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := c.DisplayBounds(&out); err != nil || out.Len() != 0 {
		t.Fatal("empty root", out.String(), err)
	}
	stoppedVariable := errors.New("stopped variable printer")
	if err := c.SetVariableDisplay(func(io.Writer, uint32) error { return stoppedVariable }); err != nil {
		t.Fatal(err)
	}
	if err := c.displayRelation(&out, 0, nil, false, false); !errors.Is(err, stoppedVariable) || out.Len() != 0 {
		t.Fatal("upper variable must precede numeral formatting", out.String(), err)
	}
	if err := c.SetVariableDisplay(nil); err != nil {
		t.Fatal(err)
	}
	if err := c.displayRelation(&out, 0, nil, false, false); !errors.Is(err, arena.ErrInvalidSlot) || out.String() != "x0 <= " {
		t.Fatal("upper numeral refusal must retain relation prefix", out.String(), err)
	}
	out.Reset()
	if err := c.displayRelation(&out, 0, nil, true, false); !errors.Is(err, arena.ErrInvalidSlot) || out.Len() != 0 {
		t.Fatal("lower numeral must precede relation output", out.String(), err)
	}
	set := func(n int32) {
		t.Helper()
		if err := m.SetInt32(value, n); err != nil {
			t.Fatal(err)
		}
	}
	set(1)
	if err := c.PropagateBound(root, 0, value, true, false); err != nil {
		t.Fatal(err)
	}
	set(5)
	if err := c.PropagateBound(root, 0, value, false, false); err != nil {
		t.Fatal(err)
	}
	left, err = c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	right, err = c.NewChild(root)
	if err != nil {
		t.Fatal(err)
	}
	set(4)
	if err := c.DecideBound(left, 0, value, false, true); err != nil {
		t.Fatal(err)
	}
	set(3)
	if err := c.DecideBound(right, 0, value, true, false); err != nil {
		t.Fatal(err)
	}
	want := "1 <= x0 x0 < 4\n=========\n3 <= x0 x0 <= 5\n"
	if err := w.DisplayBounds(&out); err != nil || out.String() != want {
		t.Fatal("leaf order", out.String(), err)
	}
	for limit := 0; limit <= len(want); limit++ {
		short := &fastPrefixWriter{limit: limit}
		err := c.DisplayBounds(short)
		if short.buf.String() != want[:limit] || (limit < len(want) && !errors.Is(err, io.ErrShortWrite)) || (limit == len(want) && err != nil) {
			t.Fatal("prefix", limit, short.buf.String(), err)
		}
	}
	set(6)
	if err := c.PropagateBound(left, 0, value, true, false); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := c.DisplayBounds(&out); err != nil || out.String() != "3 <= x0 x0 <= 5\n" {
		t.Fatal("conflicting leaf", out.String(), err)
	}
	if err := c.DisplayBounds(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
	if err := left.Close(); err != nil {
		t.Fatal(err)
	}
	if err := right.Close(); err != nil {
		t.Fatal(err)
	}
	if len(c.nodes) != 1 || c.nodes[0] != root {
		t.Fatal("closed child retention", len(c.nodes))
	}
	out.Reset()
	if err := c.DisplayBounds(&out); err != nil || out.String() != "1 <= x0 x0 <= 5\n" {
		t.Fatal("closed children leave root", out.String(), err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.DisplayBounds(&out); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed context display", err)
	}
	if err := w.DisplayBounds(&out); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed shell display", err)
	}
}
