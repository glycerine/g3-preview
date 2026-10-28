package polynomial

// Source: src/math/polynomial/polynomial.cpp:6683-6725;
// src/math/polynomial/upolynomial_factorization.cpp:845-933.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestPolynomialFactorCoreActualYoungerBackendDTProtectsReferencesAndNumeral(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("literal conditional checker erased")
	}
	b := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(b)
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}, Factorization: UPolynomialFactorizationConfig{Primes: primes, FactorizationDebugTag: true}})
	serviceMust(t, e)
	var external umemory.Block
	armed := false
	full.upm.config.Checkpoint = func() bool {
		if full.factorService != nil && full.factorService.scope != nil {
			h := &full.factorService.scope.hensel
			if h.checkSource == 2 && (h.checkPower).Cell != 0 && !armed {
				armed = true
				contextTLSNearMinimum(t, rt)
				cause := rt.Deallocate(&external)
				if !errors.Is(cause, arena.ErrBounds) || external != (umemory.Block{}) {
					t.Fatal("actual signed owner release", cause)
				}
				return false
			}
		}
		return true
	}
	_, e = mm.MkVar()
	serviceMust(t, e)
	values := []unumeral.Int{unumeral.NewInt(0), unumeral.NewInt(0), unumeral.NewInt(1)}
	serviceMust(t, q.Integers().SetString(&values[0], "-387381625547900583942"))
	serviceMust(t, q.Integers().SetString(&values[1], "-129127208515966861311"))
	p, e := full.ToPolynomial(3, values, 0)
	serviceMust(t, e)
	serviceMust(t, full.core.IncRef(p))
	serviceMust(t, full.CleanupToPolynomial())
	for i := range values {
		serviceMust(t, q.Integers().Del(&values[i]))
	}
	out, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	external, e = rt.Allocate(200000)
	serviceMust(t, e)
	body := full.Factor(p, out, DefaultUPolynomialFactorParams())
	parent := full.factorRoot.frame
	receipt := parent.univariate
	s := receipt.frame
	if !armed || s == nil || !errors.Is(body, arena.ErrBounds) || !errors.Is(body, ErrUPolynomialCanceled) || s.sourceCursor != 0 || s.constructed != 3 {
		t.Fatal("actual younger callee source stop", armed, body, s)
	}
	original := s.sourceCause
	if parent.sourceCause != original || parent.sourceStarted || parent.references != 7 || !parent.numeralConstructed {
		t.Fatal("actual youngest child did not protect Factor locals", parent.sourceCause, original, parent.references)
	}
	if original == nil || s.p.value != p || s.p1.size != 3 || s.fs == nil || s.fs.closed {
		t.Fatal("older source locals were consumed")
	}
	polynomialFactorsAuditExtra(t, out, []Polynomial{p}, nil, primes.Stats().TotalBytes)
	oldP, oldFS, oldVector := s.p, s.fs, s.p1
	sourceStarted, bodyStage, memberStage := full.sourceStarted, full.bodyStage, full.memberStage
	if full.SourceFinalize() != original || s.p != oldP || s.fs != oldFS || s.p1 != oldVector || full.sourceStarted != sourceStarted || full.bodyStage != bodyStage || full.memberStage != memberStage {
		t.Fatal("older class source prefix crossed child stop")
	}
	if full.finishFactorUnivariateSource(&receipt) != original || s.sourceCursor != 0 {
		t.Fatal("source retry replayed older locals")
	}
	own := full.cleanupFactorCoreInvocation(&full.factorRoot, false)
	if own == nil || s.cleanupCause != own || parent.cleanupCause != own {
		t.Fatal("actual own release cause", own)
	}
	polynomialFactorsAuditExtra(t, out, []Polynomial{p}, nil, primes.Stats().TotalBytes)
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return full.cleanupFactorCoreInvocation(&full.factorRoot, false) })
	contextTLSFinish(t, rt, func() error { return full.cleanupFactorCoreInvocation(&full.factorRoot, true) })
	if s.sourceCause != original || s.cleanupCause != own || parent.sourceCause != original || parent.cleanupCause != own || !errors.Is(body, original) {
		t.Fatal("exact child/parent immutable history")
	}
	// A real later call reuses the backend after exact owning acknowledgment.
	// The old source/body/first-own tuple remains on its actual charged frame.
	full.upm.config.Checkpoint = func() bool { return true }
	later, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	serviceMust(t, full.Factor(p, later, DefaultUPolynomialFactorParams()))
	newReceipt := full.factorRoot
	beforeUsed, beforeOutput := b.Used(), rtStateForFactors(later)
	serviceMust(t, full.cleanupFactorCoreInvocation(&polynomialFactorCoreInvocationReceipt{full, parent, parent.invocation}, true))
	if full.factorRoot != newReceipt || beforeUsed != b.Used() || beforeOutput != rtStateForFactors(later) || parent.sourceCause != original || parent.cleanupCause != own || !errors.Is(parent.bodyCause, ErrUPolynomialCanceled) || !errors.Is(body, original) {
		t.Fatal("later actual call or stale ACK erased old diagnostics/adopted new output")
	}
	polynomialFactorsAuditOwnersExtra(t, later, []Polynomial{p}, nil, primes.Stats().TotalBytes, []umemory.Block{recovery}, []*PolynomialFactors{out})
	contextTLSFinish(t, rt, later.CloseChecked)
	contextTLSFinish(t, rt, out.CloseChecked)
	contextTLSFinish(t, rt, func() error { return full.core.DecRef(p) })
	contextTLSFinish(t, rt, full.CloseChecked)
	serviceMust(t, primes.Close())
	contextTLSFinish(t, rt, q.CloseChecked)
	contextTLSFinish(t, rt, mm.CloseChecked)
	_ = rt.Deallocate(&recovery)
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal("actual complete graph retained", b.Used())
	}
}
