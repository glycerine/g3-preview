// Source: src/math/polynomial/upolynomial.cpp:1108-1225,3120-3124;
// src/test/upolynomial.cpp:23-65,468-490 (display driver sites).
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"strconv"
	"strings"
	"testing"
)

// The driver prints these values without ENSUREs. These exact stream checks
// are new independent output assertions, not registered source assertion credit.
type upolyDisplaySink struct {
	text        strings.Builder
	literalHook func(string) error
	numeralHook func(NumeralCStringView) error
}

func (s *upolyDisplaySink) AppendInt32(v int32) error {
	s.text.WriteString(strconv.FormatInt(int64(v), 10))
	return nil
}
func (s *upolyDisplaySink) AppendCString(v string) error {
	if s.literalHook != nil {
		if e := s.literalHook(v); e != nil {
			return e
		}
	}
	s.text.WriteString(v)
	return nil
}
func (s *upolyDisplaySink) AppendNumeralCString(v NumeralCStringView) error {
	if s.numeralHook != nil {
		if e := s.numeralHook(v); e != nil {
			return e
		}
	}
	for i := uint64(0); ; i++ {
		b, e := v.Byte(i)
		if e != nil {
			return e
		}
		if b == 0 {
			return nil
		}
		s.text.WriteByte(b)
	}
}
func TestUPolynomialDisplayActualTst1DriverOutput(t *testing.T) {
	z, s := preparationPhysicalFixture(t)
	m := s.manager
	var p upolyVectorState
	serviceMust(t, m.set(&p, []Int{NewInt(18), NewInt(9), NewInt(-20), NewInt(-10), NewInt(2), NewInt(1)}))
	defer func() { serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	out := &upolyDisplaySink{}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
	if out.text.String() != "x^5 + 2 x^4 - 10 x^3 - 20 x^2 + 9 x + 18" {
		t.Fatal(out.text.String())
	}
	// Driver's coefficient traversal adds one, then trims the actual vector.
	for i := uint32(0); i < p.size; i++ {
		one := NewInt(1)
		serviceMust(t, m.modular.Add(&m.values(&p)[i], &one, &m.values(&p)[i]))
	}
	serviceMust(t, m.trim(&p))
	out = &upolyDisplaySink{}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", true))
	if out.text.String() != "2*x^5 + 3*x^4 - 9*x^3 - 19*x^2 + 10*x + 19" {
		t.Fatal(out.text.String())
	}
	primeTrialPhysicalAuditOwners(t, z, s, nil)
}
func TestUPolynomialDisplaySMT2LiteralSingleMonomialFallthrough(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	cases := []struct {
		values []Int
		want   string
	}{
		{nil, "0"},
		{[]Int{NewInt(-3)}, "(- 3)"},
		{[]Int{NewInt(0), NewInt(1)}, "x(+ x)"},
		{[]Int{NewInt(0), NewInt(0), NewInt(-1)}, "(* (- 1) (^ x 2))(+ (* (- 1) (^ x 2)))"},
		{[]Int{NewInt(5), NewInt(-1), NewInt(2)}, "(+ (* 2 (^ x 2)) (* (- 1) x) 5)"},
	}
	for _, tc := range cases {
		var p upolyVectorState
		serviceMust(t, m.set(&p, tc.values))
		out := &upolyDisplaySink{}
		serviceMust(t, m.DisplaySMT2Into(m.stateView(&p), out, "x"))
		if out.text.String() != tc.want {
			t.Fatal(tc.want, out.text.String())
		}
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
	}
}

func TestUPolynomialDisplayActualStreamOwnerReentry(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	var p upolyVectorState
	serviceMust(t, m.set(&p, []Int{NewInt(1), NewInt(1)}))
	defer func() {
		if !m.closed {
			serviceMust(t, m.reset(&p))
			m.freeBacking(&p)
		}
	}()
	called := false
	out := &upolyDisplaySink{literalHook: func(_ string) error {
		if called {
			return nil
		}
		called = true
		before := m.config.Integer.budget.Used()
		e := m.Close()
		if e != ErrUPolynomialCleanupPending || m.closed || m.closing || before != m.config.Integer.budget.Used() {
			t.Errorf("executing source caller owner entry mutated: %v closed=%v closing=%v", e, m.closed, m.closing)
		}
		return nil
	}}
	e := m.DisplayInto(m.stateView(&p), out, "x", false)
	if e != nil || !called {
		t.Fatal("actual display stream route", e, called)
	}
}

func TestUPolynomialDisplayActualLowerOwnerReentry(t *testing.T) {
	for _, mode := range []string{"integer-source", "integer-own", "rational-source", "rational-own"} {
		t.Run(mode, func(t *testing.T) {
			r, s := preparationPhysicalFixture(t)
			m := s.manager
			var p upolyVectorState
			serviceMust(t, m.set(&p, []Int{NewInt(1), NewInt(1)}))
			defer func() {
				if !r.z.closed {
					serviceMust(t, m.reset(&p))
					m.freeBacking(&p)
				}
			}()
			called := false
			out := &upolyDisplaySink{literalHook: func(string) error {
				if called {
					return nil
				}
				called = true
				before := r.z.budget.Used()
				var e error
				switch mode {
				case "integer-source":
					e = r.z.SourceFinalize()
				case "integer-own":
					e = r.z.CloseChecked()
				case "rational-source":
					e = r.SourceFinalize()
				case "rational-own":
					e = r.CloseChecked()
				}
				if e != ErrPolynomialPending || r.z.closed || r.z.sourceFinalizing || r.closed || r.sourceFinalizing || before != r.z.budget.Used() {
					t.Errorf("executing borrowed stream retired lower owner: %v", e)
				}
				return nil
			}}
			e := m.DisplayInto(m.stateView(&p), out, "x", false)
			if e != nil || !called {
				t.Fatal("actual lower callback route", e, called)
			}
		})
	}
}

func TestUPolynomialDisplayBIGStreamFailureLiteralLifetime(t *testing.T) {
	for _, smt := range []bool{false, true} {
		t.Run(strconv.FormatBool(smt), func(t *testing.T) {
			r, s := preparationPhysicalFixture(t)
			m := s.manager
			var a Int
			serviceMust(t, r.z.SetLimbs(&a, []uint32{7, 0, 0, 1}))
			serviceMust(t, r.z.Neg(&a))
			var p upolyVectorState
			serviceMust(t, m.set(&p, []Int{a}))
			serviceMust(t, r.z.Del(&a))
			defer func() { serviceMust(t, m.CleanupDisplay()); serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
			marker := errors.New("actual ostream refused decimal borrow")
			var borrowed NumeralCStringView
			out := &upolyDisplaySink{numeralHook: func(v NumeralCStringView) error { borrowed = v; return marker }}
			var e error
			if smt {
				e = m.DisplaySMT2Into(m.stateView(&p), out, "x")
			} else {
				e = m.DisplayInto(m.stateView(&p), out, "x", false)
			}
			if !errors.Is(e, marker) || !errors.Is(m.display.bodyCause, marker) {
				t.Fatal("immutable body cause", e)
			}
			if m.display.constructed != smt {
				t.Fatal("source scoped NUM vs RAW declaration lifetime", smt, m.display.constructed)
			}
			if _, e = borrowed.Byte(0); e == nil {
				t.Fatal("decimal buffer borrow survived lexical insertion")
			}
			roots := []*Int{&m.display.numeral}
			for i := uint32(0); i < p.size; i++ {
				roots = append(roots, &m.values(&p)[i])
			}
			primeTrialPhysicalAuditOwners(t, r, s, nil, roots...)
			serviceMust(t, m.CleanupDisplay())
			if m.display.active || !errors.Is(m.display.bodyCause, marker) {
				t.Fatal("own ack erased diagnostic or retained exact NUM")
			}
			out = &upolyDisplaySink{}
			serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
			if out.text.String() != "-79228162514264337593543950343" {
				t.Fatal(out.text.String())
			}
		})
	}
}

func TestUPolynomialDisplayEveryBIGReservationPrefix(t *testing.T) {
	for _, smt := range []bool{false, true} {
		t.Run(strconv.FormatBool(smt), func(t *testing.T) {
			complete := false
			for ordinal := int64(0); ordinal < 96; ordinal++ {
				done := false
				t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
					r, s := preparationPhysicalFixture(t)
					m := s.manager
					var a Int
					serviceMust(t, r.z.SetLimbs(&a, []uint32{7, 0, 0, 1}))
					serviceMust(t, r.z.Neg(&a))
					var p upolyVectorState
					serviceMust(t, m.set(&p, []Int{a}))
					serviceMust(t, r.z.Del(&a))
					r.z.budget.FailAfter(ordinal)
					out := &upolyDisplaySink{}
					var e error
					if smt {
						e = m.DisplaySMT2Into(m.stateView(&p), out, "x")
					} else {
						e = m.DisplayInto(m.stateView(&p), out, "x", false)
					}
					roots := []*Int{&m.display.numeral}
					for i := uint32(0); i < p.size; i++ {
						roots = append(roots, &m.values(&p)[i])
					}
					primeTrialPhysicalAuditOwners(t, r, s, nil, roots...)
					if e == nil {
						done = true
					}
					// Source return and immediate named graph precede any fault lift. Owning
					// continuation is exact to the actual NUM/RAW and tracked insertion child.
					serviceMust(t, m.CleanupDisplay())
					primeTrialPhysicalAuditOwners(t, r, s, nil, roots...)
					r.z.budget.FailAfter(-1)
					serviceMust(t, m.reset(&p))
					m.freeBacking(&p)
				})
				if done {
					complete = true
					break
				}
			}
			if !complete {
				t.Fatal("actual display allocation horizon not reached")
			}
		})
	}
}

