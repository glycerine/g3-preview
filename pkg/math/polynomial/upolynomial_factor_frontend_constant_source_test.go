// Source: src/math/polynomial/upolynomial.cpp:3042-3051;
// src/math/polynomial/upolynomial.h:401-423; src/util/mpz.cpp:560-584.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestUPolynomialFactorFrontendRepeatedErasedSETWithRetainedRAW(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 256 && !reached; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var input, singleton upolyVectorState
			var b unumeral.Int
			serviceMust(t, z.Integers().SetLimbs(&b, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, m.set(&input, []unumeral.Int{unumeral.NewInt(1), b, unumeral.NewInt(1)}))
			serviceMust(t, m.set(&singleton, []unumeral.Int{unumeral.NewInt(0)}))
			defer func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
				serviceMust(t, m.reset(&singleton))
				m.freeBacking(&singleton)
				serviceMust(t, z.Integers().Del(&b))
			}()
			roots := []*unumeral.Int{&b}
			for i := range m.values(&input) {
				roots = append(roots, &m.values(&input)[i])
			}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			q := &s.frontend
			c := &q.modularPower
			if body == nil || !c.active || c.endEntered || (c.power).Cell == 0 {
				return
			}
			reached = true
			oldReceipt, oldPower, oldCause, oldSource := q.receipt, c.power, q.bodyCause, q.sourceCause
			z.Budget().FailAfter(0)
			before := z.Budget().Used()
			for repeat := 0; repeat < 3; repeat++ {
				complete, e := s.Factor(UPolynomialView{}, fs, DefaultUPolynomialFactorParams())
				serviceMust(t, e)
				if !complete {
					t.Fatal("erased zero route failed")
				}
				complete, e = s.Factor(m.stateView(&singleton), fs, DefaultUPolynomialFactorParams())
				serviceMust(t, e)
				if !complete {
					t.Fatal("erased singleton route failed")
				}
				serviceMust(t, fs.Reset())
				if z.Budget().Used() != before || q.receipt != oldReceipt || c.power != oldPower || q.bodyCause != oldCause || q.sourceCause != oldSource || !c.active {
					t.Fatal("inline SET reserved history or altered older retained source")
				}
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			}
		})
	}
	if !reached {
		t.Fatal("actual failed Power RAW not reached")
	}
}

func TestUPolynomialFactorFrontendErasedBIGSETOwnsActualPendingLeaf(t *testing.T) {
	reached, complete := false, false
	for ordinal := int64(0); ordinal < 64 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var input upolyVectorState
			var b unumeral.Int
			serviceMust(t, z.Integers().SetLimbs(&b, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, m.set(&input, []unumeral.Int{b}))
			defer func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
				serviceMust(t, z.Integers().Del(&b))
			}()
			roots := []*unumeral.Int{&b, &m.values(&input)[0]}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			if body == nil {
				complete = true
				return
			}
			if z.Integers().PendingCell.Object == (umemory.ObjectRef{}) {
				return
			}
			reached = true
			captured := s.frontendConstant.record
			if captured == nil || captured.numeric.Cell != z.Integers().PendingCell || captured.invocation != s.frontendConstant.invocation {
				t.Fatal("failed SET leaf not held in its actual allocating call", ordinal)
			}
			originalInput, originalB := m.values(&input)[0], b
			firstBody, firstSource := captured.bodyCause, captured.sourceCause
			e = s.CleanupRetained()
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("exact SET own cleanup", ordinal, e)
			}
			if e != nil {
				primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
				if captured.ownCause != e {
					t.Fatal("matched cleanup refusal did not retain first own cause", e, captured.ownCause)
				}
				firstOwn := captured.ownCause
				z.Budget().FailAfter(-1)
				serviceMust(t, s.cleanupFrontendConstantCall(captured))
				if captured.ownCause != firstOwn || captured.bodyCause != firstBody || captured.sourceCause != firstSource {
					t.Fatal("cleanup retry erased first body/source/own history")
				}
			}
			if captured.bodyCause != firstBody || captured.sourceCause != firstSource {
				t.Fatal("SET own cleanup erased original cause")
			}
			if z.Integers().PendingCell.Object != (umemory.ObjectRef{}) {
				t.Fatal("service own ACK did not retire its actual failed early SET leaf", ordinal, z.Integers().PendingCell)
			}
			if m.values(&input)[0] != originalInput || b != originalB {
				t.Fatal("early SET own ACK changed caller NUMs")
			}
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
		})
	}
	if !reached || !complete {
		t.Fatal("actual BIG singleton pending SET leaf and complete horizon not reached", reached, complete)
	}
}

