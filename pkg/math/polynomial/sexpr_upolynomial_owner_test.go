package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usexpr "github.com/glycerine/g3/pkg/util/sexpr"
)

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
