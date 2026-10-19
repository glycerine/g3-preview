// Source: src/test/polynomial.cpp:595-613; src/math/polynomial/polynomial.cpp:5954-6068,7284-7286.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// The registered test driver returns before tst_sqrt: dormant source assertions.
func TestPolynomialSqrtDormantOriginal(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{4, [3]uint32{1, 1, 0}}, {3, [3]uint32{2, 1, 0}}, {1, [3]uint32{0, 2, 0}}, {3, [3]uint32{}}})
	power, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, power.Close()) }()
	serviceMust(t, m.core.Power(p, 4, power))
	q, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, q.Close()) }()
	ok, e := m.Sqrt(power.root.value, q)
	serviceMust(t, e)
	if !ok {
		t.Fatal("dormant original VERIFY sqrt fourth power")
	}
	square, e := m.core.Mul(q.root.value, q.root.value)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(square))
	equal, e := m.core.Equal(power.root.value, square)
	serviceMust(t, e)
	serviceMust(t, m.core.DecRef(square))
	if !equal {
		t.Fatal("dormant original ENSURE q*q=p")
	}
	changed, e := m.core.Sub(power.root.value, m.core.unit)
	serviceMust(t, e)
	serviceMust(t, power.Assign(changed))
	ok, e = m.Sqrt(power.root.value, q)
	serviceMust(t, e)
	if ok {
		t.Fatal("dormant original ENSURE !sqrt(p-1)")
	}
}

func TestPolynomialSqrtIndependentSmallSquares(t *testing.T) {
	for _, terms := range [][]polynomialEvalTerm{
		{{2, [3]uint32{1, 0, 0}}, {3, [3]uint32{}}},
		{{1, [3]uint32{1, 0, 0}}, {2, [3]uint32{0, 1, 0}}, {1, [3]uint32{}}},
	} {
		m, p, _ := polynomialEvalFixture(t, terms)
		square, e := m.core.Mul(p, p)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(square))
		defer func() { serviceMust(t, m.core.DecRef(square)) }()
		r, e := NewPolynomialRef(m.core)
		serviceMust(t, e)
		defer func() { serviceMust(t, r.Close()) }()
		ok, e := m.Sqrt(square, r)
		serviceMust(t, e)
		if !ok {
			t.Fatal("independent known square rejected")
		}
		for x := int64(-2); x <= 2; x++ {
			for y := int64(-2); y <= 2; y++ {
				v := polynomialCompositionFiniteValue(t, m, r.root.value, [3]int64{x, y, 0})
				want := polynomialCompositionFiniteValue(t, m, p, [3]int64{x, y, 0})
				if v*v != want*want {
					t.Fatal("independent finite square value", x, y, v, want)
				}
			}
		}
	}
}

