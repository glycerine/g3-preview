// Source: src/math/polynomial/polynomial.cpp:1517-1610,7421-7434,7472-7480;
// src/test/polynomial.cpp:1743-1745 (display-only sites, no assertion credit).
package polynomial

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type polynomialDisplayTestSink struct {
	text         strings.Builder
	literalHook  func(string) error
	decimalHook  func(*PolynomialDecimalView) error
	variableHook func(uint32) error
}

func (s *polynomialDisplayTestSink) WriteLiteral(v string) error {
	if s.literalHook != nil {
		if e := s.literalHook(v); e != nil {
			return e
		}
	}
	s.text.WriteString(v)
	return nil
}
func (s *polynomialDisplayTestSink) WriteUnsigned(v uint32) error {
	s.text.WriteString(strconv.FormatUint(uint64(v), 10))
	return nil
}
func (s *polynomialDisplayTestSink) WriteVariable(v uint32) error {
	if s.variableHook != nil {
		if e := s.variableHook(v); e != nil {
			return e
		}
	}
	s.text.WriteByte('x')
	s.text.WriteString(strconv.FormatUint(uint64(v), 10))
	return nil
}
func (s *polynomialDisplayTestSink) WriteDecimal(v *PolynomialDecimalView) error {
	if s.decimalHook != nil {
		if e := s.decimalHook(v); e != nil {
			return e
		}
	}
	n, e := v.Length()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		b, e := v.Byte(i)
		if e != nil {
			return e
		}
		s.text.WriteByte(b)
	}
	return nil
}

func TestPolynomialDisplayLiteralSourceQuirks(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 0, 0}}, {5, [3]uint32{0, 0, 0}}})
	for _, star := range []bool{false, true} {
		s := &polynomialDisplayTestSink{}
		serviceMust(t, m.DisplayInto(p, s, star))
		want := "3 x0^2 - x0 + 5"
		if star {
			want = "3*x0^2 - x0 + 5"
		}
		if s.text.String() != want {
			t.Fatal("literal source display", s.text.String(), want)
		}
	}
	s := &polynomialDisplayTestSink{}
	serviceMust(t, m.DisplaySMT2Into(p, s))
	// Source display_mon_smt2 calls ordinary monomial display, with no space
	// inserted between numeric output and the first variable.
	if s.text.String() != "(+ (* 3x0^2) (* (- 1)x0) 5)" {
		t.Fatal("literal source SMT2 quirk", s.text.String())
	}
	serviceMust(t, m.CleanupDisplay())
}
func TestPolynomialDisplayZeroAndRefVectorSuffix(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, nil)
	s := &polynomialDisplayTestSink{}
	serviceMust(t, m.DisplayInto(p, s, false))
	serviceMust(t, m.DisplaySMT2Into(p, s))
	if s.text.String() != "00" {
		t.Fatal("source zero output", s.text.String())
	}
	v, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, v.Close()) }()
	serviceMust(t, v.Push(p))
	serviceMust(t, v.Push(p))
	s = &polynomialDisplayTestSink{}
	serviceMust(t, v.DisplayInto(s))
	if s.text.String() != "0\n0\n" {
		t.Fatal("source vector newline suffix", s.text.String())
	}
	serviceMust(t, m.CleanupDisplay())
}
func TestPolynomialDisplayScalarCStringAliveAndNested(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, nil)
	z := m.core.integer
	a := util.Int{}
	serviceMust(t, z.Copy(&a, &z.Two64))
	serviceMust(t, z.Neg(&a))
	p, e := m.core.MkConstantTake(&a)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	defer func() { serviceMust(t, m.core.DecRef(p)) }()
	s := &polynomialDisplayTestSink{}
	visits := 0
	s.decimalHook = func(view *PolynomialDecimalView) error {
		visits++
		nested := &polynomialDisplayTestSink{}
		serviceMust(t, m.DisplaySMT2Into(p, nested))
		if nested.text.String() != "(- 18446744073709551616)" {
			t.Fatal("source-defined nested display", nested.text.String())
		}
		n, e := view.Length()
		serviceMust(t, e)
		if n != 20 {
			t.Fatal("outer borrowed Cptr survives nested frame", n)
		}
		return nil
	}
	serviceMust(t, m.DisplaySMT2Into(p, s))
	if visits != 1 || s.text.String() != "(- 18446744073709551616)" {
		t.Fatal("actual borrowed numeric output", visits, s.text.String())
	}
	serviceMust(t, m.CleanupDisplay())
}

