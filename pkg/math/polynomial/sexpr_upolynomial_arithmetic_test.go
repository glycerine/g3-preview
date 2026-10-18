package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestSexprPolynomialArithmeticRefusalRetainsOlderArguments(t *testing.T) {
	for _, op := range []string{"+", "-", "*", "^"} {
		t.Run(op, func(t *testing.T) {
			sx, sy, globals := sxOwner(t, arena.NewBudget(1<<24))
			target := algebraicCopyBoundedTargetSlots(t, 10)
			m := target.config.Polynomials
			out, e := m.NewPolynomial()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&out)) })
			c, e := NewSexprPolynomialConverter(sx, m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, c.Close()) })
			other, e := NewSexprPolynomialConverter(sx, m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, other.Close()) })
			a := sxPolyNum(t, sx, globals, "1267650600228229401496703205377")
			b := sxPolyNum(t, sx, globals, "633825300114114700748351602691")
			expr := sxPolyApp(t, sx, sy, op, a, b)
			if op == "^" {
				two := sxPolyNum(t, sx, globals, "2")
				one := sxPolyNum(t, sx, globals, "1")
				expr = sxPolyApp(t, sx, sy, "+", one, sxPolyApp(t, sx, sy, "^", a, two))
			}
			body := c.Convert(expr, out)
			if !errors.Is(body, arena.ErrSlotLimit) || !c.arithmetic.active || !c.arithmetic.pending() || m.pendingSexprArithmetic != &c.arithmetic {
				t.Fatal("operator did not retain its issued numerical child", body)
			}
			if len(c.frames) != 1 || !c.frames[0].live || c.depth != 0 {
				t.Fatal("operator failure destroyed an older argument vector")
			}
			if m.config.Integer.PendingCell.Object == (umemory.ObjectRef{}) {
				t.Fatal("descriptor refusal did not retain physical storage")
			}
			physical := m.config.Integer.PendingCell
			serviceMust(t, other.Close())
			if m.pendingSexprArithmetic != &c.arithmetic || m.config.Integer.PendingCell != physical {
				t.Fatal("idle converter acquired another converter's retained arithmetic")
			}
			for _, e := range []error{m.SourceFinalize(), m.Close(), m.DelPolynomial(&out)} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Fatal("pending arithmetic lost borrowed manager", e)
				}
			}
			serviceMust(t, c.Cleanup())
			if c.arithmetic.active || c.frames[0].live || m.pendingSexprArithmetic != nil || m.sexprArithmetic != nil || m.config.Integer.PendingCell.Object != (umemory.ObjectRef{}) {
				t.Fatal("operator cleanup retained its child/vector/manager borrow")
			}
		})
	}
}
