// Source: src/math/polynomial/polynomial.cpp:3318-3338;
// src/util/mpzzp.h:58-68,111-114; src/util/mpz.cpp:quot_rem_core.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialConstantCoefficientNormalizeActualRemScratchLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = runtime
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	large := Int{}
	limbs := make([]uint32, 128)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, c.integer.SetLimbs(&large, limbs))
	p, e := c.MkConstantTakeCopy(&large)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZPUint64(251))
	// SET allocates 528 actual TLS bytes first. The direct-profile old tmp0 release subtracts
	// forty (the pooled route links it without Runtime free); the reached
	// 784-byte Rem scratch allocation crosses the actual
	// synchronization threshold. This is an external real allocation, not a
	// reservation inserted into the numerical engine.
	pad := int64(100000-700-8) - runtime.State().LocalSize
	if pad <= 0 {
		t.Fatal("cold fixture source lane already beyond intended prefix", runtime.State())
	}
	external, e := runtime.Allocate(uint64(pad))
	serviceMust(t, e)
	out := NewInt(19)
	var receipt polynomialConstantCoefficientInvocationReceipt
	defer func() {
		serviceMust(t, runtime.SetMaxSize(0))
		serviceMust(t, c.integer.Del(&out))
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
		// Historical RED teardown uses the actual named generic integer owner;
		// this is not the new invocation receipt's authority or completion proof.
		serviceMust(t, c.integer.CleanupPendingAllocation())
		serviceMust(t, runtime.Deallocate(&external))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&large))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Fatal("named owner cleanup retained charge", b.Used())
		}
	}()
	oldTmp := c.integer.tmp[0]
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !runtime.State().OutOfMemory || out.IsSmall() || out.cell == c.integer.failedRef || out.cell == c.integer.failedNullRef || c.integer.tmp[0] != 0 || oldTmp == 0 {
		t.Fatal("actual SET succeeded then reached Rem scratch refusal", e, out, c.integer.tmp[0])
	}
	roots, e := c.integer.allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (memory.Block{}) || c.integer.pendingCell.object != (memory.ObjectRef{}) {
		t.Fatal("actual new raw scratch before cell publication", roots)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external})
	if !c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("ConstantCoefficient receipt lost reached PNormalize scratch acquisition")
	}
	used, state, stats := b.Used(), runtime.State(), runtime.Stats()
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("no lexical ConstantCoefficient NUM destructor may consume scratch prefix")
	}
	// Caller NUM destruction precedes the explicit unjoined scratch owner.
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	remaining, e := c.integer.allocator.PendingRoots()
	serviceMust(t, e)
	if remaining[0] != (memory.Block{}) || c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("exact reached Normalize leaf remains owned")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external})
}