func TestUPolynomialFactorFrontendErasedSETActualOwnerTransport(t *testing.T) {
	for _, mode := range []string{"close", "source", "rational-close", "rational-source", "factors-close"} {
		t.Run(mode, func(t *testing.T) { frontendCopyIntegerOwnerTransport(t, mode, true) })
	}
}

// Real successive public singleton SET calls issue distinct allocating frames.
// Copy rejection here is an invalid-Go owner guard; the pending leaf and old
// acknowledgment checks use only actual reached source invocations.
func TestUPolynomialFactorFrontendConstantOldACKPreservesLaterPending(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	var b unumeral.Int
	serviceMust(t, z.Integers().SetLimbs(&b, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	serviceMust(t, m.set(&input, []unumeral.Int{b}))
	defer func() {
		z.Budget().FailAfter(-1)
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
		serviceMust(t, z.Integers().Del(&b))
	}()
	roots := []*unumeral.Int{&b, &m.values(&input)[0]}
	acquire := func() *upolyFactorConstantCall {
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			q := s.frontendConstant.record
			if body != nil && q != nil && q.numeric.Cell.Object != (umemory.ObjectRef{}) {
				return q
			}
			z.Budget().FailAfter(-1)
			serviceMust(t, s.CleanupRetained())
			serviceMust(t, fs.Reset())
		}
		t.Fatal("actual subsequent allocating SET pending leaf not reached")
		return nil
	}
	old := acquire()
	z.Budget().FailAfter(-1)
	serviceMust(t, s.cleanupFrontendConstantCall(old))
	firstBody, firstSource, firstOwn := old.bodyCause, old.sourceCause, old.ownCause
	if !old.retired {
		t.Fatal("old actual allocating frame not acknowledged")
	}
	later := acquire()
	if later == old || later.invocation == old.invocation {
		t.Fatal("later actual SET reused old allocating frame")
	}
	savedNumeric, savedPending := later.numeric, z.Integers().PendingCell
	savedCause, savedSource, savedOwn := later.bodyCause, later.sourceCause, later.ownCause
	beforeRuntime, beforeBudget := z.Integers().RuntimeOwner().Stats(), z.Budget().Used()
	copyOld := *old
	copyOld.self = &copyOld
	if e := s.cleanupFrontendConstantCall(&copyOld); e != ErrUPolynomialOwner {
		t.Fatal("copied allocating frame acquired cleanup", e)
	}
	serviceMust(t, s.cleanupFrontendConstantCall(old))
	if (later.numeric).NumericAllocationRoots != (savedNumeric).NumericAllocationRoots || z.Integers().PendingCell != savedPending || later.bodyCause != savedCause || later.sourceCause != savedSource || later.ownCause != savedOwn || z.Integers().RuntimeOwner().Stats() != beforeRuntime || z.Budget().Used() != beforeBudget {
		t.Fatal("old ACK changed later actual pending SET or physical graph")
	}
	if old.bodyCause != firstBody || old.sourceCause != firstSource || old.ownCause != firstOwn {
		t.Fatal("old immutable history changed")
	}
	primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
}

// The literal count-one branch performs only SET even in field mode; no
// content/vector declarations or general factorization algorithm are reached.
func TestUPolynomialFactorFrontendErasedFieldSETActualPNPrefixes(t *testing.T) {
	complete, reached, failedPN := false, false, false
	for ordinal := int64(0); ordinal < 128 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			z, s := preparationPhysicalFixture(t)
			m := s.manager
			var modulus, value unumeral.Int
			words := make([]uint32, 17)
			for i := 0; i < 16; i++ {
				words[i] = 0xffffffff
			}
			words[16] = 0x1ff
			serviceMust(t, z.Integers().SetLimbs(&modulus, words))
			words[16] = 0xff
			serviceMust(t, z.Integers().SetLimbs(&value, words))
			serviceMust(t, m.modular.SetZP(&modulus))
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var input upolyVectorState
			serviceMust(t, m.set(&input, []unumeral.Int{value}))
			defer func() {
				z.Budget().FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				serviceMust(t, fs.Close())
				serviceMust(t, m.reset(&input))
				m.freeBacking(&input)
				serviceMust(t, m.modular.SetZ())
				serviceMust(t, z.Integers().Del(&value))
				serviceMust(t, z.Integers().Del(&modulus))
			}()
			roots := []*unumeral.Int{&value, &modulus, &m.values(&input)[0]}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			if body == nil {
				complete = true
				reached = reached || s.frontendConstant.normalize.Invocation != 0
				return
			}
			q := s.frontendConstant.record
			if q == nil {
				return
			}
			if q.normalize.Invocation != 0 {
				reached = true
				failedPN = true
			}
			oldBody, oldSource, child := q.bodyCause, q.sourceCause, q.normalize
			e = s.cleanupFrontendConstantCall(q)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual field SET own refusal", ordinal, e)
			}
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			if q.bodyCause != oldBody || q.sourceCause != oldSource || q.normalize != child {
				t.Fatal("field SET ACK erased actual captured PN/history")
			}
		})
	}
	if !complete || !reached || failedPN {
		t.Fatal("field SET reaches PN without native verification allocation failures", complete, reached, failedPN)
	}
}

