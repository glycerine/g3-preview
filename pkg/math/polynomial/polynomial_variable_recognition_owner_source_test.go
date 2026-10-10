// Source: src/math/polynomial/polynomial.cpp:6978-6980;
// src/util/mpz.cpp:142-161.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type variableRecognitionOwnerCallback struct {
	full       *MultivariatePolynomialManager
	q          *unumeral.RationalManager
	called     bool
	mode       string
	p          Polynomial
	out        *unumeral.Int
	v          *uint32
	result     error
	changed    bool
	standalone bool
	pn         bool
}

var variableRecognitionOwnerCallbackCurrent *variableRecognitionOwnerCallback

func variableRecognitionOwnerTransport(f umemory.Failure) error {
	s := variableRecognitionOwnerCallbackCurrent
	if s != nil && !s.called {
		s.called = true
		m := s.full.core.integer
		statsBefore, usedBefore := s.full.core.memory.Stats(), m.Budget().Used()
		qclosing, qfinalizing, qfinalized, qcause := s.q.Closing, s.q.SourceFinalizing, s.q.SourceFinalized, s.q.CleanupCauseField
		closed, header, tmp, arg := m.Closed(), m.HeaderBytes, m.Tmp, m.Arg
		stage, started, finished, cause := m.SourceFinalizeStage, m.SourceFinalizing, m.SourceFinalized, m.CleanupCauseField
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
			used := c.integer.Budget().Used()
			value, variable := *s.out, *s.v
			for _, e := range []error{s.full.SourceFinalize(), s.full.CloseChecked(), c.SourceFinalizePrefix(), c.CloseChecked(), s.full.CleanupVariableNumeral()} {
				if !errors.Is(e, ErrPolynomialPending) {
					s.changed = true
				}
			}
			_, again := s.full.IsVariableNumeral(s.p, s.v, s.out)
			if !errors.Is(again, ErrPolynomialPending) || before != c.variableNumeral || stats != c.memory.Stats() || used != c.integer.Budget().Used() || value != *s.out || variable != *s.v {
				s.changed = true
			}
		}
		s.changed = s.changed || closed != m.Closed() || header != m.HeaderBytes || tmp != m.Tmp || arg != m.Arg || (s.pn && !m.NormalizationExecuting) || (!s.pn && !m.CopyAllocation.Executing) || statsBefore != s.full.core.memory.Stats() || usedBefore != m.Budget().Used() || qclosing != s.q.Closing || qfinalizing != s.q.SourceFinalizing || qfinalized != s.q.SourceFinalized || qcause != s.q.CleanupCauseField || stage != m.SourceFinalizeStage || started != m.SourceFinalizing || finished != m.SourceFinalized || cause != m.CleanupCauseField
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
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, Transport: variableRecognitionOwnerTransport})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 1, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 1, Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	words := make([]uint32, 500)
	words[0], words[499] = 17, 1
	var big unumeral.Int
	serviceMust(t, c.integer.SetLimbs(&big, words))
	mon, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(1), big}, []Monomial{mon, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	out := unumeral.NewInt(9)
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
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*unumeral.Int{&out, &big}, 0)
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
