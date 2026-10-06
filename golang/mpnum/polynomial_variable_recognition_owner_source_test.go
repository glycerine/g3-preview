// Source: src/math/polynomial/polynomial.cpp:6978-6980;
// src/util/mpz.cpp:142-161. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

type variableRecognitionOwnerCallback struct {
	full       *MultivariatePolynomialManager
	q          *RationalManager
	called     bool
	mode       string
	p          Polynomial
	out        *Int
	v          *uint32
	result     error
	changed    bool
	standalone bool
	pn         bool
}

var variableRecognitionOwnerCallbackCurrent *variableRecognitionOwnerCallback

func variableRecognitionOwnerTransport(f memory.Failure) error {
	s := variableRecognitionOwnerCallbackCurrent
	if s != nil && !s.called {
		s.called = true
		m := s.full.core.integer
		statsBefore, usedBefore := s.full.core.memory.Stats(), m.budget.Used()
		qclosing, qfinalizing, qfinalized, qcause := s.q.closing, s.q.sourceFinalizing, s.q.sourceFinalized, s.q.cleanupCause
		closed, header, tmp, arg := m.closed, m.headerBytes, m.tmp, m.arg
		stage, started, finished, cause := m.sourceFinalizeStage, m.sourceFinalizing, m.sourceFinalized, m.cleanupCause
		switch s.mode {
		case "IntegerSource":
			s.result = m.SourceFinalize()
		case "IntegerClose":
			s.result = m.CloseChecked()
		case "RationalSource":
			s.result = s.q.SourceFinalize()
		case "RationalClose":
			s.result = s.q.CloseChecked()
		}
		if !s.standalone {
			c := s.full.core
			before := c.variableNumeral
			stats := c.memory.Stats()
			used := c.integer.budget.Used()
			value, variable := *s.out, *s.v
			for _, e := range []error{s.full.SourceFinalize(), s.full.CloseChecked(), c.SourceFinalizePrefix(), c.CloseChecked(), s.full.CleanupVariableNumeral()} {
				if !errors.Is(e, ErrPolynomialPending) {
					s.changed = true
				}
			}
			_, again := s.full.IsVariableNumeral(s.p, s.v, s.out)
			if !errors.Is(again, ErrPolynomialPending) || before != c.variableNumeral || stats != c.memory.Stats() || used != c.integer.budget.Used() || value != *s.out || variable != *s.v {
				s.changed = true
			}
		}
		s.changed = s.changed || closed != m.closed || header != m.headerBytes || tmp != m.tmp || arg != m.arg || (s.pn && !m.normalizationExecuting) || (!s.pn && !m.copyAllocation.executing) || statsBefore != s.full.core.memory.Stats() || usedBefore != m.budget.Used() || qclosing != s.q.closing || qfinalizing != s.q.sourceFinalizing || qfinalized != s.q.sourceFinalized || qcause != s.q.cleanupCause || stage != m.sourceFinalizeStage || started != m.sourceFinalizing || finished != m.sourceFinalized || cause != m.cleanupCause
	}
	return f
}
func TestPolynomialVariableRecognitionExecutingIntegerOwnerRefusesFinalize(t *testing.T) {
	for _, mode := range []string{"IntegerSource", "IntegerClose", "RationalSource", "RationalClose"} {
		t.Run(mode, func(t *testing.T) {
			variableRecognitionExecutingOwner(t, mode, false, false)
			variableRecognitionExecutingOwner(t, mode, true, false)
		})
	}
}
func variableRecognitionExecutingOwner(t *testing.T, mode string, field, standalone bool) {
	budget := arena.NewBudget(1 << 24)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: budget, Transport: variableRecognitionOwnerTransport})
	serviceMust(t, e)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 1, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 1, Budget: budget}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	words := make([]uint32, 500)
	words[0], words[499] = 17, 1
	var big Int
	serviceMust(t, c.integer.SetLimbs(&big, words))
	mon, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{NewInt(1), big}, []Monomial{mon, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	out := NewInt(9)
	v := uint32(77)
	if field {
		serviceMust(t, c.integer.Copy(&out, &big))
		serviceMust(t, c.modular.SetZPUint64(5))
	}
	state := variableRecognitionOwnerCallback{full: full, q: q, mode: mode, p: p, out: &out, v: &v, standalone: standalone, pn: field}
	variableRecognitionOwnerCallbackCurrent = &state
	serviceMust(t, rt.SetMaxSize(1))
	var body error
	if standalone {
		body = c.modular.PNormalize(&out)
	} else {
		_, body = full.IsVariableNumeral(p, &v, &out)
	}
	variableRecognitionOwnerCallbackCurrent = nil
	// The callback result is recorded before recovery, without using it as authority.
	called, result, changed := state.called, state.result, state.changed
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*Int{&out, &big}, 0)
	if !called || body == nil || !errors.Is(result, ErrPolynomialPending) || changed {
		t.Fatalf("actual reached SET executing owner: called=%v body=%v result=%v changed=%v", called, body, result, changed)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, full.CleanupVariableNumeral())
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.integer.Del(&big))
	serviceMust(t, c.DecRef(p))
	serviceMust(t, full.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, rt.Close())
	if !called || body == nil || !errors.Is(result, ErrPolynomialPending) || changed {
		t.Fatalf("actual reached SET executing owner: called=%v body=%v result=%v changed=%v", called, body, result, changed)
	}
	if budget.Used() != 0 {
		t.Fatal("owner graph", budget.Used())
	}
}

// The public nil-receipt PN route shares the same literal engine execution guard.
func TestIntegerPNormalizeNilReturningTransportOwnerRefusal(t *testing.T) {
	for _, mode := range []string{"IntegerSource", "IntegerClose", "RationalSource", "RationalClose"} {
		t.Run(mode, func(t *testing.T) { variableRecognitionExecutingOwner(t, mode, true, true) })
	}
}