func TestPolynomialSqrtEveryReservation(t *testing.T) {
	for route := 0; route < 2; route++ {
		t.Run(fmt.Sprint(route), func(t *testing.T) {
			success := false
			for ordinal := int64(0); ordinal < 256; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{2, [3]uint32{1, 0, 0}}, {3, [3]uint32{}}})
					square, e := m.core.Mul(p, p)
					serviceMust(t, e)
					serviceMust(t, m.core.IncRef(square))
					defer func() { serviceMust(t, m.core.DecRef(square)) }()
					r, e := NewPolynomialRef(m.core)
					serviceMust(t, e)
					defer func() {
						m.core.integer.Budget().FailAfter(-1)
						serviceMust(t, m.CleanupSqrt())
						serviceMust(t, r.Close())
					}()
					if route == 1 {
						serviceMust(t, r.Assign(square))
					}
					m.core.integer.Budget().FailAfter(ordinal)
					ok, cause := m.Sqrt(square, r)
					polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, square, r.root.value, r.root.incoming}, nil, r.header)
					if cause == nil {
						if !ok {
							t.Fatal("known square rejected")
						}
						for x := int64(-3); x <= 3; x++ {
							a := polynomialCompositionFiniteValue(t, m, r.root.value, [3]int64{x})
							want := 2*x + 3
							if a*a != want*want {
								t.Fatal("independent root square", a, want)
							}
						}
						success = true
					} else if !numericPhysicalReservationRefusal(cause) {
						t.Fatal("actual reservation transport", cause)
					}
					m.core.integer.Budget().FailAfter(-1)
					serviceMust(t, m.CleanupSqrt())
					if m.core.pending.raw != (umemory.ObjectRef{}) || m.core.integer.PendingAllocationObject() != (umemory.ObjectRef{}) || m.core.monomials.creation.stage != 0 {
						t.Fatal("own operation abort omitted actual factory/numeric continuation")
					}
					polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, square, r.root.value, r.root.incoming}, nil, r.header)
				})
				if success {
					t.Log("actual success horizon", ordinal)
					break
				}
			}
			if !success {
				t.Fatal("all actual reservation prefixes never reached success")
			}
		})
	}
}
func TestPolynomialSqrtQuickBranchesAndCancel(t *testing.T) {
	for _, terms := range [][]polynomialEvalTerm{{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{2, 0, 0}}}, {{2, [3]uint32{2, 0, 0}}, {1, [3]uint32{}}}} {
		m, p, control := polynomialEvalFixture(t, terms)
		r, e := NewPolynomialRef(m.core)
		serviceMust(t, e)
		defer func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, r.Close()) }()
		ok, e := m.Sqrt(p, r)
		serviceMust(t, e)
		if ok || control.checkpoints != 0 || r.root.value != (Polynomial{}) {
			t.Fatal("actual min/max quick exit", ok, control, r.root.value)
		}
	}
	m, p, control := polynomialEvalFixture(t, []polynomialEvalTerm{{2, [3]uint32{1, 0, 0}}, {3, [3]uint32{}}})
	square, e := m.core.Mul(p, p)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(square))
	defer func() { serviceMust(t, m.core.DecRef(square)) }()
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, r.Close()) }()
	control.limit = control.checkpoints
	ok, e := m.Sqrt(square, r)
	if ok || !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal("real checkpoint cancel", ok, e)
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, square, r.root.value}, nil, r.header)
	serviceMust(t, m.CleanupSqrt())
	control.limit = 100000
	ok, e = m.Sqrt(square, r)
	serviceMust(t, e)
	if !ok {
		t.Fatal("explicit cleanup then fresh call")
	}
}

func TestPolynomialSqrtBigNumeralEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
			c := m.core
			a := unumeral.Int{}
			serviceMust(t, c.integer.SetLimbs(&a, []uint32{3, 256}))
			unit, e := c.monomials.Unit()
			serviceMust(t, e)
			term, e := c.monomialCore(p, 0)
			serviceMust(t, e)
			base, e := c.MkTerms([]unumeral.Int{a, unumeral.NewInt(3)}, []Monomial{term, unit})
			serviceMust(t, e)
			serviceMust(t, c.integer.Del(&a))
			serviceMust(t, c.IncRef(base))
			defer func() { serviceMust(t, c.DecRef(base)) }()
			square, e := c.Mul(base, base)
			serviceMust(t, e)
			serviceMust(t, c.IncRef(square))
			defer func() { serviceMust(t, c.DecRef(square)) }()
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			defer func() { c.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupSqrt()); serviceMust(t, r.Close()) }()
			c.integer.Budget().FailAfter(ordinal)
			ok, cause := m.Sqrt(square, r)
			polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, base, square, r.root.value, r.root.incoming}, nil, r.header)
			if cause == nil {
				if !ok {
					t.Fatal("actual big-coefficient square rejected")
				}
				success = true
			} else if !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual big numeric refusal", cause)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupSqrt())
			if c.integer.PendingAllocationObject() != (umemory.ObjectRef{}) || c.pending.raw != (umemory.ObjectRef{}) || c.pending.value != (Polynomial{}) {
				t.Fatal("actual operation-owned pending roots not consumed")
			}
			if cause == nil {
				product, e := c.Mul(r.root.value, r.root.value)
				serviceMust(t, e)
				serviceMust(t, c.IncRef(product))
				equal, e := c.Equal(product, square)
				serviceMust(t, e)
				serviceMust(t, c.DecRef(product))
				if !equal {
					t.Fatal("big exact root squared identity")
				}
			}
		})
		if success {
			t.Log("big actual success horizon", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("bounded big source operation never completed")
	}
}