func TestPolynomialDisplayLastCallbackLifetime(t *testing.T) {
	for _, smt := range []bool{false, true} {
		t.Run(strconv.FormatBool(smt), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}})
			// The callback explicitly aborts its actual caller-owned polynomial.
			sink := &polynomialDisplayTestSink{}
			sink.variableHook = func(uint32) error { return m.core.CleanupValue(&p) }
			var e error
			if smt {
				e = m.DisplaySMT2Into(p, sink)
			} else {
				e = m.DisplayInto(p, sink, false)
			}
			if e == nil {
				t.Fatal("source next for-condition must read destroyed polynomial SIZE")
			}
			serviceMust(t, m.CleanupDisplay())
			serviceMust(t, m.CloseChecked())
		})
	}
}
func TestPolynomialDisplayBorrowEndsAfterInsertion(t *testing.T) {
	m, _, _ := polynomialEvalFixture(t, nil)
	z := m.core.integer
	a := util.Int{}
	serviceMust(t, z.Copy(&a, &z.Two64))
	p, e := m.core.MkConstantTake(&a)
	serviceMust(t, e)
	var borrowed PolynomialDecimalView
	sink := &polynomialDisplayTestSink{decimalHook: func(v *PolynomialDecimalView) error {
		borrowed = *v
		if !v.frame.characterCompleted || v.frame.limbs.CompletedField != 0 {
			t.Fatal("actual char live, MPN locals already destructed")
		}
		polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
		return nil
	}}
	serviceMust(t, m.DisplaySMT2Into(p, sink))
	if _, e = borrowed.Byte(0); e == nil {
		t.Fatal("CString borrow escaped its insertion lifetime")
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	serviceMust(t, m.CleanupDisplay())
	serviceMust(t, m.core.CleanupValue(&p))
}

func TestPolynomialDisplayEveryReservation(t *testing.T) {
	for _, smt := range []bool{false, true} {
		t.Run(strconv.FormatBool(smt), func(t *testing.T) {
			success := -1
			for fail := int64(0); fail < 100; fail++ {
				completed := false
				t.Run(strconv.FormatInt(fail, 10), func(t *testing.T) {
					m, _, _ := polynomialEvalFixture(t, nil)
					z := m.core.integer
					digits := make([]uint32, 130)
					digits[129] = 1
					a := util.Int{}
					serviceMust(t, z.SetLimbs(&a, digits))
					serviceMust(t, z.Neg(&a))
					p, e := m.core.MkConstantTake(&a)
					serviceMust(t, e)
					// Independent host oracle is test-only, never the port's arithmetic.
					want := new(big.Int).Lsh(big.NewInt(1), 32*129).String()
					sink := &polynomialDisplayTestSink{}
					budget := z.Budget()
					budget.FailAfter(fail)
					if smt {
						e = m.DisplaySMT2Into(p, sink)
					} else {
						e = m.DisplayInto(p, sink, false)
					}
					polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
					if e != nil && !numericPhysicalReservationRefusal(e) {
						t.Fatal("actual reservation refusal", e)
					}
					budget.FailAfter(-1)
					serviceMust(t, m.CleanupDisplay())
					polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
					if e == nil {
						expected := "- " + want
						if smt {
							expected = "(- " + want + ")"
						}
						if sink.text.String() != expected {
							t.Fatal("full actual scalar source output", sink.text.String(), expected)
						}
						completed = true
					}
					serviceMust(t, m.core.CleanupValue(&p))
				})
				if completed {
					success = int(fail)
					break
				}
			}
			if success < 0 {
				t.Fatal("no full formatting success after all real reservation prefixes")
			}
			t.Log("real failures then first success", success)
		})
	}
}

func TestPolynomialDisplayCloseRefusalPreservesCache(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{7, [3]uint32{1, 0, 0}}})
	sink := &polynomialDisplayTestSink{}
	serviceMust(t, m.DisplayInto(p, sink, false))
	before := m.core.integer.Budget().Used()
	frames := m.core.display.frames
	first := frames[0]
	stats := m.core.memory.Stats()
	if e := m.CloseChecked(); !errors.Is(e, ErrPolynomialLive) {
		t.Fatal("real live caller close guard", e)
	}
	if m.core.integer.Budget().Used() != before || m.core.memory.Stats() != stats || len(m.core.display.frames) != len(frames) || m.core.display.frames[0] != first {
		t.Fatal("rejected owning close changed actual cache/header graph")
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}
func TestPolynomialDisplayRefVectorCachedCount(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{7, [3]uint32{0, 0, 0}}})
	v, e := NewPolynomialRefVector(m.core)
	serviceMust(t, e)
	defer func() { serviceMust(t, v.Close()) }()
	serviceMust(t, v.Push(p))
	serviceMust(t, v.Push(p))
	sink := &polynomialDisplayTestSink{}
	reset := false
	sink.literalHook = func(s string) error {
		if s == "\n" && !reset {
			reset = true
			return v.Reset()
		}
		return nil
	}
	e = v.DisplayInto(sink)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("source seq.get DEBUG current logical size guard", e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("native borrowed span bounds after callback reset", e)
	}
	if sink.text.String() != "7\n" {
		t.Fatal("cached count stopped at current native range", sink.text.String())
	}
	serviceMust(t, m.CleanupDisplay())
}