func TestPolynomialConstantCoefficientNormalizeActualNaturalGrowthLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = runtime
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	large := Int{}
	limbs := make([]uint32, 128)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, c.integer.SetLimbs(&large, limbs))
	p, e := c.MkConstantTakeCopy(&large)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZPUint64(251))
	// SET (528) and Rem cell scratch (784) publish before the real Natural
	// buffer's first128-byte publication and later256-byte growth. The latter
	// crosses TLS synchronization and retains only its unpublished raw token.
	pad := int64(100000-1600-8) - runtime.State().LocalSize
	if pad <= 0 {
		t.Fatal("cold fixture source lane already beyond intended prefix", runtime.State())
	}
	external, e := runtime.Allocate(uint64(pad))
	serviceMust(t, e)
	out := NewInt(19)
	var receipt polynomialConstantCoefficientInvocationReceipt
	defer func() {
		serviceMust(t, runtime.SetMaxSize(0))
		serviceMust(t, c.integer.Del(&out))
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
		// Historical RED teardown uses the actual named generic integer owner;
		// this is not the new invocation receipt's authority or completion proof.
		serviceMust(t, c.integer.CleanupPendingAllocation())
		for i := range c.integer.natural.division {
			if c.integer.natural.division[i].pending != (memory.Block{}) {
				serviceMust(t, runtime.Deallocate(&c.integer.natural.division[i].pending))
			}
		}
		serviceMust(t, runtime.Deallocate(&external))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&large))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Fatal("named owner cleanup retained charge", b.Used())
		}
	}()
	oldTmp := c.integer.tmp[0]
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !runtime.State().OutOfMemory || out.IsSmall() || out.cell == c.integer.failedRef || out.cell == c.integer.failedNullRef || c.integer.tmp[0] == 0 || oldTmp == 0 {
		t.Fatal("actual SET succeeded then reached Rem scratch refusal", e, out, c.integer.tmp[0])
	}
	roots, e := c.integer.allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] != (memory.Block{}) || c.integer.natural.division[0].pending == (memory.Block{}) {
		t.Fatal("actual new Natural buffer pending after shared earlier publication", roots)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external})
	if !c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("ConstantCoefficient receipt lost reached Natural pending growth")
	}
	shared := c.integer.natural.division[0].raw
	if shared == (memory.Block{}) {
		t.Fatal("earlier actual shared buffer was not published")
	}
	used, state, stats := b.Used(), runtime.State(), runtime.Stats()
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("no lexical ConstantCoefficient NUM destructor may consume scratch prefix")
	}
	oldPN := receipt.frame.normalize
	wrongPN := oldPN
	wrongFrame := *oldPN.frame
	wrongPN.frame = &wrongFrame
	serviceMust(t, c.modular.cleanupPNormalizeInvocation(&wrongPN, true))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("foreign frame receipt changed actual pending cohort")
	}
	// Caller NUM destruction precedes the explicit unjoined scratch owner.
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	remaining, e := c.integer.allocator.PendingRoots()
	serviceMust(t, e)
	if remaining[0] != (memory.Block{}) || c.integer.natural.division[0].pending != (memory.Block{}) || c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("exact reached Normalize leaf remains owned")
	}
	if c.integer.natural.division[0].raw != shared {
		t.Fatal("receipt consumed already-published shared buffer")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external})
	// A second actual body reuses the frame with a unique invocation. Its SET
	// precedes a new256-byte pending Natural growth; the first completed receipt
	// cannot consume this same-cardinality replacement.
	serviceMust(t, runtime.SetMaxSize(0))
	pad2 := int64(100000-650-8) - runtime.State().LocalSize
	if pad2 <= 0 {
		t.Fatal("second real prefix cannot reach selected threshold", runtime.State())
	}
	extra, e := runtime.Allocate(uint64(pad2))
	serviceMust(t, e)
	defer func() { serviceMust(t, runtime.SetMaxSize(0)); serviceMust(t, runtime.Deallocate(&extra)) }()
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !c.modular.matchesPNormalizeInvocation(&receipt.frame.normalize) || receipt.frame.normalize.invocation == oldPN.invocation || c.integer.natural.division[0].pending == (memory.Block{}) {
		t.Fatal("new actual Normalize body did not acquire a replacement cohort", e)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external, extra})
	used, state, stats = b.Used(), runtime.State(), runtime.Stats()
	newPending := c.integer.natural.division[0].pending
	serviceMust(t, c.modular.cleanupPNormalizeInvocation(&oldPN, true))
	if newPending != c.integer.natural.division[0].pending || b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("stale receipt consumed later real pending cohort")
	}
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external, extra})
}