func polynomialSqrtRefusalHandler(h *PolynomialDeletionHandler, c *MultivariatePolynomialCore, p Polynomial) error {
	if h.State == 1 {
		return c.integer.Budget().Reserve(1 << 62)
	}
	return nil
}
func TestPolynomialSqrtResultTransferredBeforeOldCallbackRefusal(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{2, [3]uint32{1, 0, 0}}, {3, [3]uint32{}}})
	c := m.core
	square, e := c.Mul(p, p)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(square))
	defer func() { serviceMust(t, c.DecRef(square)) }()
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, r.Close()) }()
	old, e := c.Add(p, c.unit)
	serviceMust(t, e)
	serviceMust(t, r.Assign(old))
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), polynomialSqrtRefusalHandler)
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	h.State = 1
	defer func() { h.State = 0; serviceMust(t, c.RemoveDeletionHandler(h)); serviceMust(t, h.CloseChecked()) }()
	ok, cause := m.Sqrt(square, r)
	if ok || cause == nil || r.root.incoming == (Polynomial{}) || !r.root.assigning {
		t.Fatal("actual reached old callback refusal after caller INC", ok, cause, r.root)
	}
	incoming := r.root.incoming
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, square, r.root.value, incoming}, nil, r.header+h.header)
	serviceMust(t, m.CleanupSqrt())
	if _, e = c.Info(incoming); e != nil {
		t.Fatal("operation abort destroyed caller-owned incoming result", e)
	}
	if r.root.incoming != incoming {
		t.Fatal("operation abort changed caller ownership")
	}
	h.State = 0
}

func TestPolynomialSqrtCreatedOnlyFalseQuotientCohort(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{4, 0, 0}}, {3, [3]uint32{3, 0, 0}}, {1, [3]uint32{}}})
			c := m.core
			var borrowed Monomial
			var e error
			if existing {
				borrowed, e = c.monomials.MkVariable(0, 1)
				serviceMust(t, e)
				defer func() { serviceMust(t, c.monomials.CleanupValue(&borrowed)) }()
			}
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			defer func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, r.Close()) }()
			ok, e := m.Sqrt(p, r)
			serviceMust(t, e)
			if ok {
				t.Fatal("two does not divide three")
			}
			live := uint32(0)
			for _, a := range m.squareRoot.orphans {
				if a != (Monomial{}) {
					live++
				}
			}
			if (!existing && live != 1) || (existing && live != 0) {
				t.Fatal("authority is actual new outcome, not RC0", existing, live)
			}
			polynomialSqrtPhysicalAudit(t, m, []Polynomial{p}, nil, r.header, borrowed)
			serviceMust(t, m.CleanupSqrt())
			if existing {
				if _, e = c.monomials.Info(borrowed); e != nil {
					t.Fatal("own abort consumed borrowed RC0 caller", e)
				}
			}
		})
	}
}
func TestPolynomialSqrtLeavesOlderPerfectSquareInvocation(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	cfg := serviceConfig(b).Integers
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, q.CloseChecked())
		if b.Used() != 0 {
			t.Fatal("actual whole graph refund", b.Used())
		}
	})
	m, p, _ := polynomialEvalFixtureWithRationals(t, q, []polynomialEvalTerm{{4, [3]uint32{2, 0, 0}}, {1, [3]uint32{}}})
	z := q.Integers()
	oldOut := unumeral.Int{}
	defer func() { b.FailAfter(-1); serviceMust(t, z.CleanupPerfectSquare()); serviceMust(t, z.Del(&oldOut)) }()
	b.FailAfter(0)
	_, cause := z.IsPerfectSquare(&z.Two64, &oldOut)
	if !numericPhysicalReservationRefusal(cause) || !z.PerfectSquare.Active {
		t.Fatal("real older raw copy failure", cause)
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	b.FailAfter(-1)
	generation := z.PerfectSquareGeneration
	values := z.PerfectSquare.Values
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, m.CleanupSqrt()); serviceMust(t, r.Close()) }()
	_, e = m.Sqrt(p, r)
	if e != unumeral.ErrPerfectSquarePending {
		t.Fatal("real reached newer raw acquisition guard", e)
	}
	if m.squareRoot.perfectSquare.Invocation != 0 {
		t.Fatal("new failed acquisition claimed unrelated raw owner")
	}
	serviceMust(t, m.CleanupSqrt())
	if z.PerfectSquareGeneration != generation || z.PerfectSquare.Values != values || !z.PerfectSquare.Active || z.PerfectSquare.BodyCause != cause {
		t.Fatal("new operation cleanup swept unrelated old raw invocation")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p}, nil, r.header)
	serviceMust(t, z.CleanupPerfectSquare())
}

