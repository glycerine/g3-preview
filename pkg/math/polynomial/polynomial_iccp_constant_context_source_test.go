package polynomial

// Source: src/math/polynomial/polynomial.cpp:3461-3465;
// src/util/mpzzp.h:111-114,229-235; Go returning error-transport boundary.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type iccpConstantContextProbe struct {
	full         *MultivariatePolynomialManager
	current, old integerAllocationInvocationContext
	calls        int
	failed       string
}

var iccpConstantContextCurrent *iccpConstantContextProbe

func iccpConstantContextTransport(f umemory.Failure) error {
	probe := iccpConstantContextCurrent
	if probe == nil {
		return f
	}
	m := probe.full
	s := &m.iccpConstant
	c := m.core
	actual := numericIntegerModularManager_iccpConstantAllocationContext(c.modular, &s.directReceipt)
	if (actual.roots(c.integer)) != (&s.numeric) {
		probe.failed = "actual executing SET context rejected"
	}
	copied := *s
	forged := s.directReceipt
	forged.frame = &copied
	invalid := integerAllocationInvocationContext{iccpConstant: &forged, iccpConstantInvocation: forged.invocation}
	if (invalid.roots(c.integer)) != nil {
		probe.failed = "copied frame acquired allocation authority"
	}
	foreignReceiver := *c.modular
	if numericIntegerModularManager_iccpConstantAllocationContext(&foreignReceiver, &s.directReceipt) != (integerAllocationInvocationContext{}) {
		probe.failed = "foreign modular receiver acquired constant authority"
	}
	c.callbackDepth++
	if (actual.roots(c.integer)) != nil {
		probe.failed = "callback-active core acquired numeric authority"
	}
	c.callbackDepth--
	if probe.old.iccpConstant != nil && probe.old.iccpConstantInvocation != s.invocation && (probe.old.roots(c.integer)) != nil {
		probe.failed = "saved receipt pointer retargeted later executing invocation"
	}
	probe.current = actual
	probe.calls++
	return f
}
func TestPolynomialICCPConstantActualExecutingContextForeignAndReusedNonce(t *testing.T) {
	budget := arena.NewBudget(1 << 25)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, Transport: iccpConstantContextTransport})
	serviceMust(t, e)
	cfg := unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 1, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 1, Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	var big, i unumeral.Int
	words := make([]uint32, 8192)
	words[0], words[len(words)-1] = 7, 1
	serviceMust(t, c.integer.SetLimbs(&big, words))
	p, e := c.MkConstantTakeCopy(&big)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var content, primitive polynomialReference
	probe := iccpConstantContextProbe{full: full}
	t.Cleanup(func() {
		iccpConstantContextCurrent = nil
		budget.FailAfter(-1)
		serviceMust(t, full.CleanupContentPrimitiveParts())
		serviceMust(t, content.close(c))
		serviceMust(t, primitive.close(c))
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&big))
		serviceMust(t, c.integer.Del(&i))
		serviceMust(t, full.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("actual constant context owner graph", budget.Used())
		}
	})
	for pass := 0; pass < 2; pass++ {
		reached := false
		for prefix := int64(0); prefix < 64 && !reached; prefix++ {
			var receipt polynomialICCPInvocationReceipt
			before := probe.calls
			iccpConstantContextCurrent = &probe
			budget.FailAfter(prefix)
			body := full.iccpPartsWithReceipt(p, 0, &i, &content, &primitive, &receipt)
			iccpConstantContextCurrent = nil
			if probe.failed != "" {
				t.Fatal(probe.failed)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, content.value, primitive.value}, []*unumeral.Int{&big, &i}, 0)
			if probe.calls > before {
				reached = true
				if body == nil {
					t.Fatal("returning transport did not reach actual SET refusal")
				}
				if pass == 0 {
					probe.old = probe.current
				}
			}
			budget.FailAfter(-1)
			serviceMust(t, full.cleanupICCPInvocation(&receipt, false))
			serviceMust(t, full.cleanupICCPInvocation(&receipt, true))
			serviceMust(t, c.integer.Del(&i))
		}
		if !reached {
			t.Fatal("actual runtime allocation transport horizon absent", pass)
		}
	}
}
