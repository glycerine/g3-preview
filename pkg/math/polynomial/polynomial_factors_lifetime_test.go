package polynomial

// Source: src/math/polynomial/polynomial.cpp:1617-1638,2396-2440;
// src/util/mpz.cpp; src/util/memory_manager.cpp.

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type polynomialFactorsCallbackState struct {
	target                                Polynomial
	source, foreign                       unumeral.Int
	calls                                 uint32
	power                                 bool
	foreignFactory                        bool
	foreignScalar                         unumeral.Int
	guardTarget                           *PolynomialFactors
	guardSource, guardCleanup, guardClose error
	mutate                                bool
	mutationPolynomial                    Polynomial
	push, setDegree, setConstant          error
	mutationUnchanged                     bool
}

var polynomialFactorsCallback *polynomialFactorsCallbackState

func polynomialFactorsDeletionCopy(h *PolynomialDeletionHandler, m *MultivariatePolynomialCore, p Polynomial) error {
	s := polynomialFactorsCallback
	if s != nil && s.power && s.target == (Polynomial{}) && m.powerLocal.active && p == m.powerLocal.result.value {
		s.target = p
		m.integer.Budget().FailAfter(0)
	}
	if s != nil && p == s.target {
		s.calls++
		if s.guardTarget != nil {
			s.guardSource = s.guardTarget.SourceFinalize()
			s.guardCleanup = s.guardTarget.Cleanup()
			s.guardClose = s.guardTarget.CloseChecked()
			if s.mutate {
				f := s.guardTarget
				n, total, degree, constant, body := len(f.factors), f.total, f.degrees[1], f.constant, f.bodyCause
				s.push = s.guardTarget.Push(s.mutationPolynomial, 9)
				s.setDegree = s.guardTarget.SetDegree(1, 7)
				value := unumeral.NewInt(23)
				s.setConstant = s.guardTarget.SetConstant(&value)
				s.mutationUnchanged = len(f.factors) == n && f.total == total && f.degrees[1] == degree && f.constant == constant && f.bodyCause == body
			}
		}
		if s.foreignFactory {
			m.integer.Budget().FailAfter(0)
			_, e := m.MkConstantTake(&s.foreignScalar)
			return e
		}
		return m.integer.Copy(&s.foreign, &s.source)
	}
	return nil
}
func TestPolynomialFactorsResetActualConsumedDrop(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, nil)
	a := unumeral.NewInt(17)
	q, e := m.core.MkConstantTake(&a)
	serviceMust(t, e)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	serviceMust(t, f.Push(q, 2))
	state := &polynomialFactorsCallbackState{target: q}
	digits := make([]uint32, 80)
	digits[79] = 1
	serviceMust(t, m.core.integer.SetLimbs(&state.source, digits))
	serviceMust(t, m.core.integer.Copy(&state.foreign, &m.core.integer.Two64))
	polynomialFactorsCallback = state
	defer func() { polynomialFactorsCallback = nil }()
	h, e := NewPolynomialDeletionHandler(m.core.integer.Budget(), polynomialFactorsDeletionCopy)
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(h))
	// A separate real fresh SET failure establishes a NULL source pointer.
	// Its exact pending allocation is cleaned as fixture setup; the value token
	// remains unread until the later destructor reaches the constant SET site.
	m.core.integer.Budget().FailAfter(0)
	constantCause := f.SetConstant(&state.source)
	if !numericPhysicalReservationRefusal(constantCause) || (f.constant).Cell != m.core.integer.FailedNullRef {
		t.Fatal("actual fresh constant failure", constantCause, f.constant)
	}
	polynomialFactorsAuditExtra(t, f, []Polynomial{p, q}, []*unumeral.Int{&state.source, &state.foreign}, h.header)
	m.core.integer.Budget().FailAfter(-1)
	serviceMust(t, f.numeric.cleanup(m.core.integer))
	constantBefore := f.constant
	m.core.integer.Budget().FailAfter(0)
	e = f.SourceFinalize()
	if !numericPhysicalReservationRefusal(e) || state.calls != 1 || f.phases[0] != factorLastDrop || (state.foreign).Cell != m.core.integer.FailedRef || f.constant != constantBefore {
		t.Fatal("reached real Copy refusal after source last DEC", e, state.calls, f.phases, state.foreign)
	}
	// Caller-created handler roots are explicitly named; factors never owns them.
	polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p, q}, []*unumeral.Int{&f.constant, &f.local, &state.source, &state.foreign}, f.header+h.header, nil)
	cause := f.sourceCause
	before := m.core.memory.State()
	used := m.core.integer.Budget().Used()
	if again := f.SourceFinalize(); again != cause || m.core.memory.State() != before || m.core.integer.Budget().Used() != used || state.calls != 1 {
		t.Fatal("source sticky cursor replayed DEC/callback", again, cause)
	}
	m.core.integer.Budget().FailAfter(-1)
	serviceMust(t, f.CloseChecked())
	if state.calls != 1 || f.sourceCause != cause || f.sourceStage != 0 {
		t.Fatal("owning abort replayed source or repaired history")
	}
	if (state.foreign).Cell != m.core.integer.FailedRef {
		t.Fatal("parent cleanup swept foreign callback numeral")
	}
	serviceMust(t, m.core.integer.Del(&state.foreign))
	serviceMust(t, m.core.integer.Del(&state.source))
	serviceMust(t, m.core.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}