func TestPolynomialSqrtPostphysicalIntegerRawReceipt(t *testing.T) {
	for _, mode := range []string{"own", "foreign", "postfree"} {
		t.Run(mode, func(t *testing.T) {
			b := arena.NewBudget(1 << 27)
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
			serviceMust(t, e)
			m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("raw Copy refuses before checkpoint") }, Cooperation: func() error { panic("raw Copy refuses before cooperation") }}})
			serviceMust(t, e)
			c := m.core
			_, e = c.monomials.MkVar()
			serviceMust(t, e)
			term, e := c.monomials.MkVariable(0, 2)
			serviceMust(t, e)
			digits := make([]uint32, 40000)
			digits[39999] = 1
			a := unumeral.Int{}
			serviceMust(t, q.Integers().SetLimbs(&a, digits))
			p, e := c.MkTerms([]unumeral.Int{a}, []Monomial{term})
			serviceMust(t, e)
			serviceMust(t, c.IncRef(p))
			serviceMust(t, q.Integers().Del(&a))
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			defer func() {
				b.FailAfter(-1)
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, m.CleanupSqrt)
				contextTLSFinish(t, rt, r.Close)
				contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
				contextTLSFinish(t, rt, m.CloseChecked)
				contextTLSFinish(t, rt, q.CloseChecked)
				contextTLSFinish(t, rt, rt.Close)
				if b.Used() != 0 {
					t.Fatal("whole actual graph refund", b.Used())
				}
			}()
			serviceMust(t, rt.SetMaxSize(1))
			_, cause := m.Sqrt(p, r)
			roots, e := q.Integers().Allocator.PendingRoots()
			serviceMust(t, e)
			if !numericPhysicalReservationRefusal(cause) || roots[0] == (umemory.Block{}) || q.Integers().PerfectSquare.Numeric.PendingAllocatorBlocks() != roots || m.squareRoot.perfectSquare.Invocation != q.Integers().PerfectSquareGeneration {
				t.Fatal("actual reached hi Copy physical raw acquisition", cause, roots, m.squareRoot.perfectSquare)
			}
			polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, r.root.value, r.root.incoming}, nil, r.header)
			if mode == "foreign" {
				serviceMust(t, q.Integers().Allocator.CleanupPending())
				later := unumeral.Int{}
				laterCause := q.Integers().SetLimbs(&later, digits)
				foreign, e := q.Integers().Allocator.PendingRoots()
				serviceMust(t, e)
				if !numericPhysicalReservationRefusal(laterCause) || foreign == roots || foreign[0] == (umemory.Block{}) {
					t.Fatal("real later raw generation", laterCause, foreign)
				}
				polynomialSqrtPhysicalAudit(t, m, []Polynomial{p}, []*unumeral.Int{&later}, r.header)
				state := rt.State()
				serviceMust(t, m.CleanupSqrt())
				after, e := q.Integers().Allocator.PendingRoots()
				serviceMust(t, e)
				if after != foreign || rt.State() != state {
					t.Fatal("own abort swept unrelated raw replacement")
				}
				serviceMust(t, rt.SetMaxSize(0))
				serviceMust(t, q.Integers().Allocator.CleanupPending())
				serviceMust(t, q.Integers().Del(&later))
			} else {
				serviceMust(t, rt.SetMaxSize(0))
				if mode == "postfree" {
					contextTLSNearMinimum(t, rt)
					first := m.CleanupSqrt()
					if !errors.Is(first, arena.ErrBounds) {
						t.Fatal("actual guarded postphysical release", first)
					}
					polynomialSqrtPhysicalAudit(t, m, []Polynomial{p}, nil, r.header)
					state := rt.State()
					serviceMust(t, m.CleanupSqrt())
					if rt.State() != state || m.squareRoot.bodyCause != cause {
						t.Fatal("owning retry repeated consumed free or replaced source body cause")
					}
				} else {
					serviceMust(t, m.CleanupSqrt())
				}
				after, e := q.Integers().Allocator.PendingRoots()
				serviceMust(t, e)
				if after != ([3]umemory.Block{}) {
					t.Fatal("actual raw root omitted from operation abort")
				}
			}
		})
	}
}

