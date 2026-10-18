package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
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

func TestSexprPolynomialPrivateArithmeticAllocationPrefixes(t *testing.T) {
	failures, successes := 0, 0
	for fault := int64(0); fault < 512; fault++ {
		done := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			sx, sy, globals := sxOwner(t, arena.NewBudget(1<<24))
			target := algebraicCopyBoundedTargetSlots(t, 512)
			m, z := target.config.Polynomials, target.config.Rationals.Integers()
			out, e := m.NewPolynomial()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&out)) })
			c, e := NewSexprPolynomialConverter(sx, m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, c.Close()) })
			x := sxPolySym(t, sx, sy, "x")
			a := sxPolyNum(t, sx, globals, "1267650600228229401496703205377")
			three := sxPolyNum(t, sx, globals, "3")
			seven := sxPolyNum(t, sx, globals, "7")
			two := sxPolyNum(t, sx, globals, "2")
			// (x+a)^3 -(x+a) +(x-7)*a +2, independently expanded below.
			sum := sxPolyApp(t, sx, sy, "+", x, a)
			expr := sxPolyApp(t, sx, sy, "+",
				sxPolyApp(t, sx, sy, "^", sum, three),
				sxPolyApp(t, sx, sy, "-", sum),
				sxPolyApp(t, sx, sy, "*", sxPolyApp(t, sx, sy, "-", x, seven), a), two)
			budget := z.Budget()
			budget.FailAfter(fault)
			body := c.Convert(expr, out)
			// Audit the receiving owner while the refusal is still armed. The
			// global source uses a different directory and budget.
			upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{
				RationalManagers: []*unumeral.RationalManager{target.config.Rationals},
				BinaryManagers:   []*unumeral.BinaryRationalManager{target.config.Binary},
			})
			budget.FailAfter(-1)
			if body != nil {
				if !upolyHenselAllocationRefusal(z, body) {
					t.Fatal("unexpected receiving failure", body)
				}
				failures++
			} else {
				successes++
				done = true
			}
			serviceMust(t, c.Cleanup())
			if body != nil {
				// A failed scalar construction may leave an unusable source
				// field. Owning cleanup releases it without repairing/replaying
				// the source algorithm or promising recovery of that field.
				return
			}
			want := []string{
				"2037035976334486086268445688414198975184245364492562136913157598908946767597677601173274619",
				"4820814132776970826625886277032361361768206587158855428341763",
				"3802951800684688204490109616131", "1",
			}
			n, e := m.Size(out)
			serviceMust(t, e)
			if n != uint32(len(want)) {
				t.Fatal("degree", n)
			}
			for i, text := range want {
				var got, expected unumeral.Int
				serviceMust(t, m.ReadCoefficient(out, uint32(i), &got))
				serviceMust(t, z.SetString(&expected, text))
				equal, e := z.Equal(&got, &expected)
				serviceMust(t, e)
				serviceMust(t, z.Del(&got))
				serviceMust(t, z.Del(&expected))
				if !equal {
					t.Fatal("independent expanded coefficient", i)
				}
			}
		})
		if done {
			break
		}
	}
	if failures == 0 || successes != 1 {
		t.Fatal("did not reach receiving refusals and full completion", failures, successes)
	}
	t.Log("actual receiving allocation prefixes", failures, successes)
}

func TestSexprPolynomialArithmeticReceivingAllocationCallback(t *testing.T) {
	sx, sy, globals := sxOwner(t, arena.NewBudget(1<<24))
	var m *UPolynomialManager
	var c *SexprPolynomialConverter
	var out UPolynomial
	reached := false
	transport := func(f umemory.Failure) error {
		if c == nil || !c.arithmetic.executing {
			return f
		}
		reached = true
		for _, e := range []error{m.Close(), m.SourceFinalize(), m.DelPolynomial(&out), m.Reset(out)} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Error("arithmetic callback acquired polynomial owner", e)
			}
		}
		for _, e := range []error{c.Close(), c.Cleanup()} {
			if !errors.Is(e, ErrSexprPolynomialPending) {
				t.Error("arithmetic callback acquired converter", e)
			}
		}
		for _, z := range []*unumeral.IntegerManager{globals.Manager.Integers(), m.config.Integer} {
			for _, e := range []error{z.CloseChecked(), z.SourceFinalize()} {
				if !errors.Is(e, unumeral.ErrNumericPending) {
					t.Error("arithmetic callback acquired numeral owner", e)
				}
			}
		}
		if e := m.modular.Close(); !errors.Is(e, unumeral.ErrNumericPending) {
			t.Error("arithmetic callback acquired modular owner", e)
		}
		for _, e := range []error{globals.CloseChecked(), globals.SourceFinalize()} {
			if !errors.Is(e, arena.ErrInvalidSlot) {
				t.Error("arithmetic callback reentered global service mutex", e)
			}
		}
		if _, e := globals.Zero(); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Error("arithmetic callback reentered global value API", e)
		}
		return f
	}
	target, up, rt := algebraicNegTLSFixture(t, transport)
	m = up
	var e error
	out, e = m.NewPolynomial()
	serviceMust(t, e)
	c, e = NewSexprPolynomialConverter(sx, m)
	serviceMust(t, e)
	var pad umemory.Block
	armed := false
	m.config.Checkpoint = func() bool {
		if c.arithmetic.executing && !armed {
			pad = algebraicCellSourceTLSBoundary(t, target)
			armed = true
		}
		return true
	}
	a := sxPolyNum(t, sx, globals, "1267650600228229401496703205377")
	b := sxPolyNum(t, sx, globals, "633825300114114700748351602691")
	sourceBindings, targetBindings := globals.Manager.Integers().CallerBindings(), m.config.Integer.CallerBindings()
	body := c.Convert(sxPolyApp(t, sx, sy, "*", a, b), out)
	serviceMust(t, rt.SetMaxSize(0))
	contextTLSFinish(t, rt, func() error { return rt.Deallocate(&pad) })
	if body == nil || !reached || !c.arithmetic.pending() || len(c.frames) != 1 || !c.frames[0].live {
		t.Fatal("actual arithmetic allocation did not retain child and older argument", body)
	}
	if globals.Manager.Integers().CallerBindings() != sourceBindings || m.config.Integer.CallerBindings() != targetBindings {
		t.Fatal("arithmetic did not restore source and receiver bindings")
	}
	contextTLSFinish(t, rt, c.Cleanup)
	contextTLSFinish(t, rt, c.Close)
	contextTLSFinish(t, rt, func() error { return m.DelPolynomial(&out) })
	zero, e := globals.Zero()
	serviceMust(t, e)
	serviceMust(t, globals.Del(&zero))
}
