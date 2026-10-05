// Source: src/math/polynomial/polynomial.cpp:6600-6679;
// src/test/polynomial.cpp:1017-1029,1053-1080.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/clibrand"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// This is the actual dormant input passed directly to the quadratic helper.
// It grants no registered-driver or complete public Factor credit.
func TestPolynomialQuadraticFactorDormantInputAndParity(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for k := uint32(1); k <= 4; k++ {
			t.Run(fmt.Sprint(negative, "/", k), func(t *testing.T) {
				terms := []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 0, 3}}, {2, [3]uint32{1, 0, 0}}, {-1, [3]uint32{1, 3, 0}}, {1, [3]uint32{0, 3, 3}}, {-2, [3]uint32{0, 3, 0}}}
				if negative {
					for i := range terms {
						terms[i].coefficient = -terms[i].coefficient
					}
				}
				m, p, _ := polynomialEvalFixture(t, terms)
				factors, e := NewPolynomialFactors(m)
				serviceMust(t, e)
				defer func() { serviceMust(t, factors.CloseChecked()) }()
				serviceMust(t, m.factorQuadraticSquareFreePrimitive(p, factors, 0, k))
				n, e := factors.Distinct()
				serviceMust(t, e)
				if n != 2 {
					t.Fatal("source quadratic helper count", n)
				}
				for i := uint32(0); i < n; i++ {
					d, e := factors.Degree(i)
					serviceMust(t, e)
					if d != k {
						t.Fatal("literal pushed multiplicity", d, k)
					}
				}
				actual, e := NewPolynomialRef(m.core)
				serviceMust(t, e)
				defer func() { serviceMust(t, actual.Close()) }()
				expected, e := NewPolynomialRef(m.core)
				serviceMust(t, e)
				defer func() { serviceMust(t, expected.Close()) }()
				serviceMust(t, factors.Multiply(actual))
				serviceMust(t, m.core.Power(p, k, expected))
				equal, e := m.core.Equal(actual.root.value, expected.root.value)
				serviceMust(t, e)
				if !equal {
					t.Fatal("independent source parity/product identity")
				}
			})
		}
	}
}

func TestPolynomialQuadraticIrreducibleNegativeReturnsBeforeParityFlip(t *testing.T) {
	for k := uint32(1); k <= 2; k++ {
		m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{-1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{0, 2, 0}}, {-1, [3]uint32{}}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		defer func() { serviceMust(t, f.CloseChecked()) }()
		serviceMust(t, m.factorQuadraticSquareFreePrimitive(p, f, 0, k))
		n, e := f.Distinct()
		serviceMust(t, e)
		if n != 1 {
			t.Fatal("source irreducible branch", n)
		}
		constant, e := f.Constant()
		serviceMust(t, e)
		one, e := m.core.modular.IsOne(constant)
		serviceMust(t, e)
		if !one {
			t.Fatal("sqrt-false must return before parity flip", *constant)
		}
		r, e := f.At(0)
		serviceMust(t, e)
		value, e := r.Get()
		serviceMust(t, e)
		if value != p {
			t.Fatal("source push original identity")
		}
		serviceMust(t, r.Close())
	}
}