// This exercises the real reached R.mk allocator primitive and the same lexical
// owner constructor. It is not a claim that a mathematical sqrt reaches this
// large final result before any earlier TLS synchronization.
func TestPolynomialSqrtFinalFactoryPostphysicalRaw(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("factory primitive no checkpoint") }, Cooperation: func() error { panic("factory primitive no cooperation") }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var held umemory.Block
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupSqrt())
		serviceMust(t, c.monomials.allocator.CleanupPending())
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Deallocate(&held))
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("whole actual factory graph refund", b.Used())
		}
	}()
	a := unumeral.NewInt(1)
	for i := uint32(1); i <= 4000; i++ {
		term, e := c.monomials.MkVariable(0, i)
		serviceMust(t, e)
		serviceMust(t, c.som.add(&a, term, false))
	}
	// Prime the real TLS lane with a named caller allocation; native monomial
	// storage no longer contributes the old pointer-vector allocation traffic.
	pad := int64(99996) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("actual TLS priming extent", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	if rt.State().LocalSize != 99996 {
		t.Fatal("real TLS lane not primed", rt.State())
	}
	serviceMust(t, m.beginSquareRoot())
	serviceMust(t, rt.SetMaxSize(1))
	_, cause := c.som.mkWithFactoryReceipt(false, &m.squareRoot.factory)
	roots, e := c.monomials.allocator.PendingRoots()
	serviceMust(t, e)
	if !numericPhysicalReservationRefusal(cause) || roots[0] == (umemory.Block{}) || c.pending.generation != m.squareRoot.factory.invocation {
		t.Fatal("actual final cell source allocator postphysical stop", cause, roots, c.pending)
	}
	_ = m.finishSquareRoot(cause)
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []umemory.Block{held})
	polynomialOperationMonomialAudit(t, c, nil, nil)
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.CleanupSqrt())
	after, e := c.monomials.allocator.PendingRoots()
	serviceMust(t, e)
	if after != ([3]umemory.Block{}) {
		t.Fatal("matching final factory raw root omitted from operation abort", after)
	}
}

func polynomialSqrtPhysicalAudit(t *testing.T, m *MultivariatePolynomialManager, polys []Polynomial, input []*unumeral.Int, extra uint64, borrowed ...Monomial) {
	t.Helper()
	polynomialImpPhysicalAuditNamed(t, m, polys, input, extra)
	polynomialOperationMonomialAudit(t, m.core, polys, borrowed)
}

