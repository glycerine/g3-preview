package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usexpr "github.com/glycerine/g3/pkg/util/sexpr"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSexprPolynomialRefusedLexicalEntryPreservesIndependentScope(t *testing.T) {
	t.Run("never-entered", func(t *testing.T) { sexprPolynomialRefusedLexicalEntry(t, false) })
	t.Run("older-completed-entry", func(t *testing.T) { sexprPolynomialRefusedLexicalEntry(t, true) })
}

func sexprPolynomialRefusedLexicalEntry(t *testing.T, prior bool) {
	globals, m, out, rt, budget := contextTLSOwner(t)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: budget, PageSize: 64, DescriptorPageSize: 2})
	serviceMust(t, e)
	sx, e := usexpr.NewSexprManager(usexpr.SexprConfig{Budget: budget, PageSize: 2, Symbols: sy, Rationals: globals})
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, sx.Close); serviceMust(t, sy.Close()) })
	c, e := NewSexprPolynomialConverter(sx, m)
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, c.Close) })
	one := sxPolyNum(t, sx, globals, "1")
	if prior {
		serviceMust(t, c.Convert(one, out))
	}
	a := contextBigOdd(t, globals, 2)
	t.Cleanup(func() { contextTLSDelete(t, globals, &a, rt) })
	original := errors.New("independent noninteger input")
	body := numericRationalService_WithUPolynomialRationals(globals, m, func(ctx *UPolynomialRationalContext) error {
		return ctx.SetConstant(out, &a, func() error {
			contextTLSNearMinimum(t, rt)
			return original
		})
	})
	before := globals.Lexical.State()
	if !errors.Is(body, original) || !errors.Is(body, arena.ErrBounds) || before.Roots != 1 || before.Completed != 1 || !before.Pending {
		t.Fatal("independent actual destructor stop was not reached", body, before)
	}
	if e := c.Convert(one, out); !errors.Is(e, unumeral.ErrRationalLexicalPending) {
		t.Fatal("converter entered an independent pending lexical scope", e)
	}
	serviceMust(t, c.Cleanup())
	if after := globals.Lexical.State(); after != before {
		t.Error("refused converter cleanup acquired independent lexical roots", before, after)
	}
	serviceMust(t, c.Close())
	if after := globals.Lexical.State(); after != before {
		t.Error("refused converter close acquired independent lexical roots", before, after)
	}
	serviceMust(t, numericRationalService_CleanupUPolynomialRationals(globals, m, func() error { return nil }))
	contextTLSFinish(t, rt, func() error { return m.DelPolynomial(&out) })
}

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

func TestSexprPolynomialForeignChildRetainsOlderArgumentVector(t *testing.T) {
	sx, sy, globals := sxOwner(t, arena.NewBudget(1<<24))
	target := algebraicCopyBoundedTarget(t)
	m := target.config.Polynomials
	out, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&out)) })
	c, e := NewSexprPolynomialConverter(sx, m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	one := sxPolyNum(t, sx, globals, "1")
	big := sxPolyNum(t, sx, globals, "1267650600228229401496703205377")
	expr := sxPolyApp(t, sx, sy, "+", one, big)
	body := c.Convert(expr, out)
	if !errors.Is(body, arena.ErrSlotLimit) || !m.rationalInput.pending(m) {
		t.Fatal("foreign child did not retain an actual receiving allocation", body)
	}
	if len(c.frames) != 1 || !c.frames[0].live || c.cause == nil || c.depth != 0 {
		t.Fatal("older argument vector was destroyed before receiving child acknowledgement")
	}
	d, e := m.descriptor(out)
	serviceMust(t, e)
	if d.state.size != 1 || m.values(&d.state)[0].NativeScalar() != 1 {
		t.Fatal("failed later argument changed the completed first-argument prefix")
	}
	zero, e := globals.Zero()
	serviceMust(t, e)
	serviceMust(t, globals.Del(&zero))
	serviceMust(t, c.Cleanup())
	if c.frames[0].live || c.cause != nil || m.rationalInput.pending(m) {
		t.Fatal("converter cleanup did not acknowledge child before older vector")
	}
	serviceMust(t, c.Convert(sxPolyApp(t, sx, sy, "+", one, one), out))
	sxPolyCoefficients(t, m, out, []int32{2})
}

func TestSexprPolynomialRefusedInputPreservesIndependentReceivingCopy(t *testing.T) {
	sx, _, globals := sxOwner(t, arena.NewBudget(1<<24))
	target := algebraicCopyBoundedTarget(t)
	m, z := target.config.Polynomials, target.config.Rationals.Integers()
	older, e := m.NewPolynomial()
	serviceMust(t, e)
	refused, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CleanupRationalInput())
		serviceMust(t, m.DelPolynomial(&refused))
		serviceMust(t, m.DelPolynomial(&older))
	})
	c, e := NewSexprPolynomialConverter(sx, m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	big := sxNumber(t, globals, "1267650600228229401496703205377")
	t.Cleanup(func() { serviceMust(t, globals.Del(&big)) })
	one := sxPolyNum(t, sx, globals, "1")
	// Retain an older completed converter receipt as well: same frame alone
	// must not grant authority over the following independent invocation.
	serviceMust(t, c.Convert(one, refused))
	body := m.setRationalsFromOwner(older, globals.Manager.Integers(), []unumeral.Rational{big.Raw})
	if !errors.Is(body, arena.ErrSlotLimit) || !m.rationalInput.pending(m) {
		t.Fatal("independent receiving copy did not acquire physical storage", body)
	}
	issued, physical := m.rationalInput.copy, z.PendingCell.Object
	if e := c.Convert(one, refused); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("converter replaced independent pending input", e)
	}
	serviceMust(t, c.Cleanup())
	if !z.MatchesCopyAllocationInvocation(&issued) || z.PendingCell.Object != physical {
		t.Error("refused converter cleanup acquired independent receiving copy")
	}
	serviceMust(t, m.CleanupRationalInput())
}

func TestAlgebraicPrivateSexprFailureRetainsReceivingChildAndParser(t *testing.T) {
	sx, sy, globals := sxOwner(t, arena.NewBudget(1<<24))
	target := algebraicCopyBoundedTarget(t)
	x := sxPolySym(t, sx, sy, "x")
	two := sxPolyNum(t, sx, globals, "2")
	big := sxPolyNum(t, sx, globals, "1267650600228229401496703205377")
	power := sxPolyApp(t, sx, sy, "^", x, two)
	expr := sxPolyApp(t, sx, sy, "+", power, big)
	var out Algebraic
	t.Cleanup(func() {
		serviceMust(t, target.CleanupRetained())
		serviceMust(t, target.CleanupValue(&out))
	})
	body := target.MkSexprRoot(sx, expr, 2, &out)
	s := &target.rootConstruction
	if !errors.Is(body, arena.ErrSlotLimit) || out != (Algebraic{}) || !s.active || s.parser == nil {
		t.Fatal("private root parser did not retain failed receiving child", body)
	}
	parser := s.parser
	if !target.config.Polynomials.rationalInputPending(&parser.inputReceipt) || len(parser.frames) != 1 || !parser.frames[0].live {
		t.Fatal("root parser lost exact receiving child or older argument vector")
	}
	serviceMust(t, target.CleanupRetained())
	if s.active || s.parser != nil || !parser.closed || len(parser.frames) != 0 || target.config.Polynomials.rationalInput.pending(target.config.Polynomials) {
		t.Fatal("root owning cleanup did not release its actual child/parser sequence")
	}
	sxPolyCoefficients(t, target.config.Polynomials, target.isolateTmp1, []int32{0, 0, 1})
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
