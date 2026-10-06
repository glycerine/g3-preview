// Source: src/math/polynomial/upolynomial.cpp:213-232,3042-3057;
// src/util/mpz.cpp:560-843; src/util/memory_manager.cpp:274-285.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestUPolynomialFactorFrontendActualCopyIntegerOwnerTransport(t *testing.T) {
	for _, mode := range []string{"close", "source", "rational-close", "rational-source", "factors-close"} {
		t.Run(mode, func(t *testing.T) { frontendCopyIntegerOwnerTransport(t, mode) })
	}
}
func frontendCopyIntegerOwnerTransport(t *testing.T, mode string, erased ...bool) {
	b := arena.NewBudget(1 << 26)
	var z *IntegerManager
	var q *RationalManager
	var fs *UPolynomialFactors
	armed := false
	calls := 0
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f memory.Failure) error {
		if armed {
			calls++
			before := b.Used()
			wasClosed, wasFinalizing := z.closed, z.sourceFinalizing
			rationalBefore := *q
			factorsBefore := *fs
			var stop error
			switch mode {
			case "close":
				stop = z.CloseChecked()
			case "source":
				stop = z.SourceFinalize()
			case "rational-close":
				stop = q.CloseChecked()
			case "rational-source":
				stop = q.SourceFinalize()
			case "factors-close":
				stop = fs.Close()
			}
			if !reflect.DeepEqual(rationalBefore, *q) {
				t.Error("executing frontend changed Rational owner before refusal")
			}
			expected := error(ErrPolynomialPending)
			if mode == "factors-close" {
				expected = ErrUPolynomialCleanupPending
			}
			if !reflect.DeepEqual(factorsBefore, *fs) {
				t.Error("executing frontend changed caller factors before refusal")
			}
			if stop != expected || z.closed != wasClosed || z.sourceFinalizing != wasFinalizing || before != b.Used() {
				t.Errorf("executing frontend primitive Copy destroyed Integer owner: stop=%v closed=%v source=%v bytes=%d/%d", stop, z.closed, z.sourceFinalizing, before, b.Used())
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Cells.PageSize = 1
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e = NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	z = q.z
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := NewPrimeGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	fs, e = m.NewFactors()
	serviceMust(t, e)
	var large Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, z.SetLimbs(&large, words))
	var input upolyVectorState
	inputValues := []Int{large, NewInt(0), NewInt(1)}
	if len(erased) != 0 && erased[0] {
		inputValues = []Int{large}
	}
	serviceMust(t, m.set(&input, inputValues))
	var held memory.Block
	defer func() {
		armed = false
		b.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		contextTLSFinish(t, rt, s.CleanupRetained)
		contextTLSFinish(t, rt, fs.Close)
		contextTLSFinish(t, rt, func() error { return m.reset(&input) })
		m.freeBacking(&input)
		contextTLSFinish(t, rt, func() error { return z.Del(&large) })
		contextTLSFinish(t, rt, func() error { return rt.Deallocate(&held) })
		contextTLSFinish(t, rt, s.Close)
		serviceMust(t, primes.Close())
		contextTLSFinish(t, rt, m.Close)
		contextTLSFinish(t, rt, q.CloseChecked)
		contextTLSFinish(t, rt, rt.Close)
		if b.Used() != 0 {
			t.Error("actual owner refunds", b.Used())
		}
	}()
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal("actual TLS source threshold", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	armed = false
	if body == nil || calls != 1 {
		t.Fatal("actual primitive BIG+zero Copy transport not reached", body, calls)
	}
	caller := []*Int{&large}
	for i := range m.values(&input) {
		caller = append(caller, &m.values(&input)[i])
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []memory.Block{held}, caller...)
}