func TestPolynomialQuadraticEveryActualReservation(t *testing.T) {
	for _, negative := range []bool{false, true} {
		success := false
		for ordinal := int64(0); ordinal < 2048; ordinal++ {
			t.Run(fmt.Sprint(negative, "/", ordinal), func(t *testing.T) {
				terms := []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 0, 3}}, {2, [3]uint32{1, 0, 0}}, {-1, [3]uint32{1, 3, 0}}, {1, [3]uint32{0, 3, 3}}, {-2, [3]uint32{0, 3, 0}}}
				if negative {
					for i := range terms {
						terms[i].coefficient = -terms[i].coefficient
					}
				}
				m, p, _ := polynomialEvalFixture(t, terms)
				c := m.core
				f, e := NewPolynomialFactors(m)
				serviceMust(t, e)
				defer func() {
					c.integer.budget.FailAfter(-1)
					serviceMust(t, m.CleanupGCD())
					serviceMust(t, f.CloseChecked())
				}()
				c.integer.budget.FailAfter(ordinal)
				cause := m.factorQuadraticSquareFreePrimitive(p, f, 0, 1)
				for i, phase := range f.phases {
					if phase != factorPublished {
						continue
					}
					borrowed, e := f.factor(uint32(i))
					serviceMust(t, e)
					if borrowed == (Polynomial{}) {
						t.Fatal("actual pointer publication missing")
					}
					// Source Push has not acquired INC for this row. Caller lexical
					// unwind may already have consumed the pointed-to polynomial.
					if _, e = c.polynomialLocation(borrowed); e != nil && e != ErrPolynomialOwner {
						t.Fatal(e)
					}
				}
				polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
				if cause != nil && !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual factor prefix category", cause)
				}
				c.integer.budget.FailAfter(-1)
				serviceMust(t, m.CleanupGCD())
				for i, phase := range f.phases {
					if phase != factorPublished {
						continue
					}
					borrowed, e := f.factor(uint32(i))
					serviceMust(t, e)
					if borrowed == (Polynomial{}) {
						t.Fatal("actual pointer publication missing")
					}
					// Source Push has not acquired INC for this row. Caller lexical
					// unwind may already have consumed the pointed-to polynomial.
					if _, e = c.polynomialLocation(borrowed); e != nil && e != ErrPolynomialOwner {
						t.Fatal(e)
					}
				}
				polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
				if cause == nil {
					success = true
					n, e := f.Distinct()
					serviceMust(t, e)
					if n != 2 {
						t.Fatal("completed source pushes", n)
					}
				}
			})
			if success {
				t.Log("quadratic actual success", negative, ordinal)
				break
			}
		}
		if !success {
			t.Fatal("bounded source helper never completed", negative)
		}
	}
}

// Source Push publishes its borrowed pointer before degree-vector growth and
// INC. A failed degree push followed by caller lexical unwind therefore leaves
// a native dangling pointer. The Go source destructor preserves the reached
// identity guard; owning abort skips precisely the reference never acquired.
func TestPolynomialQuadraticPublishedBorrowedFailureSourceGuard(t *testing.T) {
	t.Cleanup(func() { clibrand.Srand(1) })
	observed := false
	for ordinal := int64(0); ordinal < 2048 && !observed; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			terms := []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 0, 3}}, {2, [3]uint32{1, 0, 0}}, {-1, [3]uint32{1, 3, 0}}, {1, [3]uint32{0, 3, 3}}, {-2, [3]uint32{0, 3, 0}}}
			m, p, _ := polynomialEvalFixture(t, terms)
			c := m.core
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			defer func() {
				c.integer.budget.FailAfter(-1)
				serviceMust(t, m.CleanupGCD())
				serviceMust(t, f.CloseChecked())
			}()
			// Fix the actual process-wide source random input for each prefix.
			// Production GCD never reseeds per owner/invocation.
			clibrand.Srand(1)
			c.integer.budget.FailAfter(ordinal)
			cause := m.factorQuadraticSquareFreePrimitive(p, f, 0, 1)
			polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
			for i, phase := range f.phases {
				if phase != factorPublished {
					continue
				}
				borrowed, e := f.factor(uint32(i))
				serviceMust(t, e)
				if _, e = c.polynomialLocation(borrowed); e != ErrPolynomialOwner {
					continue
				}
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("source degree-push refusal", cause)
				}
				observed = true
				if e = f.SourceFinalize(); e != ErrPolynomialOwner {
					t.Fatal("reached source dangling-pointer guard", e)
				}
				stage, index, ptr := f.sourceStage, f.resetIndex, borrowed
				if e = f.SourceFinalize(); e != ErrPolynomialOwner || f.sourceStage != stage || f.resetIndex != index {
					t.Fatal("source retry changed stopped frontier", e)
				}
				again, e := f.factor(uint32(i))
				serviceMust(t, e)
				if again != ptr || f.phases[i] != factorPublished {
					t.Fatal("source guard repaired borrowed publication")
				}
				polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
			}
			c.integer.budget.FailAfter(-1)
			serviceMust(t, m.CleanupGCD())
			serviceMust(t, f.Cleanup())
			polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
		})
	}
	if !observed {
		t.Fatal("actual borrowed publication refusal not reached")
	}
}

