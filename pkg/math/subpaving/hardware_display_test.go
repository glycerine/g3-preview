package subpaving

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareDisplayConstraintsAndLeafBoundsSourceOrder(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	x, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	y, err := c.NewVariable(false)
	if err != nil {
		t.Fatal(err)
	}
	mono, err := c.DefineMonomial([]HardwarePower{{Variable: y, Degree: 1}, {Variable: x, Degree: 2}})
	if err != nil {
		t.Fatal(err)
	}
	value := func(f float64) *numeral.HardwareFloat {
		t.Helper()
		v := new(numeral.HardwareFloat)
		if err := m.SetFloat64(v, f); err != nil {
			t.Fatal(err)
		}
		return v
	}
	poly, err := c.DefinePolynomial(value(2), []HardwarePolynomialTerm{{Variable: y, Coefficient: value(.5)}, {Variable: x, Coefficient: value(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if mono != 2 || poly != 3 {
		t.Fatal("definition variable order")
	}
	a, err := c.NewInequality(x, value(1), true, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.NewInequality(y, value(3), false, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(a, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddClause([]*HardwareInequality{b, a}, false, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddClause([]*HardwareInequality{a, b}, true, true); err != nil {
		t.Fatal(err)
	}
	name := func(w io.Writer, x uint32) error { return hardwareWrite(w, "k!"+string(rune('0'+x))) }
	var out bytes.Buffer
	if err := c.SetVariableDisplay(name); err != nil {
		t.Fatal(err)
	}
	if err := c.DisplayConstraints(&out, false); err != nil {
		t.Fatal(err)
	}
	want := "k!2 = k!0^2 k!1\nk!3 = 2 + k!0 + 1/2 k!1\n1 <= k!0\n1 <= k!0 or k!1 < 3\n"
	wantConstraints := want
	if out.String() != want {
		t.Fatalf("constraints:\n%s\nwant:\n%s", out.String(), want)
	}
	out.Reset()
	if err := c.SetVariableDisplay(nil); err != nil {
		t.Fatal(err)
	}
	if err := c.DisplayConstraints(&out, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "x2 = x0^2*x1\n") || !strings.Contains(out.String(), "x3 = 2 + x0 + 1/2*x1\n") {
		t.Fatal("default variable names and star formatting", out.String())
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	left, right, err := c.SplitMidpoint(root, x, true, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(left, x, value(-1), true, false); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(right, x, value(2), false, false); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := c.DisplayBounds(&out); err != nil {
		t.Fatal(err)
	}
	want = "-1 <= x0 x0 < 0\n=========\n0 <= x0 x0 <= 2\n"
	if out.String() != want {
		t.Fatalf("leaf bounds:\n%s\nwant:\n%s", out.String(), want)
	}
	if err := c.PropagateBound(left, x, value(-2), false, false); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := c.DisplayBounds(&out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "0 <= x0 x0 <= 2\n" {
		t.Fatal("conflicting leaf must be omitted", out.String())
	}
	budget := arena.NewBudget(1 << 20)
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	wrapper, err := NewHardwareShellCoefficients(c, qm)
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := wrapper.SetVariableDisplay(name); err != nil {
		t.Fatal(err)
	}
	if err := wrapper.DisplayConstraints(&out, false); err != nil || out.String() != wantConstraints {
		t.Fatalf("shell constraints: %v, %q", err, out.String())
	}
	if err := wrapper.SetVariableDisplay(nil); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := wrapper.DisplayBounds(&out); err != nil || out.String() != "0 <= x0 x0 <= 2\n" {
		t.Fatalf("shell bounds: %v, %q", err, out.String())
	}
	if err := qm.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatal("rational manager roots", budget.Used())
	}
	if err := c.SetVariableDisplay(name); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if c.displayVariable != nil {
		t.Fatal("closed context retained tactic display printer")
	}
	if err := wrapper.SetVariableDisplay(name); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed shell printer: %v", err)
	}
}
