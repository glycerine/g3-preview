// Source: src/math/polynomial/polynomial.cpp:236-445,919-1268.
package polynomial

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Fresh allocated partial records exercise the typed guard at the reached field;
// they do not claim that an incomplete C++ placement constructor completed.
func monomialReadWrite(t *testing.T, m *MonomialManager, a Monomial, offset uint64, v uint32) {
	t.Helper()
	l, e := m.location(a)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(l.raw, offset, v))
}
func monomialReadMissing(t *testing.T, e error) {
	t.Helper()
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal("actual reached uninitialized field", e)
	}
}
func TestMonomialReadOrderEmptyQueries(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 0)
	v, e := m.IndexOf(a, 7)
	serviceMust(t, e)
	if v != math.MaxUint32 {
		t.Fatal(v)
	}
	v, e = m.DegreeOf(a, 7)
	serviceMust(t, e)
	if v != 0 {
		t.Fatal(v)
	}
	v, e = m.MaxVariable(a)
	serviceMust(t, e)
	if v != math.MaxUint32 {
		t.Fatal(v)
	}
	v, e = m.MaxVariableDegree(a)
	serviceMust(t, e)
	if v != 0 {
		t.Fatal(v)
	}
	v, e = m.MaxSmallerThan(a, 7)
	serviceMust(t, e)
	if v != math.MaxUint32 {
		t.Fatal(v)
	}
	ok, e := m.IsSquare(a)
	serviceMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	ok, e = m.IsPowerOfTwo(a)
	serviceMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	var sink monomialTestSink
	serviceMust(t, m.DisplayInto(a, &sink, false))
	if sink.text.String() != "1" {
		t.Fatal(sink.text.String())
	}
	sink.text.Reset()
	serviceMust(t, m.DisplaySMT2Into(a, &sink))
	if sink.text.String() != "1)" {
		t.Fatal(sink.text.String())
	}
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderVariableOnlySearch(t *testing.T) {
	for _, n := range []uint32{1, 7, 8} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			m := monomialFixture(t)
			a := monomialReadProfileObject(t, m, n)
			for i := uint32(0); i < n; i++ {
				monomialReadWrite(t, m, a, 20+8*uint64(i), 2*i)
			}
			v, e := m.IndexOf(a, 0)
			serviceMust(t, e)
			if v != 0 {
				t.Fatal(v)
			}
			v, e = m.IndexOf(a, 2*(n-1))
			serviceMust(t, e)
			if v != n-1 {
				t.Fatal(v)
			}
			v, e = m.IndexOf(a, 19)
			serviceMust(t, e)
			if v != math.MaxUint32 {
				t.Fatal(v)
			}
			v, e = m.DegreeOf(a, 19)
			serviceMust(t, e)
			if v != 0 {
				t.Fatal(v)
			}
			_, e = m.DegreeOf(a, 0)
			monomialReadMissing(t, e)
			monomialReadWrite(t, m, a, 24, 5)
			v, e = m.DegreeOf(a, 0)
			serviceMust(t, e)
			if v != 5 {
				t.Fatal(v)
			}
			monomialPhysicalAudit(t, m)
		})
	}
}
func TestMonomialReadOrderSeparateMaximumFields(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 1)
	monomialReadWrite(t, m, a, 20, 9)
	v, e := m.MaxVariable(a)
	serviceMust(t, e)
	if v != 9 {
		t.Fatal(v)
	}
	b := monomialReadProfileObject(t, m, 1)
	monomialReadWrite(t, m, b, 24, 6)
	v, e = m.MaxVariableDegree(b)
	serviceMust(t, e)
	if v != 6 {
		t.Fatal(v)
	}
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderMaxSmallerVariableOnly(t *testing.T) {
	for _, n := range []uint32{1, 7, 8} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			m := monomialFixture(t)
			a := monomialReadProfileObject(t, m, n)
			for i := uint32(0); i < n; i++ {
				monomialReadWrite(t, m, a, 20+8*uint64(i), 2*i)
			}
			for _, c := range []struct{ x, w uint32 }{{0, math.MaxUint32}, {1, 0}, {2, 0}, {3, 0}, {16, 2 * (n - 1)}} {
				if c.x == 3 && n > 1 {
					c.w = 2
				}
				v, e := m.MaxSmallerThan(a, c.x)
				serviceMust(t, e)
				if v != c.w {
					t.Fatal(n, c, v)
				}
			}
			monomialPhysicalAudit(t, m)
		})
	}
}
func TestMonomialReadOrderParityOnly(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 2)
	monomialReadWrite(t, m, a, 24, 3)
	ok, e := m.IsSquare(a)
	serviceMust(t, e)
	if ok {
		t.Fatal(ok)
	}
	ok, e = m.IsPowerOfTwo(a)
	serviceMust(t, e)
	if ok {
		t.Fatal(ok)
	}
	monomialReadWrite(t, m, a, 24, 2)
	_, e = m.IsSquare(a)
	monomialReadMissing(t, e)
	monomialReadWrite(t, m, a, 32, 4)
	ok, e = m.IsSquare(a)
	serviceMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	monomialPhysicalAudit(t, m)
}