func TestPolynomialQuadraticEveryActualCheckpoint(t *testing.T) {
	for _, negative := range []bool{false, true} {
		success := false
		for limit := uint32(0); limit < 2048 && !success; limit++ {
			t.Run(fmt.Sprint(negative, "/", limit), func(t *testing.T) {
				terms := []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 0, 3}}, {2, [3]uint32{1, 0, 0}}, {-1, [3]uint32{1, 3, 0}}, {1, [3]uint32{0, 3, 3}}, {-2, [3]uint32{0, 3, 0}}}
				if negative {
					for i := range terms {
						terms[i].coefficient = -terms[i].coefficient
					}
				}
				m, p, control := polynomialEvalFixture(t, terms)
				f, e := NewPolynomialFactors(m)
				serviceMust(t, e)
				defer func() { control.limit = ^uint32(0); serviceMust(t, m.CleanupGCD()); serviceMust(t, f.CloseChecked()) }()
				control.checkpoints, control.cooperation, control.limit = 0, 0, limit
				cause := m.factorQuadraticSquareFreePrimitive(p, f, 0, 1)
				polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
				if cause == nil {
					success = true
					if control.checkpoints != control.cooperation {
						t.Fatal("source checkpoint pairing", control)
					}
					n, e := f.Distinct()
					serviceMust(t, e)
					if n != 2 {
						t.Fatal("source completed pushes", n)
					}
				} else {
					if !errors.Is(cause, ErrUPolynomialCanceled) {
						t.Fatal("actual source cancellation", cause)
					}
					if control.checkpoints != limit+1 || control.cooperation != limit {
						t.Fatal("exact cancel frontier", control, limit)
					}
				}
				control.limit = ^uint32(0)
				serviceMust(t, m.CleanupGCD())
				polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
			})
			if success {
				t.Log("quadratic actual checkpoint success", negative, limit)
			}
		}
		if !success {
			t.Fatal("source checkpoint horizon", negative)
		}
	}
}

func TestPolynomialQuadraticForeignCallbackRootsSurviveOwnAbort(t *testing.T) {
	terms := []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 0, 3}}, {2, [3]uint32{1, 0, 0}}, {-1, [3]uint32{1, 3, 0}}, {1, [3]uint32{0, 3, 3}}, {-2, [3]uint32{0, 3, 0}}}
	m, p, _ := polynomialEvalFixture(t, terms)
	c := m.core
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	var foreign, deleted Polynomial
	foreignNumber := Int{}
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.budget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, current Polynomial) error {
		if deleted != (Polynomial{}) {
			if current == deleted {
				calls++
				t.Error("completed callback replay")
			}
			return nil
		}
		deleted = current
		calls++
		var x error
		foreign, x = c.MkVariable(2, 1)
		if x != nil {
			return x
		}
		if x = c.IncRef(foreign); x != nil {
			return x
		}
		if x = c.integer.SetLimbs(&foreignNumber, []uint32{0, 0, 1}); x != nil {
			return x
		}
		c.integer.budget.FailAfter(0)
		_, x = c.memory.Allocate(8)
		return x
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	defer func() {
		c.integer.budget.FailAfter(-1)
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		serviceMust(t, c.DecRef(foreign))
		serviceMust(t, c.integer.Del(&foreignNumber))
		serviceMust(t, f.CloseChecked())
	}()
	cause := m.factorQuadraticSquareFreePrimitive(p, f, 0, 1)
	polynomialFactorsAuditExtra(t, f, []Polynomial{p, foreign}, []*Int{&foreignNumber}, h.header)
	if !numericPhysicalReservationRefusal(cause) || calls != 1 || foreign == (Polynomial{}) {
		t.Fatal("actual callback foreign roots then refusal", cause, calls)
	}
	c.integer.budget.FailAfter(-1)
	serviceMust(t, m.CleanupGCD())
	polynomialFactorsAuditExtra(t, f, []Polynomial{p, foreign}, []*Int{&foreignNumber}, h.header)
	polynomialQueryExpect(t, c, foreign, map[[3]uint32]int64{{0, 0, 1}: 1})
	value := integerString(t, c.integer, &foreignNumber)
	if value != "18446744073709551616" || calls != 1 {
		t.Fatal("foreign root consumed or callback replay", value, calls)
	}
}

