// Source: src/math/polynomial/upolynomial.cpp:213-232,3042-3057;
// src/util/mpz.cpp:560-843; src/util/memory_manager.cpp:274-285.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestUPolynomialFactorFrontendActualCopyIntegerOwnerTransport(t *testing.T) {
	for _, mode := range []string{"close", "source", "rational-close", "rational-source", "factors-close"} {
		t.Run(mode, func(t *testing.T) { frontendCopyIntegerOwnerTransport(t, mode) })
	}
}
func frontendCopyIntegerOwnerTransport(t *testing.T, mode string, erased ...bool) {
	b := arena.NewBudget(1 << 26)
	var z *util.IntegerManager
	var q *util.RationalManager
	var fs *UPolynomialFactors
	armed := false
	calls := 0
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f util.Failure) error {
		if armed {
			calls++
			before := b.Used()
			wasClosed, wasFinalizing := z.Closed(), z.SourceFinalizing
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
			if stop != expected || z.Closed() != wasClosed || z.SourceFinalizing != wasFinalizing || before != b.Used() {
				t.Errorf("executing frontend primitive Copy destroyed Integer owner: stop=%v closed=%v source=%v bytes=%d/%d", stop, z.Closed(), z.SourceFinalizing, before, b.Used())
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Cells.PageSize = 1
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e = util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	z = q.Integers()
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	fs, e = m.NewFactors()
	serviceMust(t, e)
	var large util.Int
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, z.SetLimbs(&large, words))
	var input upolyVectorState
	inputValues := []util.Int{large, util.NewInt(0), util.NewInt(1)}
	if len(erased) != 0 && erased[0] {
		inputValues = []util.Int{large}
	}
	serviceMust(t, m.set(&input, inputValues))
	var held util.Block
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
	caller := []*util.Int{&large}
	for i := range m.values(&input) {
		caller = append(caller, &m.values(&input)[i])
	}
	primeTrialPhysicalAuditAll(t, q, s, []*UPolynomialFactors{fs}, []util.Block{held}, caller...)
}