func TestPolynomialConstantCoefficientNormalizeActualVerificationConstructorLeaf(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	b := arena.NewBudget(1 << 26)
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = runtime
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	large := Int{}
	limbs := make([]uint32, 128)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, c.integer.SetLimbs(&large, limbs))
	p, e := c.MkConstantTakeCopy(&large)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZPUint64(251))
	// Shared Algorithm-D storage publishes fully before the checked lexical
	// constructor's later real1024-byte growth fails at synchronization. Earlier
	// constructor512-byte raw storage remains untransferred and not constructed.
	pad := int64(100000-3100-8) - runtime.State().LocalSize
	if pad <= 0 {
		t.Fatal("cold fixture source lane already beyond intended prefix", runtime.State())
	}
	external, e := runtime.Allocate(uint64(pad))
	serviceMust(t, e)
	out := NewInt(19)
	var receipt polynomialConstantCoefficientInvocationReceipt
	defer func() {
		serviceMust(t, runtime.SetMaxSize(0))
		serviceMust(t, c.integer.Del(&out))
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
		// Historical RED teardown uses the actual named generic integer owner;
		// this is not the new invocation receipt's authority or completion proof.
		serviceMust(t, c.integer.CleanupPendingAllocation())
		serviceMust(t, c.integer.natural.verification.finalize(runtime))
		c.integer.natural.verificationActive = false
		for i := range c.integer.natural.division {
			if c.integer.natural.division[i].pending != (memory.Block{}) {
				serviceMust(t, runtime.Deallocate(&c.integer.natural.division[i].pending))
			}
		}
		serviceMust(t, runtime.Deallocate(&external))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&large))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Fatal("named owner cleanup retained charge", b.Used())
		}
	}()
	oldTmp := c.integer.tmp[0]
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !runtime.State().OutOfMemory || out.IsSmall() || out.cell == c.integer.failedRef || out.cell == c.integer.failedNullRef || c.integer.tmp[0] == 0 || oldTmp == 0 {
		t.Fatal("actual SET succeeded then reached Rem scratch refusal", e, out, c.integer.tmp[0])
	}
	roots, e := c.integer.allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] != (memory.Block{}) || c.integer.natural.verification.pending == (memory.Block{}) {
		t.Fatal("actual new Natural buffer pending after shared earlier publication", roots)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external})
	if !c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("ConstantCoefficient receipt lost reached verification constructor growth")
	}
	shared := c.integer.natural.verification.raw
	if shared == (memory.Block{}) {
		t.Fatal("earlier actual shared buffer was not published")
	}
	used, state, stats := b.Used(), runtime.State(), runtime.Stats()
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("no lexical ConstantCoefficient NUM destructor may consume scratch prefix")
	}
	oldVerification := receipt.frame.normalize.frame.verification
	wrongVerification := oldVerification
	wrongFrame := *oldVerification.frame
	wrongVerification.frame = &wrongFrame
	serviceMust(t, c.integer.natural.cleanupPNVerification(&wrongVerification, true))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("foreign verification frame changed exact cohort")
	}
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	if c.integer.natural.verification.pending != (memory.Block{}) || c.integer.natural.verification.raw != (memory.Block{}) || c.integer.natural.verificationActive {
		t.Fatal("unfinished verification ctor still owns exact roots")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external})
	serviceMust(t, runtime.SetMaxSize(0))
	pad2 := int64(100000-1800-8) - runtime.State().LocalSize
	if pad2 <= 0 {
		t.Fatal("second verification constructor threshold", runtime.State())
	}
	extra, e := runtime.Allocate(uint64(pad2))
	serviceMust(t, e)
	defer func() { serviceMust(t, runtime.SetMaxSize(0)); serviceMust(t, runtime.Deallocate(&extra)) }()
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("second actual checked constructor did not stop", e)
	}
	child := receipt.frame.normalize.frame.verification
	if !c.integer.natural.matchesPNVerification(&child) || child.invocation == oldVerification.invocation || c.integer.natural.verification.pending == (memory.Block{}) || c.integer.natural.verification.raw == (memory.Block{}) {
		t.Fatal("same-frame unique actual verification replacement")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external, extra})
	used, state, stats = b.Used(), runtime.State(), runtime.Stats()
	pending, published := c.integer.natural.verification.pending, c.integer.natural.verification.raw
	serviceMust(t, c.integer.natural.cleanupPNVerification(&oldVerification, true))
	if pending != c.integer.natural.verification.pending || published != c.integer.natural.verification.raw || b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("stale verification receipt consumed later constructor cohort")
	}
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, []memory.Block{external, extra})
}