// Owner-primitive signed-host overflow proof, not an assertion that ordinary
// quadratic mathematics or a free-OOM callback can reach this source frontier.
func TestPolynomialQuadraticYoungerSqrtSourceStopPreservesOlderLocal(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { panic("helper checkpoint unreachable") }, Cooperation: func() error { panic("helper cooperation unreachable") }}})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, rt, m.CleanupGCD)
		contextTLSFinish(t, rt, func() error {
			e := m.CloseChecked()
			if e != nil {
				t.Logf("actual full close frontier: cleanup=%d coreSource=%d upClosed=%v mmRefs=%d mmSource=%d unitConsumed=%v unitCause=%v coreCause=%v fullCause=%v", m.cleanupStage, m.core.sourceStage, m.upm.closed, m.core.monomials.managerRefs, m.core.monomials.sourceStage, m.core.monomials.unitDropConsumed, m.core.monomials.unitDropCause, m.core.originalCause, m.cleanupCause)
			}
			return e
		})
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("actual whole graph", b.Used())
		}
	}()
	// Ordinary source constructors and DEC warm the actual source ID freelist.
	var warm [4]Polynomial
	for i := range warm {
		v := NewInt(int32(7 + i))
		warm[i], e = m.core.MkConstantTake(&v)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(warm[i]))
	}
	for i := range warm {
		serviceMust(t, m.core.DecRef(warm[i]))
	}
	// The monomial ID recycle vector is independently cold. Warm it through
	// actual ordinary factory/INC/DEC before the signed-domain recipe so late
	// allocator free cannot be offset by a genuine destructor-time allocation.
	variable, e := m.core.monomials.MkVar()
	serviceMust(t, e)
	monomial, e := m.core.monomials.MkVariable(variable, 3)
	serviceMust(t, e)
	serviceMust(t, m.core.monomials.IncRef(monomial))
	serviceMust(t, m.core.monomials.DecRef(monomial))
	parent, e := m.beginGCDFunction()
	serviceMust(t, e)
	parent.constructNumeral(0)
	serviceMust(t, q.z.Copy(&parent.nums[0], &q.z.two64))
	serviceMust(t, m.beginSquareRootWithReceipt(&parent.quadraticSqrt))
	child := &m.squareRoot
	child.constructNumeral(1)
	child.constructNumeral(2)
	child.constructNumeral(3)
	child.referenceCompleted = true
	serviceMust(t, q.z.SetInt32(&child.values[0], 13))
	for i := 1; i < 4; i++ {
		serviceMust(t, q.z.Copy(&child.values[i], &q.z.two64))
	}
	serviceMust(t, q.z.ensureOwnedCapacity(&child.values[3], 40000))
	older, depth := parent.nums[0], parent.depth
	contextTLSNearMinimum(t, rt)
	first := m.finishSquareRoot(nil)
	if !errors.Is(first, arena.ErrBounds) || child.sourceCause == nil {
		t.Fatal("actual younger source stop", first)
	}
	m.gcdDepth--
	outer := m.finishGCDFunctionSource(parent)
	m.core.pscLeave()
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	if !errors.Is(outer, arena.ErrBounds) || parent.cleanupCause != child.sourceCause || parent.nums[0] != older || parent.depth != depth {
		t.Fatal("source outer must stop before older NUM", outer)
	}
	saved, state := parent.cleanupCause, rt.State()
	again := m.finishGCDFunctionSource(parent)
	if !errors.Is(again, arena.ErrBounds) || parent.cleanupCause != saved || parent.nums[0] != older || parent.depth != depth || rt.State() != state {
		t.Fatal("sticky outer source retry", again)
	}
	contextTLSFinish(t, rt, m.CleanupGCD)
	if parent.active || child.sourceCause != saved || !errors.Is(outer, arena.ErrBounds) {
		t.Fatal("owning completion or literal child history")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}