func TestPolynomialSqrtEveryCheckpointFrontier(t *testing.T) {
	success := false
	for frontier := uint32(0); frontier < 20; frontier++ {
		t.Run(fmt.Sprint(frontier), func(t *testing.T) {
			m, p, control := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {2, [3]uint32{0, 1, 0}}, {3, [3]uint32{}}})
			square, e := m.core.Mul(p, p)
			serviceMust(t, e)
			serviceMust(t, m.core.IncRef(square))
			defer func() { serviceMust(t, m.core.DecRef(square)) }()
			r, e := NewPolynomialRef(m.core)
			serviceMust(t, e)
			defer func() { control.limit = 100000; serviceMust(t, m.CleanupSqrt()); serviceMust(t, r.Close()) }()
			before := control.checkpoints
			control.limit = before + frontier
			ok, cause := m.Sqrt(square, r)
			polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, square, r.root.value, r.root.incoming}, nil, r.header)
			if cause == nil {
				if !ok {
					t.Fatal("independent square rejected")
				}
				success = true
			} else if !errors.Is(cause, ErrUPolynomialCanceled) {
				t.Fatal("actual reached checkpoint refusal", cause)
			}
			if cause != nil && control.checkpoints-control.cooperation != 1 {
				t.Fatal("source denial before cooperation", control)
			}
			serviceMust(t, m.CleanupSqrt())
		})
		if success {
			t.Log("actual checkpoint success frontier", frontier)
			break
		}
	}
	if !success {
		t.Fatal("complete checkpoint campaign never succeeded")
	}
}

// A real source lexical-owner destructor proof using actual completed scoped
// NUM fields. The large auxiliary capacity is an owner-primitive profile, not
// a claimed naturally reached polynomial square-root result.
func TestPolynomialSqrtSourceAuxStopKeepsOlderFields(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("source local destructor no checkpoint") }, Cooperation: func() error { panic("source local destructor no cooperation") }}})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, m.CleanupSqrt)
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("whole named owner graph", b.Used())
		}
	}()
	// Real ordinary create/INC/DEC warms ID free-vector capacity before the
	// guarded signed-domain helper. This isolates the lexical leaf release;
	// it is not claimed mathematical square-root failure reachability.
	var warm [4]Polynomial
	for i := range warm {
		v := unumeral.NewInt(int32(7 + i))
		warm[i], e = m.core.MkConstantTake(&v)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(warm[i]))
	}
	for i := range warm {
		serviceMust(t, m.core.DecRef(warm[i]))
	}
	x, e := m.core.monomials.MkVar()
	serviceMust(t, e)
	term, e := m.core.monomials.MkVariable(x, 3)
	serviceMust(t, e)
	serviceMust(t, m.core.monomials.IncRef(term))
	serviceMust(t, m.core.monomials.DecRef(term))
	serviceMust(t, m.beginSquareRoot())
	s := &m.squareRoot
	// Actual source scoped constructors default the fields; no malloc event.
	s.constructNumeral(1)
	s.constructNumeral(2)
	s.constructNumeral(3)
	s.referenceCompleted = true
	serviceMust(t, q.Integers().SetInt32(&s.values[0], 13))
	serviceMust(t, q.Integers().Copy(&s.values[1], &q.Integers().Two64))
	serviceMust(t, q.Integers().Copy(&s.values[2], &q.Integers().Two64))
	serviceMust(t, q.Integers().Copy(&s.values[3], &q.Integers().Two64))
	serviceMust(t, q.Integers().EnsureOwnedCapacity(&s.values[3], 40000))
	older := [3]unumeral.Int{s.values[0], s.values[1], s.values[2]}
	contextTLSNearMinimum(t, rt)
	first := m.finishSquareRoot(nil)
	if !errors.Is(first, arena.ErrBounds) || !s.stopped || s.stage != 0 || s.sourceCause == nil || s.values[0] != older[0] || s.values[1] != older[1] || s.values[2] != older[2] {
		t.Fatal("actual younger auxiliary release stopped before older source dtors", first, s.stage)
	}
	polynomialSqrtPhysicalAudit(t, m, nil, nil, 0)
	state := rt.State()
	again := m.finishSquareRoot(nil)
	if !errors.Is(again, arena.ErrBounds) || s.sourceCause == nil || rt.State() != state || s.stage != 0 {
		t.Fatal("source retry replayed completed physical free", again)
	}
	saved := s.sourceCause
	contextTLSFinish(t, rt, m.CleanupSqrt)
	if s.sourceCause != saved || s.stage != 0 || !s.stopped {
		t.Fatal("own abort erased source frontier/history")
	}
	polynomialSqrtPhysicalAudit(t, m, nil, nil, 0)
}