// Public TLS operations deliberately reach the source signed-domain guard;
// this proves the exact lexical stop, not an ordinary exhaustion path.
func TestPolynomialConstantCoefficientNormalizeVerificationDTBeforeCallerNUM(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	b := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, warm, out Int
	digits := make([]uint32, 16)
	digits[0] = 17
	digits[15] = 1
	serviceMust(t, c.integer.SetLimbs(&large, digits))
	p, e := c.MkConstantTakeCopy(&large)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZPUint64(251))
	// Actual ordinary Rem warms the source persistent integer/Natural scratch.
	serviceMust(t, c.integer.Rem(&large, &c.modular.p, &warm))
	var receipt polynomialConstantCoefficientInvocationReceipt
	defer func() {
		// A real external allocation restores the signed accounting domain only
		// after the selected source/owning prefix proofs. It stays explicitly
		// named until all numerical owners have been retired.
		recovery, e := rt.Allocate(400000)
		serviceMust(t, e)
		contextTLSFinish(t, rt, func() error { return c.deleteNumeral(&out) })
		contextTLSFinish(t, rt, func() error { return c.cleanupConstantCoefficientInvocation(&receipt, true) })
		if e := c.DecRef(p); e != nil {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("source polynomial destruction", e)
			}
			contextTLSFinish(t, rt, func() error { return c.CleanupValue(&p) })
		}
		contextTLSFinish(t, rt, func() error { return c.integer.Del(&large) })
		contextTLSFinish(t, rt, func() error { return c.integer.Del(&warm) })
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, q.CloseChecked)
		e = rt.Deallocate(&recovery)
		if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
			t.Fatal("real recovery block consumption", e)
		}
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Fatal("named owner graph remains", b.Used())
		}
	}()
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	contextTLSNearMinimum(t, rt)
	e = rt.Deallocate(&external)
	if e == nil || external != (memory.Block{}) {
		t.Fatal("real external physical free must stop at signed guard", e, external)
	}
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || out.IsSmall() || !c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("younger completed verification DT must precede caller BIG NUM publication", e, out)
	}
	child := receipt.frame.normalize.frame.verification
	if !c.integer.natural.matchesPNVerification(&child) || !child.frame.constructed || child.frame.ended || child.frame.sourceCause == nil || c.integer.natural.verification.raw != (memory.Block{}) {
		t.Fatal("exact physically consumed completed verification DT", child)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &warm, &out}, 0, nil)
	caller := out
	used, state, stats := b.Used(), rt.State(), rt.Stats()
	cause := c.modular.pNormalizeInvocationSourceCause(&receipt.frame.normalize)
	if cause == nil {
		t.Fatal("missing younger sticky source cause")
	}
	if again := c.integer.natural.finishPNVerificationSource(child.frame); again != cause || out != caller || used != b.Used() || state != rt.State() || stats != rt.Stats() {
		t.Fatal("source destructor stop repeated actual free or advanced caller")
	}
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
	if c.modular.pNormalizeInvocationSourceCause(&receipt.frame.normalize) != cause || child.frame.sourceCause != cause || child.frame.ended || !child.frame.owned || out != caller {
		t.Fatal("owning acknowledgement must preserve actual source stop before caller NUM")
	}
	contextTLSFinish(t, rt, func() error { return c.deleteNumeral(&out) })
	contextTLSFinish(t, rt, func() error { return c.cleanupConstantCoefficientInvocation(&receipt, true) })
	if c.constantCoefficientInvocationSourceCause(&receipt) != cause || child.frame.sourceCause != cause || child.frame.ended || !child.frame.owned {
		t.Fatal("explicit abort erased child/parent historical source stop")
	}
	used, state, stats = b.Used(), rt.State(), rt.Stats()
	if again := c.integer.natural.finishPNVerificationSource(child.frame); again != cause || used != b.Used() || state != rt.State() || stats != rt.Stats() {
		t.Fatal("acknowledged source retry must retain exact cause without new action")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &warm, &out}, 0, nil)
	// A later real PN invocation may reuse its child slot without rewriting
	// the earlier ConstantCoefficient frame's retained source error history.
	small := NewInt(7)
	var later integerPNormalizeInvocationReceipt
	oldPN := receipt.frame.normalize
	serviceMust(t, c.modular.pNormalizeWithReceipt(&small, &later))
	if later.invocation == oldPN.invocation || c.constantCoefficientInvocationSourceCause(&receipt) != cause {
		t.Fatal("later child invocation erased parent source history")
	}
}

func TestPolynomialConstantCoefficientNormalizeEveryActualReservationPrefix(t *testing.T) {
	firstSuccess := -1
	for n := 0; n < 128; n++ {
		b := arena.NewBudget(1 << 26)
		rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
		serviceMust(t, e)
		cfg := serviceConfig(b).Integers
		cfg.Memory = rt
		q, e := NewRationalManager(RationalConfig{Integers: cfg})
		serviceMust(t, e)
		m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
		serviceMust(t, e)
		c := m.core
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		var large Int
		digits := make([]uint32, 128)
		digits[0] = 17
		digits[127] = 1
		serviceMust(t, c.integer.SetLimbs(&large, digits))
		p, e := c.MkConstantTakeCopy(&large)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		serviceMust(t, c.modular.SetZPUint64(251))
		out := NewInt(19)
		var receipt polynomialConstantCoefficientInvocationReceipt
		b.FailAfter(int64(n))
		constant, callError := c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
		// Every refusal's literal published prefix is audited while injection remains
		// active, including shared scratch and any unfinished lexical constructor.
		polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, nil)
		if callError == nil {
			var value uint64
			for i := len(digits) - 1; i >= 0; i-- {
				value = (value*(uint64(1)<<32) + uint64(digits[i])) % 251
			}
			expected := int32(value)
			if expected > 125 {
				expected -= 251
			}
			if !constant || !out.IsSmall() || out.val != expected {
				t.Fatal("independent bounded modular result", n, out, expected)
			}
			firstSuccess = n
		}
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
		serviceMust(t, c.deleteNumeral(&out))
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
		polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*Int{&large, &out}, 0, nil)
		b.FailAfter(-1)
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&large))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual prefix retained unnamed charge", n, b.Used())
		}
		if callError == nil {
			break
		}
	}
	if firstSuccess < 0 {
		t.Fatal("bounded campaign never reached first actual success")
	}
	t.Log("first actual success reservation prefix", firstSuccess)
}