type polynomialDisplayCountingSink struct{ bytes uint64 }

func (s *polynomialDisplayCountingSink) WriteLiteral(v string) error {
	s.bytes += uint64(len(v))
	return nil
}
func (s *polynomialDisplayCountingSink) WriteUnsigned(v uint32) error {
	s.bytes += uint64(v)
	return nil
}
func (s *polynomialDisplayCountingSink) WriteVariable(v uint32) error {
	s.bytes += uint64(v)
	return nil
}
func (s *polynomialDisplayCountingSink) WriteDecimal(v *PolynomialDecimalView) error {
	n, e := v.Length()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if _, e = v.Byte(i); e != nil {
			return e
		}
	}
	s.bytes += uint64(n)
	return nil
}
func TestPolynomialDisplayWarmInlineFrames(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{7, [3]uint32{0, 0, 0}}})
	sink := &polynomialDisplayCountingSink{}
	serviceMust(t, m.DisplaySMT2Into(p, sink))
	serviceMust(t, m.DisplayInto(p, sink, false))
	used := m.core.integer.Budget().Used()
	stats := m.core.memory.Stats()
	for _, smt := range []bool{false, true} {
		a := testing.AllocsPerRun(100, func() {
			var e error
			if smt {
				e = m.DisplaySMT2Into(p, sink)
			} else {
				e = m.DisplayInto(p, sink, false)
			}
			if e != nil {
				panic(e)
			}
		})
		want := float64(1)
		if smt {
			want = 0
		}
		if a != want {
			t.Fatal("warm source scalar transport allocations", smt, a, want)
		}
	}
	if used != m.core.integer.Budget().Used() || stats != m.core.memory.Stats() {
		t.Fatal("inline warm cache changed real physical/header graph")
	}
	for _, f := range m.core.display.frames {
		if f.active || (f.numeral).Cell != 0 || len(f.characters) != 0 || f.limbs.CompletedField != 0 {
			t.Fatal("successful cached frame retained a lexical owning root")
		}
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}
func TestPolynomialDisplayOwnerGuards(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, nil)
	sink := &polynomialDisplayTestSink{}
	copy := *m
	if e := copy.DisplayInto(p, sink, false); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("copied owner guard", e)
	}
	var nilOwner *MultivariatePolynomialManager
	if e := nilOwner.DisplaySMT2Into(p, sink); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("nil owner guard", e)
	}
	serviceMust(t, m.core.DecRef(p))
	serviceMust(t, m.CloseChecked())
	if e := m.DisplayInto(p, sink, false); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner guard", e)
	}
}

func TestPolynomialDisplayMonomialWrapperAndFiniteField(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{-2, [3]uint32{2, 1, 0}}})
	mon, e := m.core.TermMonomial(p, 0)
	serviceMust(t, e)
	sink := &polynomialDisplayTestSink{}
	serviceMust(t, m.DisplayMonomialInto(mon, sink, true))
	if sink.text.String() != "x0^2*x1" {
		t.Fatal("source monomial delegation", sink.text.String())
	}
	modulus := util.NewInt(5)
	serviceMust(t, m.core.modular.SetZP(&modulus))
	sink = &polynomialDisplayTestSink{}
	serviceMust(t, m.DisplayInto(p, sink, false))
	if sink.text.String() != "- 2 x0^2 x1" {
		t.Fatal("literal balanced finite field display", sink.text.String())
	}
	sink = &polynomialDisplayTestSink{}
	serviceMust(t, m.DisplaySMT2Into(p, sink))
	if sink.text.String() != "(* (- 2)x0^2 x1)" {
		t.Fatal("literal finite field SMT2 display", sink.text.String())
	}
	serviceMust(t, m.core.modular.SetZ())
}