func TestPolynomialSqrtDeletionCallbackForeignRootsStayCallerOwned(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual callback whole graph refund", b.Used())
		}
	})
	m, p, _ := polynomialEvalFixtureWithRationals(t, q, []polynomialEvalTerm{{2, [3]uint32{1, 0, 0}}, {3, [3]uint32{}}})
	c := m.core
	square, e := c.Mul(p, p)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(square))
	defer func() { serviceMust(t, c.DecRef(square)) }()
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	old, e := c.Add(p, c.unit)
	serviceMust(t, e)
	serviceMust(t, r.Assign(old))
	input, out := unumeral.Int{}, unumeral.Int{}
	digits := make([]uint32, 40000)
	digits[39999] = 1
	serviceMust(t, q.Integers().SetLimbs(&input, digits))
	var foreign Monomial
	invoked := false
	h, e := NewPolynomialDeletionHandler(b, func(_ *PolynomialDeletionHandler, called *MultivariatePolynomialCore, value Polynomial) error {
		// Compare the full checked handle, including generation, not public ID.
		if value != old {
			return nil
		}
		if invoked {
			t.Fatal("actual stopped callback replayed")
		}
		invoked = true
		var cause error
		foreign, cause = called.monomials.MkVariable(2, 7)
		if cause != nil {
			return cause
		}
		if cause = rt.SetMaxSize(1); cause != nil {
			return cause
		}
		return called.integer.Copy(&out, &input)
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupSqrt())
		serviceMust(t, q.Integers().Allocator.CleanupPending())
		serviceMust(t, q.Integers().Del(&input))
		serviceMust(t, q.Integers().Del(&out))
		serviceMust(t, c.monomials.CleanupValue(&foreign))
		serviceMust(t, r.Close())
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
	}()
	_, cause := m.Sqrt(square, r)
	roots, e := q.Integers().Allocator.PendingRoots()
	serviceMust(t, e)
	if !numericPhysicalReservationRefusal(cause) || !invoked || foreign == (Monomial{}) || roots[0] == (umemory.Block{}) || r.root.incoming == (Polynomial{}) {
		t.Fatal("actual foreign callback work after caller incoming publication", cause, foreign, roots)
	}
	if polynomialNegNumericPending(&m.squareRoot.numeric) {
		t.Fatal("callback work acquired invocation authority")
	}
	polynomialSqrtPhysicalAudit(t, m, []Polynomial{p, square, r.root.value, r.root.incoming}, []*unumeral.Int{&input, &out}, r.header+h.header, foreign)
	state := rt.State()
	serviceMust(t, m.CleanupSqrt())
	after, e := q.Integers().Allocator.PendingRoots()
	serviceMust(t, e)
	if after != roots || rt.State() != state {
		t.Fatal("operation abort swept foreign callback raw root")
	}
	if _, e = c.monomials.Info(foreign); e != nil {
		t.Fatal("operation abort swept actual foreign RC0 result", e)
	}
	serviceMust(t, rt.SetMaxSize(0))
}
