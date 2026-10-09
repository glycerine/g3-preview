// Source: src/math/polynomial/polynomial.cpp:3318-3338;
// src/util/mpzzp.h:58-68,111-114; src/util/mpz.cpp:quot_rem_core.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialConstantCoefficientNormalizeActualRemScratchLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	runtime, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = runtime
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	large := util.Int{}
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
	out := util.NewInt(19)
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
	oldTmp := c.integer.Tmp[0]
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !runtime.State().OutOfMemory || out.IsSmall() || (out).Cell == c.integer.FailedRef || (out).Cell == c.integer.FailedNullRef || c.integer.Tmp[0] != 0 || oldTmp == 0 {
		t.Fatal("actual SET succeeded then reached Rem scratch refusal", e, out, c.integer.Tmp[0])
	}
	roots, e := c.integer.SourceAllocator().PendingRoots()
	serviceMust(t, e)
	if roots[0] == (util.Block{}) || c.integer.PendingAllocationObject() != (util.ObjectRef{}) {
		t.Fatal("actual new raw scratch before cell publication", roots)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, []util.Block{external})
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
	remaining, e := c.integer.SourceAllocator().PendingRoots()
	serviceMust(t, e)
	if remaining[0] != (util.Block{}) || c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("exact reached Normalize leaf remains owned")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, []util.Block{external})
}

func TestPolynomialConstantCoefficientNormalizeActualNaturalGrowthLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	runtime, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = runtime
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	large := util.Int{}
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
	out := util.NewInt(19)
	var receipt polynomialConstantCoefficientInvocationReceipt
	defer func() {
		serviceMust(t, runtime.SetMaxSize(0))
		serviceMust(t, c.integer.Del(&out))
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
		// Historical RED teardown uses the actual named generic integer owner;
		// this is not the new invocation receipt's authority or completion proof.
		serviceMust(t, c.integer.CleanupPendingAllocation())
		for i := range c.integer.Naturals().Division {
			if c.integer.Naturals().Division[i].Pending != (util.Block{}) {
				serviceMust(t, runtime.Deallocate(&c.integer.Naturals().Division[i].Pending))
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
	oldTmp := c.integer.Tmp[0]
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !runtime.State().OutOfMemory || out.IsSmall() || (out).Cell == c.integer.FailedRef || (out).Cell == c.integer.FailedNullRef || c.integer.Tmp[0] == 0 || oldTmp == 0 {
		t.Fatal("actual SET succeeded then reached Rem scratch refusal", e, out, c.integer.Tmp[0])
	}
	roots, e := c.integer.SourceAllocator().PendingRoots()
	serviceMust(t, e)
	if roots[0] != (util.Block{}) || c.integer.Naturals().Division[0].Pending == (util.Block{}) {
		t.Fatal("actual new Natural buffer pending after shared earlier publication", roots)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, []util.Block{external})
	if !c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("ConstantCoefficient receipt lost reached Natural pending growth")
	}
	shared := c.integer.Naturals().Division[0].Raw
	if shared == (util.Block{}) {
		t.Fatal("earlier actual shared buffer was not published")
	}
	used, state, stats := b.Used(), runtime.State(), runtime.Stats()
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("no lexical ConstantCoefficient NUM destructor may consume scratch prefix")
	}
	oldPN := receipt.frame.normalize
	wrongPN := oldPN
	wrongFrame := *oldPN.Frame
	wrongPN.Frame = &wrongFrame
	serviceMust(t, c.modular.CleanupPNormalizeInvocation(&wrongPN, true))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("foreign frame receipt changed actual pending cohort")
	}
	// Caller NUM destruction precedes the explicit unjoined scratch owner.
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	remaining, e := c.integer.SourceAllocator().PendingRoots()
	serviceMust(t, e)
	if remaining[0] != (util.Block{}) || c.integer.Naturals().Division[0].Pending != (util.Block{}) || c.matchesConstantCoefficientInvocation(&receipt) {
		t.Fatal("exact reached Normalize leaf remains owned")
	}
	if c.integer.Naturals().Division[0].Raw != shared {
		t.Fatal("receipt consumed already-published shared buffer")
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, []util.Block{external})
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
	if e == nil || !c.modular.MatchesPNormalizeInvocation(&receipt.frame.normalize) || receipt.frame.normalize.Invocation == oldPN.Invocation || c.integer.Naturals().Division[0].Pending == (util.Block{}) {
		t.Fatal("new actual Normalize body did not acquire a replacement cohort", e)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, []util.Block{external, extra})
	used, state, stats = b.Used(), runtime.State(), runtime.Stats()
	newPending := c.integer.Naturals().Division[0].Pending
	serviceMust(t, c.modular.CleanupPNormalizeInvocation(&oldPN, true))
	if newPending != c.integer.Naturals().Division[0].Pending || b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("stale receipt consumed later real pending cohort")
	}
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, []util.Block{external, extra})
}

func TestPolynomialConstantCoefficientNormalizeEveryActualReservationPrefix(t *testing.T) {
	firstSuccess := -1
	for n := 0; n < 128; n++ {
		b := arena.NewBudget(1 << 26)
		rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
		serviceMust(t, e)
		cfg := serviceConfig(b).Integers
		cfg.Memory = rt
		q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
		serviceMust(t, e)
		m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
		serviceMust(t, e)
		c := m.core
		_, e = c.monomials.MkVar()
		serviceMust(t, e)
		var large util.Int
		digits := make([]uint32, 128)
		digits[0] = 17
		digits[127] = 1
		serviceMust(t, c.integer.SetLimbs(&large, digits))
		p, e := c.MkConstantTakeCopy(&large)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		serviceMust(t, c.modular.SetZPUint64(251))
		out := util.NewInt(19)
		var receipt polynomialConstantCoefficientInvocationReceipt
		b.FailAfter(int64(n))
		constant, callError := c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
		// Every refusal's literal published prefix is audited while injection remains
		// active, including shared scratch and any unfinished lexical constructor.
		polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, nil)
		if callError == nil {
			var value uint64
			for i := len(digits) - 1; i >= 0; i-- {
				value = (value*(uint64(1)<<32) + uint64(digits[i])) % 251
			}
			expected := int32(value)
			if expected > 125 {
				expected -= 251
			}
			if !constant || !out.IsSmall() || out.NativeScalar() != expected {
				t.Fatal("independent bounded modular result", n, out, expected)
			}
			firstSuccess = n
		}
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
		serviceMust(t, c.deleteNumeral(&out))
		serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
		polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, []*util.Int{&large, &out}, 0, nil)
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
