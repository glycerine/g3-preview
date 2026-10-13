package subpaving

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwarePolynomialForwardInverseAndUnboundedSelection(t *testing.T) {
	type fixture struct {
		c      *HardwareContext
		m      *numeral.HardwareFloatNumeralManager
		n      *HardwareNode
		x0, x1 uint32
		output uint32
	}
	setup := func(t *testing.T) fixture {
		t.Helper()
		m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
		if err != nil {
			t.Fatal(err)
		}
		c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 0, 2, 8192)
		if err != nil {
			t.Fatal(err)
		}
		x0, err := c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
		x1, err := c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
		var constant, two, three numeral.HardwareFloat
		for _, entry := range []struct {
			out   *numeral.HardwareFloat
			value int32
		}{{&constant, 5}, {&two, 2}, {&three, 3}} {
			if err := m.SetInt32(entry.out, entry.value); err != nil {
				t.Fatal(err)
			}
		}
		output, err := c.DefinePolynomial(&constant, []HardwarePolynomialTerm{{Variable: x0, Coefficient: &two}, {Variable: x1, Coefficient: &three}})
		if err != nil {
			t.Fatal(err)
		}
		n, err := c.CreateRootNode()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := c.Close(); err != nil {
				t.Error(err)
			}
		})
		return fixture{c: c, m: m, n: n, x0: x0, x1: x1, output: output}
	}
	bound := func(t *testing.T, f fixture, variable uint32, lo, hi int32) {
		t.Helper()
		var lower, upper numeral.HardwareFloat
		if err := f.m.SetInt32(&lower, lo); err != nil {
			t.Fatal(err)
		}
		if err := f.m.SetInt32(&upper, hi); err != nil {
			t.Fatal(err)
		}
		if err := f.c.PropagateBound(f.n, variable, &lower, true, false); err != nil {
			t.Fatal(err)
		}
		if err := f.c.PropagateBound(f.n, variable, &upper, false, false); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("forward-ignores-stored-constant", func(t *testing.T) {
		f := setup(t)
		bound(t, f, f.x0, 1, 2)
		bound(t, f, f.x1, 3, 4)
		if err := f.c.PropagatePolynomial(f.output, f.n); err != nil {
			t.Fatal(err)
		}
		l, u := f.n.lower[f.output], f.n.upper[f.output]
		if l == nil || u == nil || f.m.ToFloat64(&l.value) != 11 || f.m.ToFloat64(&u.value) != 16 ||
			l.reason != hardwareReasonVariableDefinition || u.reasonVariable != f.output || f.c.polynomials[f.output].visited != 4 {
			t.Fatal("source forward sum omits stored constant and retains definition reason")
		}
		before := len(f.c.queue)
		if err := f.c.PropagatePolynomial(f.output, f.n); err != nil || len(f.c.queue) != before {
			t.Fatal("all-bounded pass must reject weaker candidates", err)
		}
	})
	t.Run("inverse-directed-rational-bounds", func(t *testing.T) {
		f := setup(t)
		bound(t, f, f.x0, 1, 2)
		bound(t, f, f.output, 11, 16)
		if err := f.c.PropagatePolynomial(f.output, f.n); err != nil {
			t.Fatal(err)
		}
		l, u := f.n.lower[f.x1], f.n.upper[f.x1]
		if l == nil || u == nil ||
			new(big.Rat).SetFloat64(f.m.ToFloat64(&l.value)).Cmp(big.NewRat(7, 3)) > 0 ||
			new(big.Rat).SetFloat64(f.m.ToFloat64(&u.value)).Cmp(big.NewRat(14, 3)) < 0 ||
			l.reason != hardwareReasonVariableDefinition || u.reasonVariable != f.output {
			t.Fatal("source inverse sum must outwardly enclose rational target")
		}
	})
	t.Run("multiple-unbounded-variables", func(t *testing.T) {
		f := setup(t)
		bound(t, f, f.x0, 1, 2)
		before := len(f.c.queue)
		if err := f.c.PropagatePolynomial(f.output, f.n); err != nil {
			t.Fatal(err)
		}
		if len(f.c.queue) != before || f.n.lower[f.output] != nil || f.n.lower[f.x1] != nil {
			t.Fatal("source refuses two unbounded polynomial variables")
		}
	})
}