func TestPolynomialFactorsResetActualTLSConsumedConstant(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, p, _ := polynomialEvalFixtureWithRationals(t, q, nil)
	// Ordinary source create/INC/DEC establishes real freelist capacity before
	// the guarded signed-host-domain recipe; no free-OOM is manufactured.
	var warm [6]Polynomial
	for i := range warm {
		a := unumeral.NewInt(int32(7 + i))
		warm[i], e = m.core.MkConstantTake(&a)
		serviceMust(t, e)
		serviceMust(t, m.core.IncRef(warm[i]))
	}
	for _, w := range warm {
		serviceMust(t, m.core.DecRef(w))
	}
	x, e := m.core.monomials.MkVar()
	serviceMust(t, e)
	mon, e := m.core.monomials.MkVariable(x, 7)
	serviceMust(t, e)
	serviceMust(t, m.core.monomials.IncRef(mon))
	serviceMust(t, m.core.monomials.DecRef(mon))

	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	digits := make([]uint32, 30000)
	digits[29999] = 1
	serviceMust(t, q.Integers().SetLimbs(&f.constant, digits))
	polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
	contextTLSNearMinimum(t, rt)
	e = f.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || f.sourceStage != 0 || !f.sourceStarted {
		t.Fatal("real consumed constant physical free stopped SET1", e, f.sourceStage)
	}
	cause := f.sourceCause
	state := rt.State()
	stats := rt.Stats()
	polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
	if again := f.SourceFinalize(); again != cause || state != rt.State() || stats != rt.Stats() {
		t.Fatal("source retried consumed physical leaf", again)
	}
	contextTLSFinish(t, rt, f.CloseChecked)
	if f.sourceCause != cause || f.sourceStage != 0 {
		t.Fatal("owning cleanup changed source frontier")
	}
	contextTLSFinish(t, rt, func() error { return m.core.CleanupValue(&p) })
	contextTLSFinish(t, rt, m.CloseChecked)
	contextTLSFinish(t, rt, q.CloseChecked)
	serviceMust(t, rt.Close())
	if budget.Used() != 0 {
		t.Fatal("actual graph retained", budget.Used())
	}
}
func TestPolynomialFactorsFreshNullVersusDestroyedConstant(t *testing.T) {
	for _, oldBig := range []bool{false, true} {
		t.Run(strconv.FormatBool(oldBig), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, nil)
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			if oldBig {
				serviceMust(t, f.SetConstant(&m.core.integer.Two64))
			}
			a := unumeral.Int{}
			digits := make([]uint32, 80)
			digits[79] = 1
			serviceMust(t, m.core.integer.SetLimbs(&a, digits))
			defer func() { serviceMust(t, m.core.integer.Del(&a)) }()
			// Diagnostic archives use native storage. Refuse the actual next
			// numerical allocation for both fresh and previously BIG fields.
			m.core.integer.Budget().FailAfter(0)
			e = f.SetConstant(&a)
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal("genuine reached replacement refusal", e)
			}
			expected := m.core.integer.FailedNullRef
			if oldBig {
				expected = m.core.integer.FailedRef
			}
			if (f.constant).Cell != expected {
				t.Fatal("actual setter-recorded pointer state", f.constant)
			}
			polynomialFactorsAudit(t, f, []Polynomial{p}, []*unumeral.Int{&a})
			m.core.integer.Budget().FailAfter(-1)
			e = f.SourceFinalize()
			if oldBig {
				if e == nil || f.sourceStage != 0 {
					t.Fatal("destroyed pointer must guard actual reset SET/free", e)
				}
			} else {
				serviceMust(t, e)
				if f.sourceStage != 4 {
					t.Fatal("fresh NULL defined SET/dtor suffix", f.sourceStage)
				}
			}
			serviceMust(t, f.CloseChecked())
		})
	}
}