type monomialReadSink struct {
	text          strings.Builder
	variableError error
}

func (s *monomialReadSink) WriteLiteral(v string) error { s.text.WriteString(v); return nil }
func (s *monomialReadSink) WriteUnsigned(v uint32) error {
	if v == 2 {
		s.text.WriteString("2")
	} else {
		s.text.WriteString("unexpected")
	}
	return nil
}
func (s *monomialReadSink) WriteVariable(v uint32) error {
	if v == 4 {
		s.text.WriteString("x4")
	} else {
		s.text.WriteString("unexpected")
	}
	return s.variableError
}

var monomialSinkRefusal = errors.New("test monomial variable output refusal")

func TestMonomialReadOrderDisplayReachedPrefix(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 1)
	monomialReadWrite(t, m, a, 20, 4)
	s := monomialReadSink{variableError: monomialSinkRefusal}
	e := m.DisplayInto(a, &s, false)
	if !errors.Is(e, monomialSinkRefusal) || s.text.String() != "x4" {
		t.Fatal(e, s.text.String())
	}
	s = monomialReadSink{}
	e = m.DisplayInto(a, &s, false)
	monomialReadMissing(t, e)
	if s.text.String() != "x4" {
		t.Fatal(s.text.String())
	}
	b := monomialReadProfileObject(t, m, 2)
	monomialReadWrite(t, m, b, 20, 4)
	monomialReadWrite(t, m, b, 24, 1)
	s = monomialReadSink{}
	e = m.DisplayInto(b, &s, true)
	monomialReadMissing(t, e)
	if s.text.String() != "x4*" {
		t.Fatal(s.text.String())
	}
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderSMT2DegreeBeforeVariable(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 1)
	monomialReadWrite(t, m, a, 24, 2)
	var s monomialReadSink
	e := m.DisplaySMT2Into(a, &s)
	monomialReadMissing(t, e)
	if s.text.String() != "(*" {
		t.Fatal(s.text.String())
	}
	monomialReadWrite(t, m, a, 20, 4)
	s = monomialReadSink{}
	serviceMust(t, m.DisplaySMT2Into(a, &s))
	if s.text.String() != "(* x4 x4)" {
		t.Fatal(s.text.String())
	}
	monomialReadWrite(t, m, a, 24, 1)
	s = monomialReadSink{}
	serviceMust(t, m.DisplaySMT2Into(a, &s))
	if s.text.String() != "x4)" {
		t.Fatal(s.text.String())
	}
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderRenameVariablePublication(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 1)
	monomialReadWrite(t, m, a, 20, 0)
	e := m.renameObject(a, []uint32{3})
	monomialReadMissing(t, e)
	l, e := m.location(a)
	serviceMust(t, e)
	v, e := m.allocator.ReadUint32(l.raw, 20)
	serviceMust(t, e)
	if v != 3 {
		t.Fatal("source VAR mutation before hash DEG guard", v)
	}
	_, e = m.allocator.ReadUint32(l.raw, 24)
	monomialReadMissing(t, e)
	_, e = m.allocator.ReadUint32(l.raw, 16)
	monomialReadMissing(t, e)
	monomialPhysicalAudit(t, m)
}
func monomialMergePartial(t *testing.T, m *MonomialManager, v uint32, d *uint32) Monomial {
	t.Helper()
	a := monomialReadProfileObject(t, m, 1)
	monomialReadWrite(t, m, a, 20, v)
	if d != nil {
		monomialReadWrite(t, m, a, 24, *d)
	}
	return a
}
func TestMonomialReadOrderProductChosenCopyPrefix(t *testing.T) {
	m := monomialFixture(t)
	d := uint32(3)
	a := monomialMergePartial(t, m, 0, &d)
	b := monomialMergePartial(t, m, 1, nil)
	_, e := m.Mul(a, b)
	monomialReadMissing(t, e)
	p, e := m.tempPower(&m.temps[1], 0)
	serviceMust(t, e)
	if p != (MonomialPower{0, 3}) {
		t.Fatal(p)
	}
	_, e = m.tempPower(&m.temps[1], 1)
	monomialReadMissing(t, e)
	n, e := m.tempSize(&m.temps[1])
	serviceMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderDivisionUnreachedDegrees(t *testing.T) {
	m := monomialFixture(t)
	a := monomialMergePartial(t, m, 2, nil)
	b := monomialMergePartial(t, m, 1, nil)
	monomialReadWrite(t, m, a, 8, 4)
	monomialReadWrite(t, m, b, 8, 4)
	ok, e := m.Divides(a, b)
	serviceMust(t, e)
	if ok {
		t.Fatal(ok)
	}
	_, ok, e = m.Divide(a, b)
	serviceMust(t, e)
	if ok {
		t.Fatal(ok)
	}
	c := monomialReadProfileObject(t, m, 2)
	monomialReadWrite(t, m, c, 20, 0)
	monomialReadWrite(t, m, c, 28, 2)
	monomialReadWrite(t, m, c, 32, 3)
	d := uint32(2)
	r := monomialMergePartial(t, m, 2, &d)
	ok, e = m.divCore(c, r, false)
	serviceMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	_, e = m.divCore(c, r, true)
	monomialReadMissing(t, e)
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderGCDNoCommonUnchosenDegree(t *testing.T) {
	m := monomialFixture(t)
	d := uint32(3)
	a := monomialMergePartial(t, m, 0, &d)
	b := monomialMergePartial(t, m, 1, nil)
	g, q1, q2, e := m.GCD(a, b)
	serviceMust(t, e)
	if g != m.unit || q1 != b || q2 != a {
		t.Fatal(g, q1, q2)
	}
	p, e := m.tempPower(&m.temps[2], 0)
	serviceMust(t, e)
	if p != (MonomialPower{0, 3}) {
		t.Fatal(p)
	}
	_, _, ok, e := m.Unify(a, b)
	serviceMust(t, e)
	if ok {
		t.Fatal(ok)
	}
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderSqrtOddBeforeVariable(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 1)
	monomialReadWrite(t, m, a, 24, 3)
	r, ok, e := m.Sqrt(a)
	serviceMust(t, e)
	if ok || r != (Monomial{}) {
		t.Fatal(r, ok)
	}
	monomialReadWrite(t, m, a, 24, 4)
	_, _, e = m.Sqrt(a)
	monomialReadMissing(t, e)
	monomialPhysicalAudit(t, m)
}
func TestMonomialReadOrderRemovedVariableIgnoresDegree(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 3; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	a := monomialMergePartial(t, m, 2, nil)
	r, e := m.RemoveVariable(a, 2)
	serviceMust(t, e)
	if r != m.unit {
		t.Fatal(r)
	}
	_, e = m.RemoveVariable(a, 1)
	monomialReadMissing(t, e)
	monomialPhysicalAudit(t, m)
}

// The sink can release the actual caller-owned cell. Its next source loop
// condition then reaches a dangling source pointer; Go reports the typed guard
// after preserving output, without pretending the C++ access is defined.
type monomialReleaseSink struct {
	monomialReadSink
	manager   *MonomialManager
	value     *Monomial
	variables uint32
}

func (s *monomialReleaseSink) WriteVariable(v uint32) error {
	if e := s.monomialReadSink.WriteVariable(v); e != nil {
		return e
	}
	s.variables++
	if s.variables == 2 {
		return s.manager.CleanupValue(s.value)
	}
	return nil
}
func (s *monomialReleaseSink) WriteUnsigned(v uint32) error {
	if e := s.monomialReadSink.WriteUnsigned(v); e != nil {
		return e
	}
	return s.manager.CleanupValue(s.value)
}
func TestMonomialReadOrderFinalCallbackReleaseReachedLoop(t *testing.T) {
	for _, smt := range []bool{false, true} {
		name := "ordinary"
		if smt {
			name = "product"
		}
		t.Run(name, func(t *testing.T) {
			m := monomialFixture(t)
			for i := 0; i < 5; i++ {
				_, e := m.MkVar()
				serviceMust(t, e)
			}
			a, e := m.MkVariable(4, 2)
			serviceMust(t, e)
			t.Cleanup(func() {
				if a != (Monomial{}) {
					serviceMust(t, m.CleanupValue(&a))
				}
			})
			sink := monomialReleaseSink{manager: m, value: &a}
			if smt {
				e = m.DisplaySMT2Into(a, &sink)
			} else {
				e = m.DisplayInto(a, &sink, false)
			}
			if !errors.Is(e, ErrMonomialOwner) {
				t.Fatal("next actual source SIZE loop access", e)
			}
			want := "x4^2"
			if smt {
				want = "(* x4 x4"
			}
			if sink.text.String() != want || a != (Monomial{}) {
				t.Fatal(sink.text.String(), a)
			}
			monomialPhysicalAudit(t, m)
		})
	}
}