// The C++ Sturm driver prints the computed sequence without ENSUREs. The
// independently checked leading polynomial and row separators below are new
// output assertions; the actual Sturm engine supplies the remaining rows.
func TestUPolynomialDisplayActualSturmSequenceDriver(t *testing.T) {
	_, service := preparationPhysicalFixture(t)
	m := service.manager
	var p upolyVectorState
	serviceMust(t, m.set(&p, []Int{NewInt(8), NewInt(2), NewInt(8), NewInt(10), NewInt(10), NewInt(0), NewInt(1), NewInt(0), NewInt(1), NewInt(3), NewInt(7)}))
	defer func() { serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	seq, e := m.NewSequence()
	serviceMust(t, e)
	defer func() { serviceMust(t, seq.Close()) }()
	serviceMust(t, m.SturmSequence(m.stateView(&p), seq))
	n, e := seq.Size()
	serviceMust(t, e)
	out := &upolyDisplaySink{literalHook: func(text string) error {
		if text == "\n" {
			before := m.config.Integer.budget.Used()
			if e := seq.Close(); e != ErrUPolynomialCleanupPending || seq.closed || before != m.config.Integer.budget.Used() {
				t.Error("borrowed source sequence retired", e)
			}
			if e := m.config.Integer.SourceFinalize(); e != ErrPolynomialPending {
				t.Error("sequence newline retired numerical owner", e)
			}
		}
		return nil
	}}
	serviceMust(t, m.DisplaySequenceInto(seq, out, "x"))
	lines := strings.Split(out.text.String(), "\n")
	if len(lines) != int(n)+1 || lines[len(lines)-1] != "" || lines[0] != "7 x^10 + 3 x^9 + x^8 + x^6 + 10 x^4 + 10 x^3 + 8 x^2 + 2 x + 8" {
		t.Fatal("actual printed driver rows", n, out.text.String())
	}
}

func TestUPolynomialDisplayContextCopiedStaleCallbackGuards(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	var p upolyVectorState
	serviceMust(t, m.set(&p, []Int{NewInt(0), NewInt(1)}))
	defer func() { serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	var old integerAllocationInvocationContext
	called := false
	out := &upolyDisplaySink{literalHook: func(string) error {
		if called {
			return nil
		}
		called = true
		old = m.display.context()
		dst := NewInt(9)
		src := NewInt(1)
		before := m.config.Integer.budget.Used()
		if e := m.modular.copyWithDisplayContext(&dst, &src, old); e != ErrUPolynomialOwner || dst != NewInt(9) || before != m.config.Integer.budget.Used() {
			t.Fatal("callback adopted display allocation context", e)
		}
		copied := m.display
		c := old
		c.upolyDisplay = &copied
		if c.upolyDisplayFrame(m.config.Integer) != nil {
			t.Fatal("copied frame acquired authority")
		}
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
	called = false
	out = &upolyDisplaySink{literalHook: func(string) error {
		if called {
			return nil
		}
		called = true
		if old.upolyDisplayFrame(m.config.Integer) != nil {
			t.Fatal("old captured nonce retargeted to later call")
		}
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
}

func TestUPolynomialDisplayActualNestedBeforeBorrow(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	var p, n upolyVectorState
	serviceMust(t, m.set(&p, []Int{NewInt(0), NewInt(1)}))
	serviceMust(t, m.set(&n, []Int{NewInt(-2)}))
	defer func() { serviceMust(t, m.reset(&p)); m.freeBacking(&p); serviceMust(t, m.reset(&n)); m.freeBacking(&n) }()
	called := false
	out := &upolyDisplaySink{literalHook: func(string) error {
		if called {
			return nil
		}
		called = true
		inner := &upolyDisplaySink{}
		e := m.DisplaySMT2Into(m.stateView(&n), inner, "y")
		if e != nil || inner.text.String() != "(- 2)" {
			t.Fatal("source-valid nested display before borrowed output", e, inner.text.String())
		}
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
	if !called || out.text.String() != "x" {
		t.Fatal(out.text.String())
	}
}

func TestUPolynomialDisplayNestedInlineErasedAllocation(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	var p, n upolyVectorState
	serviceMust(t, m.set(&p, []Int{NewInt(0), NewInt(1)}))
	serviceMust(t, m.set(&n, []Int{NewInt(-2)}))
	defer func() {
		m.config.Integer.budget.FailAfter(-1)
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
		serviceMust(t, m.reset(&n))
		m.freeBacking(&n)
	}()
	called := false
	out := &upolyDisplaySink{literalHook: func(string) error {
		if called {
			return nil
		}
		called = true
		m.config.Integer.budget.FailAfter(0)
		for i := 0; i < 3; i++ {
			inner := &upolyDisplaySink{}
			serviceMust(t, m.DisplaySMT2Into(m.stateView(&n), inner, "y"))
			if inner.text.String() != "(- 2)" {
				t.Fatal(inner.text.String())
			}
		}
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
	if m.displayFrames != nil {
		t.Fatal("erased nested route reserved persistent archival storage")
	}
}

func TestUPolynomialDisplayNestedBIGPreservesActualOuterBorrow(t *testing.T) {
	r, s := preparationPhysicalFixture(t)
	m := s.manager
	var a Int
	serviceMust(t, r.z.SetLimbs(&a, []uint32{7, 0, 0, 1}))
	var p upolyVectorState
	serviceMust(t, m.set(&p, []Int{a}))
	serviceMust(t, r.z.Del(&a))
	defer func() { serviceMust(t, m.CleanupDisplay()); serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	called := false
	var old NumeralCStringView
	out := &upolyDisplaySink{numeralHook: func(v NumeralCStringView) error {
		if called {
			return nil
		}
		called = true
		old = v
		first, e := v.Byte(0)
		serviceMust(t, e)
		inner := &upolyDisplaySink{}
		serviceMust(t, m.DisplaySMT2Into(m.stateView(&p), inner, "y"))
		if inner.text.String() != "79228162514264337593543950343" {
			t.Fatal(inner.text.String())
		}
		after, e := v.Byte(0)
		serviceMust(t, e)
		if first != after {
			t.Fatal("nested lexical char buffer overwrote outer borrow")
		}
		primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0])
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
	if !called || out.text.String() != "79228162514264337593543950343" {
		t.Fatal(out.text.String())
	}
	if _, e := old.Byte(0); e == nil {
		t.Fatal("outer char borrow survived actual lexical DT")
	}
}

func TestUPolynomialDisplayNestedInlinePreservesActualOuterBorrow(t *testing.T) {
	r, s := preparationPhysicalFixture(t)
	m := s.manager
	var a Int
	serviceMust(t, r.z.SetLimbs(&a, []uint32{7, 0, 0, 1}))
	var p, n upolyVectorState
	serviceMust(t, m.set(&p, []Int{a}))
	serviceMust(t, r.z.Del(&a))
	serviceMust(t, m.set(&n, []Int{NewInt(-2)}))
	defer func() {
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
		serviceMust(t, m.reset(&n))
		m.freeBacking(&n)
	}()
	out := &upolyDisplaySink{numeralHook: func(v NumeralCStringView) error {
		first, e := v.Byte(0)
		serviceMust(t, e)
		inner := &upolyDisplaySink{}
		serviceMust(t, m.DisplaySMT2Into(m.stateView(&n), inner, "y"))
		if inner.text.String() != "(- 2)" {
			t.Fatal(inner.text.String())
		}
		after, e := v.Byte(0)
		serviceMust(t, e)
		if first != after {
			t.Fatal("inline nested call invalidated real outer CString")
		}
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
}
func TestUPolynomialDisplayNestedAllocationFailureExactRetirement(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 96; ordinal++ {
		done := false
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			r, s := preparationPhysicalFixture(t)
			m := s.manager
			var a Int
			serviceMust(t, r.z.SetLimbs(&a, []uint32{7, 0, 0, 1}))
			serviceMust(t, r.z.Neg(&a))
			var p, n upolyVectorState
			serviceMust(t, m.set(&p, []Int{NewInt(0), NewInt(1)}))
			serviceMust(t, m.set(&n, []Int{a}))
			serviceMust(t, r.z.Del(&a))
			var nested error
			called := false
			out := &upolyDisplaySink{literalHook: func(string) error {
				if called {
					return nil
				}
				called = true
				r.z.budget.FailAfter(ordinal)
				nested = m.DisplaySMT2Into(m.stateView(&n), &upolyDisplaySink{}, "y")
				return nested
			}}
			cause := m.DisplayInto(m.stateView(&p), out, "x", false)
			if nested == nil {
				done = true
			} else if !errors.Is(cause, nested) {
				t.Fatal("outer return lost actual nested body cause", cause, nested)
			}
			primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0], &m.values(&p)[1], &m.values(&n)[0])
			serviceMust(t, m.CleanupDisplay())
			primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0], &m.values(&p)[1], &m.values(&n)[0])
			r.z.budget.FailAfter(-1)
			serviceMust(t, m.DisplayInto(m.stateView(&p), &upolyDisplaySink{}, "x", false))
			if nested != nil && !errors.Is(cause, nested) {
				t.Fatal("later source call mutated old returned failure")
			}
			serviceMust(t, m.reset(&p))
			m.freeBacking(&p)
			serviceMust(t, m.reset(&n))
			m.freeBacking(&n)
		})
		if done {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("nested actual allocation horizon not reached")
	}
}

func TestUPolynomialDisplayActualNaturalBorrowOwnerReentry(t *testing.T) {
	for _, mode := range []string{"source", "own"} {
		t.Run(mode, func(t *testing.T) {
			r, s := preparationPhysicalFixture(t)
			m := s.manager
			var a Int
			serviceMust(t, r.z.SetLimbs(&a, []uint32{7, 0, 0, 1}))
			var p upolyVectorState
			serviceMust(t, m.set(&p, []Int{a}))
			serviceMust(t, r.z.Del(&a))
			defer func() { serviceMust(t, m.CleanupDisplay()); serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
			called := false
			out := &upolyDisplaySink{numeralHook: func(v NumeralCStringView) error {
				called = true
				n := r.z.natural
				before := r.z.budget.Used()
				first, e := v.Byte(0)
				serviceMust(t, e)
				if mode == "source" {
					e = n.SourceFinalize()
				} else {
					e = n.CloseChecked()
				}
				if e == nil || n.closed || n.closing || n.sourceFinalizing || before != r.z.budget.Used() {
					t.Errorf("live lexical borrowed character scope retired: %v", e)
				}
				after, e := v.Byte(0)
				if e != nil || first != after {
					t.Errorf("actual live borrow changed %v", e)
				}
				return nil
			}}
			e := m.DisplayInto(m.stateView(&p), out, "x", false)
			if e != nil || !called {
				t.Fatal("actual natural callback route", e, called)
			}
		})
	}
}

// Signed-host TLS guard fixture: all boundary changes use public Runtime
// Allocate/Reallocate/free actions. This is not an ordinary malloc-fault claim.
func TestUPolynomialDisplayActualStoppedCharacterSuffix(t *testing.T) {
	for _, bodyFails := range []bool{false, true} {
		t.Run(strconv.FormatBool(bodyFails), func(t *testing.T) {
			b := arena.NewBudget(1 << 27)
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Cells.PageSize = 1
			cfg.Memory = rt
			cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
			r, e := NewRationalManager(RationalConfig{Integers: cfg})
			serviceMust(t, e)
			m, e := NewUPolynomialManager(UPolynomialConfig{Integer: r.z, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
			serviceMust(t, e)
			primes, e := NewPrimeGenerator(b)
			serviceMust(t, e)
			s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
			serviceMust(t, e)
			var p upolyVectorState
			var lane memory.Block
			defer func() {
				b.FailAfter(-1)
				contextTLSFinish(t, rt, m.CleanupDisplay)
				if lane != (memory.Block{}) {
					contextTLSFinish(t, rt, func() error {
						if lane == (memory.Block{}) {
							return nil
						}
						return rt.Deallocate(&lane)
					})
				}
				contextTLSFinish(t, rt, func() error { return m.reset(&p) })
				m.freeBacking(&p)
				contextTLSFinish(t, rt, s.Close)
				contextTLSFinish(t, rt, primes.Close)
				contextTLSFinish(t, rt, m.Close)
				contextTLSFinish(t, rt, r.CloseChecked)
				contextTLSFinish(t, rt, rt.Close)
				if b.Used() != 0 {
					t.Fatal("actual display whole refund", b.Used())
				}
			}()
			digits := make([]uint32, 94)
			digits[len(digits)-1] = 1
			var a Int
			serviceMust(t, r.z.SetLimbs(&a, digits))
			serviceMust(t, m.set(&p, []Int{a}))
			serviceMust(t, r.z.Del(&a))
			marker := errors.New("actual ostream failed before character DT")
			var older numericCellRef
			var characterToken memory.Block
			out := &upolyDisplaySink{numeralHook: func(v NumeralCStringView) error {
				older = m.display.numeral.cell
				child := m.display.insertion.character
				if child == nil || !child.completed || child.buffer.raw == (memory.Block{}) {
					t.Fatal("actual heap character constructor not reached")
				}
				characterToken = child.buffer.raw
				if _, e := rt.Size(characterToken); e != nil {
					t.Fatal("actual character raw allocation", e)
				}
				if _, e := v.Byte(0); e != nil {
					t.Fatal(e)
				}
				primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0])
				contextTLSNearMinimum(t, rt)
				// Arm the real local TLS lane just above its release synchronization
				// threshold; this valid 94-limb heap character DT then reaches the guard.
				lane, e = rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-100006)
				if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("public signed-host prephysical realloc guard", e)
				}
				if rt.State().LocalSize != -99990 {
					t.Fatal("actual bounded local lane", rt.State())
				}
				if bodyFails {
					return marker
				}
				return nil
			}}
			returned := m.DisplayInto(m.stateView(&p), out, "x", false)
			if !errors.Is(returned, arena.ErrBounds) || (bodyFails && !errors.Is(returned, marker)) || !m.display.constructed || m.display.numeral.cell != older || older == 0 {
				t.Fatal("actual younger character DT must protect older NUM", returned, m.display.constructed)
			}
			if m.display.insertion.character.buffer.raw != (memory.Block{}) || m.display.insertion.character.cleanupCause != arena.ErrBounds {
				t.Fatal("character DT did not reach physical release cursor")
			}
			if _, e := rt.Size(characterToken); e == nil {
				t.Fatal("character physical token survived actual reached DT")
			}
			if c := r.z.cell(older); c.released || c.cleanupPending {
				t.Fatal("older NUM release cursor advanced after younger stop")
			}
			cause := m.display.sourceCause
			state := rt.State()
			used := b.Used()
			cell := m.display.numeral.cell
			if e = m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) || m.sourceFinalizing || rt.State() != state || b.Used() != used || m.display.sourceCause != cause || m.display.numeral.cell != cell {
				t.Fatal("source retry replayed older unread NUM/private members", e)
			}
			primeTrialPhysicalAuditExternal(t, r, s, []memory.Block{lane}, &m.values(&p)[0])
			contextTLSFinish(t, rt, m.CleanupDisplay)
			if m.display.active || m.display.sourceCause != cause {
				t.Fatal("own cleanup changed stopped source history")
			}
			var inline upolyVectorState
			serviceMust(t, m.set(&inline, []Int{NewInt(1)}))
			serviceMust(t, m.DisplayInto(m.stateView(&inline), &upolyDisplaySink{}, "x", false))
			serviceMust(t, m.reset(&inline))
			m.freeBacking(&inline)
			if !errors.Is(returned, arena.ErrBounds) || (bodyFails && !errors.Is(returned, marker)) {
				t.Fatal("later invocation changed original returned causes")
			}
		})
	}
}

func TestUPolynomialDisplayErasedRoutePreservesEarlierUnjoinedRAW(t *testing.T) {
	r, s := preparationPhysicalFixture(t)
	m := s.manager
	var a Int
	serviceMust(t, r.z.SetLimbs(&a, []uint32{7, 0, 0, 1}))
	serviceMust(t, r.z.Neg(&a))
	var p upolyVectorState
	serviceMust(t, m.set(&p, []Int{a}))
	serviceMust(t, r.z.Del(&a))
	defer func() {
		r.z.budget.FailAfter(-1)
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
	}()
	marker := errors.New("actual earlier failed ostream body")
	e := m.DisplaySMT2Into(m.stateView(&p), &upolyDisplaySink{numeralHook: func(NumeralCStringView) error { return marker }}, "x")
	if !errors.Is(e, marker) || !m.display.active || !m.display.constructed || m.display.numeral.cell == 0 {
		t.Fatal("actual source failed RAW prefix", e)
	}
	older := m.display.numeral
	invocation := m.display.invocation
	cause := m.display.bodyCause
	before := r.z.budget.Used()
	r.z.budget.FailAfter(0)
	for i := 0; i < 2; i++ {
		out := &upolyDisplaySink{}
		serviceMust(t, m.DisplaySMT2Into(UPolynomialView{}, out, "x"))
		if out.text.String() != "0" {
			t.Fatal(out.text.String())
		}
	}
	if m.display.numeral != older || m.display.invocation != invocation || m.display.bodyCause != cause || !m.display.active || r.z.budget.Used() != before {
		t.Fatal("erased later route changed earlier actual RAW authority")
	}
	primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0])
	serviceMust(t, m.CleanupDisplay())
}
