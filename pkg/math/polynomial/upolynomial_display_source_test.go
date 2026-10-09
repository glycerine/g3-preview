// Source: src/math/polynomial/upolynomial.cpp:1108-1225,3120-3124;
// src/test/upolynomial.cpp:23-65,468-490 (display driver sites).
package polynomial

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// The driver prints these values without ENSUREs. These exact stream checks
// are new independent output assertions, not registered source assertion credit.
type upolyDisplaySink struct {
	text        strings.Builder
	literalHook func(string) error
	numeralHook func(util.NumeralCStringView) error
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
func (s *upolyDisplaySink) AppendNumeralCString(v util.NumeralCStringView) error {
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
	serviceMust(t, m.set(&p, []util.Int{util.NewInt(18), util.NewInt(9), util.NewInt(-20), util.NewInt(-10), util.NewInt(2), util.NewInt(1)}))
	defer func() { serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	out := &upolyDisplaySink{}
	serviceMust(t, m.DisplayInto(m.stateView(&p), out, "x", false))
	if out.text.String() != "x^5 + 2 x^4 - 10 x^3 - 20 x^2 + 9 x + 18" {
		t.Fatal(out.text.String())
	}
	// Driver's coefficient traversal adds one, then trims the actual vector.
	for i := uint32(0); i < p.size; i++ {
		one := util.NewInt(1)
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
		values []util.Int
		want   string
	}{
		{nil, "0"},
		{[]util.Int{util.NewInt(-3)}, "(- 3)"},
		{[]util.Int{util.NewInt(0), util.NewInt(1)}, "x(+ x)"},
		{[]util.Int{util.NewInt(0), util.NewInt(0), util.NewInt(-1)}, "(* (- 1) (^ x 2))(+ (* (- 1) (^ x 2)))"},
		{[]util.Int{util.NewInt(5), util.NewInt(-1), util.NewInt(2)}, "(+ (* 2 (^ x 2)) (* (- 1) x) 5)"},
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
	serviceMust(t, m.set(&p, []util.Int{util.NewInt(1), util.NewInt(1)}))
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
		before := m.config.Integer.Budget().Used()
		e := m.Close()
		if e != ErrUPolynomialCleanupPending || m.closed || m.closing || before != m.config.Integer.Budget().Used() {
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
			serviceMust(t, m.set(&p, []util.Int{util.NewInt(1), util.NewInt(1)}))
			defer func() {
				if !r.Integers().Closed() {
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
				before := r.Integers().Budget().Used()
				var e error
				switch mode {
				case "integer-source":
					e = r.Integers().SourceFinalize()
				case "integer-own":
					e = r.Integers().CloseChecked()
				case "rational-source":
					e = r.SourceFinalize()
				case "rational-own":
					e = r.CloseChecked()
				}
				if e != ErrPolynomialPending || r.Integers().Closed() || r.Integers().SourceFinalizing || r.Closed() || r.SourceFinalizing || before != r.Integers().Budget().Used() {
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
			var a util.Int
			serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
			serviceMust(t, r.Integers().Neg(&a))
			var p upolyVectorState
			serviceMust(t, m.set(&p, []util.Int{a}))
			serviceMust(t, r.Integers().Del(&a))
			defer func() { serviceMust(t, m.CleanupDisplay()); serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
			marker := errors.New("actual ostream refused decimal borrow")
			var borrowed util.NumeralCStringView
			out := &upolyDisplaySink{numeralHook: func(v util.NumeralCStringView) error { borrowed = v; return marker }}
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
			roots := []*util.Int{&m.display.numeral}
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
					var a util.Int
					serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
					serviceMust(t, r.Integers().Neg(&a))
					var p upolyVectorState
					serviceMust(t, m.set(&p, []util.Int{a}))
					serviceMust(t, r.Integers().Del(&a))
					r.Integers().Budget().FailAfter(ordinal)
					out := &upolyDisplaySink{}
					var e error
					if smt {
						e = m.DisplaySMT2Into(m.stateView(&p), out, "x")
					} else {
						e = m.DisplayInto(m.stateView(&p), out, "x", false)
					}
					roots := []*util.Int{&m.display.numeral}
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
					r.Integers().Budget().FailAfter(-1)
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
	serviceMust(t, m.set(&p, []util.Int{util.NewInt(8), util.NewInt(2), util.NewInt(8), util.NewInt(10), util.NewInt(10), util.NewInt(0), util.NewInt(1), util.NewInt(0), util.NewInt(1), util.NewInt(3), util.NewInt(7)}))
	defer func() { serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	seq, e := m.NewSequence()
	serviceMust(t, e)
	defer func() { serviceMust(t, seq.Close()) }()
	serviceMust(t, m.SturmSequence(m.stateView(&p), seq))
	n, e := seq.Size()
	serviceMust(t, e)
	out := &upolyDisplaySink{literalHook: func(text string) error {
		if text == "\n" {
			before := m.config.Integer.Budget().Used()
			if e := seq.Close(); e != ErrUPolynomialCleanupPending || seq.closed || before != m.config.Integer.Budget().Used() {
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
	serviceMust(t, m.set(&p, []util.Int{util.NewInt(0), util.NewInt(1)}))
	defer func() { serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	var old integerAllocationInvocationContext
	called := false
	out := &upolyDisplaySink{literalHook: func(string) error {
		if called {
			return nil
		}
		called = true
		old = m.display.context()
		dst := util.NewInt(9)
		src := util.NewInt(1)
		before := m.config.Integer.Budget().Used()
		if e := numericIntegerModularManager_copyWithDisplayContext(m.modular, &dst, &src, old); e != ErrUPolynomialOwner || dst != util.NewInt(9) || before != m.config.Integer.Budget().Used() {
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
	serviceMust(t, m.set(&p, []util.Int{util.NewInt(0), util.NewInt(1)}))
	serviceMust(t, m.set(&n, []util.Int{util.NewInt(-2)}))
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
	serviceMust(t, m.set(&p, []util.Int{util.NewInt(0), util.NewInt(1)}))
	serviceMust(t, m.set(&n, []util.Int{util.NewInt(-2)}))
	defer func() {
		m.config.Integer.Budget().FailAfter(-1)
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
		m.config.Integer.Budget().FailAfter(0)
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
	var a util.Int
	serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
	var p upolyVectorState
	serviceMust(t, m.set(&p, []util.Int{a}))
	serviceMust(t, r.Integers().Del(&a))
	defer func() { serviceMust(t, m.CleanupDisplay()); serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
	called := false
	var old util.NumeralCStringView
	out := &upolyDisplaySink{numeralHook: func(v util.NumeralCStringView) error {
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
	var a util.Int
	serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
	var p, n upolyVectorState
	serviceMust(t, m.set(&p, []util.Int{a}))
	serviceMust(t, r.Integers().Del(&a))
	serviceMust(t, m.set(&n, []util.Int{util.NewInt(-2)}))
	defer func() {
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
		serviceMust(t, m.reset(&n))
		m.freeBacking(&n)
	}()
	out := &upolyDisplaySink{numeralHook: func(v util.NumeralCStringView) error {
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
			var a util.Int
			serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
			serviceMust(t, r.Integers().Neg(&a))
			var p, n upolyVectorState
			serviceMust(t, m.set(&p, []util.Int{util.NewInt(0), util.NewInt(1)}))
			serviceMust(t, m.set(&n, []util.Int{a}))
			serviceMust(t, r.Integers().Del(&a))
			var nested error
			called := false
			out := &upolyDisplaySink{literalHook: func(string) error {
				if called {
					return nil
				}
				called = true
				r.Integers().Budget().FailAfter(ordinal)
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
			r.Integers().Budget().FailAfter(-1)
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
			var a util.Int
			serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
			var p upolyVectorState
			serviceMust(t, m.set(&p, []util.Int{a}))
			serviceMust(t, r.Integers().Del(&a))
			defer func() { serviceMust(t, m.CleanupDisplay()); serviceMust(t, m.reset(&p)); m.freeBacking(&p) }()
			called := false
			out := &upolyDisplaySink{numeralHook: func(v util.NumeralCStringView) error {
				called = true
				n := r.Integers().Naturals()
				before := r.Integers().Budget().Used()
				first, e := v.Byte(0)
				serviceMust(t, e)
				if mode == "source" {
					e = n.SourceFinalize()
				} else {
					e = n.CloseChecked()
				}
				if e == nil || n.ClosedField || n.Closing || n.SourceFinalizing || before != r.Integers().Budget().Used() {
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
func TestUPolynomialDisplayErasedRoutePreservesEarlierUnjoinedRAW(t *testing.T) {
	r, s := preparationPhysicalFixture(t)
	m := s.manager
	var a util.Int
	serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
	serviceMust(t, r.Integers().Neg(&a))
	var p upolyVectorState
	serviceMust(t, m.set(&p, []util.Int{a}))
	serviceMust(t, r.Integers().Del(&a))
	defer func() {
		r.Integers().Budget().FailAfter(-1)
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
	}()
	marker := errors.New("actual earlier failed ostream body")
	e := m.DisplaySMT2Into(m.stateView(&p), &upolyDisplaySink{numeralHook: func(util.NumeralCStringView) error { return marker }}, "x")
	if !errors.Is(e, marker) || !m.display.active || !m.display.constructed || (m.display.numeral).Cell == 0 {
		t.Fatal("actual source failed RAW prefix", e)
	}
	older := m.display.numeral
	invocation := m.display.invocation
	cause := m.display.bodyCause
	before := r.Integers().Budget().Used()
	r.Integers().Budget().FailAfter(0)
	for i := 0; i < 2; i++ {
		out := &upolyDisplaySink{}
		serviceMust(t, m.DisplaySMT2Into(UPolynomialView{}, out, "x"))
		if out.text.String() != "0" {
			t.Fatal(out.text.String())
		}
	}
	if m.display.numeral != older || m.display.invocation != invocation || m.display.bodyCause != cause || !m.display.active || r.Integers().Budget().Used() != before {
		t.Fatal("erased later route changed earlier actual RAW authority")
	}
	primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0])
	serviceMust(t, m.CleanupDisplay())
}