func TestPolynomialFactorsYoungerPowerStop(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	serviceMust(t, f.Push(p, 3))
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	state := &polynomialFactorsCallbackState{power: true, guardTarget: f}
	digits := make([]uint32, 80)
	digits[79] = 1
	serviceMust(t, m.core.integer.SetLimbs(&state.source, digits))
	serviceMust(t, m.core.integer.Copy(&state.foreign, &m.core.integer.Two64))
	polynomialFactorsCallback = state
	defer func() { polynomialFactorsCallback = nil }()
	h, e := NewPolynomialDeletionHandler(m.core.integer.Budget(), polynomialFactorsDeletionCopy)
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(h))
	cause := f.Multiply(r)
	if !numericPhysicalReservationRefusal(cause) || state.calls != 1 || !f.currentActive || !m.core.powerLocal.active || f.sourceCause == nil {
		t.Fatal("younger actual source stop", cause, state.calls, f.currentActive, m.core.powerLocal.active)
	}
	for _, e := range []error{state.guardSource, state.guardCleanup, state.guardClose} {
		if !errors.Is(e, ErrPolynomialPending) {
			t.Fatal("actual power deletion callback crossed caller lifetime", e)
		}
	}
	if f.closing || f.closed || f.sourceStarted {
		t.Fatal("refused power callback changed source destructor progress")
	}
	polynomialFactorsAuditExtra(t, f, []Polynomial{p, state.target}, []*unumeral.Int{&state.source, &state.foreign}, h.header, r)
	sourceCause := f.sourceCause
	current := f.current.value
	if e = f.finishCurrent(nil); e == nil || f.current.value != current || state.calls != 1 {
		t.Fatal("older source current drained after younger stop", e)
	}
	m.core.integer.Budget().FailAfter(-1)
	serviceMust(t, f.Cleanup())
	if state.calls != 1 || f.sourceCause != sourceCause || (state.foreign).Cell != m.core.integer.FailedRef {
		t.Fatal("cleanup replay/foreign/history")
	}
	polynomialFactorsAuditExtra(t, f, []Polynomial{p}, []*unumeral.Int{&state.source, &state.foreign}, h.header, r)
	serviceMust(t, r.Close())
	serviceMust(t, f.CloseChecked())
	serviceMust(t, m.core.integer.Del(&state.source))
	serviceMust(t, m.core.integer.Del(&state.foreign))
	serviceMust(t, m.core.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}

func TestPolynomialFactorsCallbackForeignFactoryAndIncoming(t *testing.T) {
	q, budget := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: q.Integers().Memory, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	control := &polynomialEvalControl{limit: 100000}
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: control.checkpoint, Cooperation: control.cooperate}})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	p, e := m.core.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	a := unumeral.NewInt(17)
	old, e := m.core.MkConstantTake(&a)
	serviceMust(t, e)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	serviceMust(t, f.Push(p, 1))
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	serviceMust(t, r.Assign(old))
	state := &polynomialFactorsCallbackState{target: old, foreignFactory: true, foreignScalar: unumeral.NewInt(23)}
	polynomialFactorsCallback = state
	defer func() { polynomialFactorsCallback = nil }()
	h, e := NewPolynomialDeletionHandler(budget, polynomialFactorsDeletionCopy)
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(h))
	cause := f.Multiply(r)
	if !numericPhysicalReservationRefusal(cause) || state.calls != 1 || r.root.incoming != p || !m.core.singletonReference {
		t.Fatal("actual callback foreign acquisition after output INC", cause, state.calls, r.root.incoming)
	}
	pending := m.core.pending.generation
	singleton := m.core.singletonGeneration
	polynomialFactorsAuditExtra(t, f, []Polynomial{p, old}, []*unumeral.Int{&state.foreignScalar}, h.header, r)
	budget.FailAfter(-1)
	serviceMust(t, f.Cleanup())
	if r.root.incoming != p || m.core.pending.generation != pending || !m.core.singletonReference || m.core.singletonGeneration != singleton || state.calls != 1 {
		t.Fatal("parent cleanup consumed foreign factory or caller incoming")
	}
	polynomialFactorsAuditExtra(t, f, []Polynomial{p, old}, []*unumeral.Int{&state.foreignScalar}, h.header, r)
	// Only the caller of the callback-created factory aborts its continuation.
	serviceMust(t, m.core.CleanupConstruction())
	serviceMust(t, r.Close())
	serviceMust(t, f.CloseChecked())
	if state.calls != 1 {
		t.Fatal("older callback replayed")
	}
	serviceMust(t, m.core.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, m.core.DecRef(p))
	serviceMust(t, m.CloseChecked())
}