func TestUPolynomialFactorFrontendConstantDirectLeafAfterIntegerSourceEnd(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 32 && !reached; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			budget := arena.NewBudget(1 << 26)
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 64})
			serviceMust(t, e)
			cfg := serviceConfig(budget).Integers
			cfg.Cells.PageSize = 1
			cfg.Memory = rt
			cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			z, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
			serviceMust(t, e)
			m, e := NewUPolynomialManager(UPolynomialConfig{Integer: z.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
			serviceMust(t, e)
			primes, e := primegen.NewGenerator(budget)
			serviceMust(t, e)
			s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
			serviceMust(t, e)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			var value unumeral.Int
			words := make([]uint32, 80)
			words[79] = 1
			serviceMust(t, z.Integers().SetLimbs(&value, words))
			var input upolyVectorState
			serviceMust(t, m.set(&input, []unumeral.Int{value}))

			defer func() {
				budget.FailAfter(-1)
				serviceMust(t, s.CleanupRetained())
				if !fs.closed {
					serviceMust(t, fs.Close())
				}
				if !m.closed {
					serviceMust(t, m.reset(&input))
					m.freeBacking(&input)
					serviceMust(t, z.Integers().Del(&value))
				}
				serviceMust(t, s.Close())
				serviceMust(t, primes.Close())
				if !m.closed {
					serviceMust(t, m.Close())
				}
				serviceMust(t, z.CloseChecked())
				serviceMust(t, rt.Close())
				if budget.Used() != 0 {
					t.Error("source-end exact owner refunds", budget.Used())
				}
			}()
			roots := []*unumeral.Int{&value, &m.values(&input)[0]}
			z.Budget().FailAfter(ordinal)
			_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
			primeTrialPhysicalAuditOwners(t, z, s, []*UPolynomialFactors{fs}, roots...)
			q := s.frontendConstant.record
			if body == nil || q == nil || q.numeric.Cell.Object == (umemory.ObjectRef{}) || q.numeric.Cell.Offset != 0 {
				return
			}
			reached = true
			savedCell, savedBody, savedSource := q.numeric.Cell, q.bodyCause, q.sourceCause
			z.Budget().FailAfter(-1)
			serviceMust(t, fs.Close())
			serviceMust(t, m.reset(&input))
			m.freeBacking(&input)
			serviceMust(t, z.Integers().Del(&value))
			serviceMust(t, m.Close())
			// Retained numeric authority is the actual allocating call frame;
			// all ordinary upper caller objects have completed destruction.
			primeTrialPhysicalAuditOwners(t, z, s, nil)
			serviceMust(t, z.SourceFinalize())
			if !z.Integers().SourceFinalized {
				t.Fatal("actual Integer source owner did not end")
			}
			e = s.cleanupFrontendConstantCall(q)
			if e != nil {
				t.Fatal("exact failed direct SET leaf not disposable after actual Integer source end", ordinal, e, savedCell)
			}
			if q.numeric.Cell != (unumeral.SourceCellBacking{}) || q.bodyCause != savedBody || q.sourceCause != savedSource {
				t.Fatal("source-end own ACK lost exact cause or retained physical leaf")
			}
			primeTrialPhysicalAuditOwners(t, z, s, nil)
		})
	}
	if !reached {
		t.Fatal("actual direct failed SET leaf/source-end frontier not reached")
	}
}
