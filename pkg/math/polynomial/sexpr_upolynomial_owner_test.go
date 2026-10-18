package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usexpr "github.com/glycerine/g3/pkg/util/sexpr"
)

func TestSexprPolynomialForeignCoefficientAllocationCallback(t *testing.T) {
	sx, _, globals := sxOwner(t, arena.NewBudget(1<<24))
	n := sxPolyNum(t, sx, globals, "1267650600228229401496703205377")
	var m *UPolynomialManager
	var c *SexprPolynomialConverter
	var out UPolynomial
	reached := false
	transport := func(f umemory.Failure) error {
		if m == nil || !m.rationalInput.running {
			return f
		}
		reached = true
		for _, e := range []error{m.CleanupRationalInput(), m.Close(), m.SourceFinalize(), m.DelPolynomial(&out)} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Error("receiving callback acquired polynomial lifetime", e)
			}
		}
		for _, e := range []error{c.Cleanup(), c.Close()} {
			if !errors.Is(e, ErrSexprPolynomialPending) {
				t.Error("receiving callback acquired converter lifetime", e)
			}
		}
		for _, e := range []error{globals.CloseChecked(), globals.SourceFinalize()} {
			if !errors.Is(e, arena.ErrInvalidSlot) {
				t.Error("receiving callback entered the borrowed source service mutex", e)
			}
		}
		if _, e := globals.Zero(); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Error("receiving callback reentered ordinary source service API", e)
		}
		for _, z := range []*unumeral.IntegerManager{globals.Manager.Integers(), m.config.Integer} {
			guarded := false
			for b := z.CallerBindings(); b != nil; b = b.Previous() {
				if b.Guard().NumericExecuting() {
					guarded = true
				}
			}
			if !guarded {
				t.Error("foreign constant copy did not bind both numeral owners")
				continue
			}
			for _, e := range []error{z.CloseChecked(), z.SourceFinalize()} {
				if !errors.Is(e, unumeral.ErrNumericPending) {
					t.Error("receiving callback acquired numeral lifetime", e)
				}
			}
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
	pad := algebraicCellSourceTLSBoundary(t, target)
	sourceBindings, targetBindings := globals.Manager.Integers().CallerBindings(), m.config.Integer.CallerBindings()
	body := c.Convert(n, out)
	serviceMust(t, rt.SetMaxSize(0))
	contextTLSFinish(t, rt, func() error { return rt.Deallocate(&pad) })
	if body == nil || !reached || !m.config.Integer.MatchesCopyAllocationInvocation(&m.rationalInput.copy) {
		t.Fatal("receiving allocation callback/receipt not reached", body)
	}
	if globals.Manager.Integers().CallerBindings() != sourceBindings || m.config.Integer.CallerBindings() != targetBindings || m.rationalInput.source != globals.Manager.Integers() {
		t.Fatal("failed SET did not restore synchronous bindings and retain its actual source owner")
	}
	contextTLSFinish(t, rt, c.Cleanup)
	if m.rationalInput.source != nil || m.rationalInput.bound != 0 {
		t.Fatal("completed input cleanup retained borrowed numeral owners")
	}
	zero, e := globals.Zero()
	serviceMust(t, e)
	serviceMust(t, globals.Del(&zero))
	contextTLSFinish(t, rt, c.Close)
	contextTLSFinish(t, rt, func() error { return m.DelPolynomial(&out) })
}

func TestSexprPolynomialForeignCoefficientPhysicalRefusal(t *testing.T) {
	sx, _, globals := sxOwner(t, arena.NewBudget(1<<24))
	target := algebraicCopyBoundedTarget(t)
	m := target.config.Polynomials
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&p)) })
	c, e := NewSexprPolynomialConverter(sx, m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	n := sxPolyNum(t, sx, globals, "1267650600228229401496703205377")
	body := c.Convert(n, p)
	z := m.config.Integer
	if !errors.Is(body, arena.ErrSlotLimit) || !z.MatchesCopyAllocationInvocation(&m.rationalInput.copy) || z.PendingCell.Object == (umemory.ObjectRef{}) {
		t.Fatal("foreign constant SET lost its accepted physical receipt", body)
	}
	physical := z.PendingCell.Object
	if e := c.Convert(n, p); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("pending foreign coefficient copy was overwritten", e)
	}
	serviceMust(t, c.Cleanup())
	if m.rationalInput.pending(m) || z.PendingCell.Object != (umemory.ObjectRef{}) {
		t.Fatal("converter cleanup retained receiving allocation")
	}
	if _, e := z.Allocator.ObjectExtent(physical); e == nil {
		t.Fatal("converter cleanup left physical backing live")
	}
	serviceMust(t, globals.Manager.CheckOwnerReady())
}

func TestAlgebraicPrivateOwnerSexprRoot(t *testing.T) {
	cfg, context, budget := algebraicManagerFixture(t)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: serviceConfig(budget).Integers})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, q.CloseChecked()) })
	cfg.Numerals = q
	m, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	sx, e := usexpr.NewSexprManager(usexpr.SexprConfig{Budget: budget, PageSize: 2, Symbols: context.SymbolsField, Rationals: cfg.Rationals})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sx.Close()) })
	x := sxPolySym(t, sx, context.SymbolsField, "x")
	two := sxPolyNum(t, sx, cfg.Rationals, "2")
	a := sxPolyNum(t, sx, cfg.Rationals, "1267650600228229401496703205376")
	b := sxPolyNum(t, sx, cfg.Rationals, "2535301200456458802993406410752")
	power := sxPolyApp(t, sx, context.SymbolsField, "^", x, two)
	product := sxPolyApp(t, sx, context.SymbolsField, "*", a, power)
	expr := sxPolyApp(t, sx, context.SymbolsField, "-", product, b)
	var root, square Algebraic
	t.Cleanup(func() {
		serviceMust(t, m.CleanupRetained())
		serviceMust(t, m.CleanupValue(&square))
		serviceMust(t, m.CleanupValue(&root))
	})
	serviceMust(t, m.MkSexprRoot(sx, expr, 2, &root))
	algebraicExpectBetween(t, m.AlgebraicRootCore, &root, 1, 2)
	serviceMust(t, m.Power(&root, 2, &square))
	want := unumeral.NewRational()
	serviceMust(t, q.SetInt32(&want, 2))
	cmp, e := m.CompareRational(&square, &want)
	serviceMust(t, e)
	serviceMust(t, q.Del(&want))
	if cmp != 0 {
		t.Fatal("private-owner parsed root did not square to two", cmp)
	}
}
