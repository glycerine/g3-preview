// Active src/test/mpff.cpp routes.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type ffSource struct {
	t               *testing.T
	count, verifies uint64
	rng             uint32
}

func (h *ffSource) q(ok bool, e error) bool { h.t.Helper(); serviceMust(h.t, e); return ok }
func (h *ffSource) ensure(ok bool) {
	h.t.Helper()
	h.count++
	if !ok {
		h.t.Fatal("source ENSURE", h.count)
	}
}
func (h *ffSource) verify(ok bool) {
	h.t.Helper()
	h.verifies++
	if !ok {
		h.t.Fatal("source VERIFY")
	}
}

// Deterministic nonnegative31bit RAND_MAX adapter, never a libc-sequence claim.
func (h *ffSource) rand() uint32 { h.rng = h.rng*1664525 + 1013904223; return h.rng >> 1 }
func (h *ffSource) manager(p uint32) *FastFloatManager {
	b := arena.NewBudget(1 << 28)
	c := DefaultFastFloatConfig(serviceConfig(b).Integers.Words)
	c.PrecisionWords = p
	m, e := NewFastFloatManager(c)
	serviceMust(h.t, e)
	return m
}
func (h *ffSource) finish(m *FastFloatManager) {
	m.Close()
	if m.config.Storage.Descriptors.Budget.Used() != 0 {
		h.t.Fatal("source owner leak")
	}
}
func (h *ffSource) set64(p uint32) {
	m := h.manager(p)
	defer h.finish(m)
	var a FastFloat
	defer m.Del(&a)
	serviceMust(h.t, m.SetInt64(&a, 9223372036854775807))
	h.ensure(h.q(m.IsInt64(&a)))
	h.ensure(h.q(m.IsUint64(&a)))
	serviceMust(h.t, m.Inc(&a))
	h.ensure(!h.q(m.IsInt64(&a)))
	h.ensure(h.q(m.IsUint64(&a)))
	h.ensure(h.q(m.IsInteger(&a)))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(h.q(m.IsInt64(&a)))
	h.ensure(h.q(m.IsUint64(&a)))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(h.q(m.IsInt64(&a)))
	h.ensure(h.q(m.IsUint64(&a)))
	serviceMust(h.t, m.SetInt64(&a, -9223372036854775808))
	h.ensure(h.q(m.IsInt64(&a)))
	h.ensure(!h.q(m.IsUint64(&a)))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(!h.q(m.IsInt64(&a)))
	h.ensure(!h.q(m.IsUint64(&a)))
	h.ensure(h.q(m.IsInteger(&a)))
	serviceMust(h.t, m.Inc(&a))
	h.ensure(h.q(m.IsInt64(&a)))
	h.ensure(!h.q(m.IsUint64(&a)))
	serviceMust(h.t, m.Inc(&a))
	h.ensure(h.q(m.IsInt64(&a)))
	h.ensure(!h.q(m.IsUint64(&a)))
	serviceMust(h.t, m.SetUint64(&a, ^uint64(0)))
	h.ensure(h.q(m.IsUint64(&a)))
	h.ensure(!h.q(m.IsInt64(&a)))
	serviceMust(h.t, m.Inc(&a))
	h.ensure(!h.q(m.IsUint64(&a)))
	h.ensure(!h.q(m.IsInt64(&a)))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(h.q(m.IsUint64(&a)))
	h.ensure(!h.q(m.IsInt64(&a)))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(h.q(m.IsUint64(&a)))
	h.ensure(!h.q(m.IsInt64(&a)))
	for i := 0; i < 1000; i++ {
		u := uint64(h.rand())<<32 | uint64(h.rand())
		serviceMust(h.t, m.SetUint64(&a, u))
		h.ensure(h.q(m.IsUint64(&a)))
		u = uint64(h.rand()%3)<<32 | uint64(h.rand())
		serviceMust(h.t, m.SetUint64(&a, u))
		h.ensure(h.q(m.IsUint64(&a)))
		v := int64(uint64(h.rand()%2147483647)<<32 | uint64(h.rand()))
		if h.rand()%2 == 0 {
			v = -v
		}
		serviceMust(h.t, m.SetInt64(&a, v))
		h.ensure(h.q(m.IsInt64(&a)))
		v = int64(uint64(h.rand()%3)<<32 | uint64(h.rand()))
		if h.rand()%2 == 0 {
			v = -v
		}
		serviceMust(h.t, m.SetInt64(&a, v))
		h.ensure(h.q(m.IsInt64(&a)))
	}
}
func (h *ffSource) capacity(p uint32) {
	m := h.manager(p)
	defer h.finish(m)
	v, e := NewScopedFastFloatVector(m)
	serviceMust(h.t, e)
	defer v.Close()
	var a FastFloat
	defer m.Del(&a)
	for i := uint32(0); i < 50000; i++ {
		serviceMust(h.t, m.SetUint32(&a, i))
		serviceMust(h.t, v.Push(&a))
		x, e := v.Back()
		serviceMust(h.t, e)
		h.ensure(h.q(m.IsInteger(x)))
		h.ensure(h.q(m.IsInt64(x)))
		h.ensure(h.q(m.IsUint64(x)))
	}
	for i := uint32(0); i < 50000; i++ {
		x, e := v.At(i)
		serviceMust(h.t, e)
		n, e := m.Int64(x)
		serviceMust(h.t, e)
		h.ensure(n == int64(i))
	}
}
func (h *ffSource) power(p uint32) {
	m := h.manager(p)
	defer h.finish(m)
	var a, b FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	h.ensure(h.q(m.IsZero(&a)))
	serviceMust(h.t, m.Power(&a, 10, &a))
	h.ensure(h.q(m.IsZero(&a)))
	for _, n := range []int32{33, -33} {
		serviceMust(h.t, m.SetInt32(&a, n))
		serviceMust(h.t, m.Power(&a, 0, &a))
		h.ensure(h.q(m.IsOne(&a)))
	}
	for _, n := range []int32{33, -33} {
		serviceMust(h.t, m.SetInt32(&a, n))
		serviceMust(h.t, m.Power(&a, 1, &b))
		h.ensure(h.q(m.Equal(&a, &b)))
	}
	for _, tc := range []struct {
		n   int32
		yes bool
		k   uint32
	}{{1, true, 0}, {2, true, 1}, {3, false, 0}, {4, true, 2}, {-4, false, 0}, {8, true, 3}} {
		serviceMust(h.t, m.SetInt32(&a, tc.n))
		ok, k, e := m.PowerOfTwo(&a)
		serviceMust(h.t, e)
		h.ensure(ok == tc.yes && (!ok || k == tc.k))
	}
	serviceMust(h.t, m.SetInt32(&a, 0))
	h.ensure(!h.q(m.IsPowerOfTwo(&a)))
	serviceMust(h.t, m.SetUint32(&a, ^uint32(0)))
	serviceMust(h.t, m.Inc(&a))
	ok, k, e := m.PowerOfTwo(&a)
	serviceMust(h.t, e)
	h.ensure(ok && k == 32)
	u, e := m.Uint64(&a)
	serviceMust(h.t, e)
	h.ensure(u == uint64(^uint32(0))+1)
	for _, tc := range []struct{ p, k uint32 }{{2, 64}, {4, 256}} {
		serviceMust(h.t, m.Power(&a, tc.p, &a))
		ok, k, e = m.PowerOfTwo(&a)
		serviceMust(h.t, e)
		h.ensure(ok && k == tc.k)
	}
	serviceMust(h.t, m.RoundToPlusInf())
	serviceMust(h.t, m.Inc(&a))
	h.ensure(!h.q(m.IsPowerOfTwo(&a)))
	for _, tc := range []struct {
		n, w int32
		p    uint32
	}{{-4, -64, 3}, {-4, 256, 4}, {5, 125, 3}, {-5, -125, 3}} {
		serviceMust(h.t, m.SetInt32(&a, tc.n))
		serviceMust(h.t, m.Power(&a, tc.p, &a))
		serviceMust(h.t, m.SetInt32(&b, tc.w))
		h.ensure(h.q(m.Equal(&a, &b)))
	}
}
func (h *ffSource) signs(p uint32) {
	m := h.manager(p)
	defer h.finish(m)
	var a, b FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	zero := func() bool {
		return h.q(m.IsZero(&a)) && !h.q(m.IsPos(&a)) && !h.q(m.IsNeg(&a)) && h.q(m.IsNonPos(&a)) && h.q(m.IsNonNeg(&a))
	}
	pos := func() bool {
		return !h.q(m.IsZero(&a)) && h.q(m.IsPos(&a)) && !h.q(m.IsNeg(&a)) && !h.q(m.IsNonPos(&a)) && h.q(m.IsNonNeg(&a))
	}
	neg := func() bool {
		return !h.q(m.IsZero(&a)) && !h.q(m.IsPos(&a)) && h.q(m.IsNeg(&a)) && h.q(m.IsNonPos(&a)) && !h.q(m.IsNonNeg(&a))
	}
	h.ensure(zero())
	serviceMust(h.t, m.SetInt32(&a, 3))
	h.ensure(pos())
	serviceMust(h.t, m.SetInt32(&a, -3))
	h.ensure(neg())
	serviceMust(h.t, m.SetInt32(&a, 8))
	serviceMust(h.t, m.Power(&a, 256, &a))
	h.ensure(pos())
	serviceMust(h.t, m.Copy(&b, &a))
	serviceMust(h.t, m.Neg(&a))
	h.ensure(h.q(m.NotEqual(&a, &b)))
	h.ensure(neg())
	serviceMust(h.t, m.Neg(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.SetInt32(&a, 1))
	h.ensure(h.q(m.IsOne(&a)) && !h.q(m.IsZero(&a)) && !h.q(m.IsMinusOne(&a)) && h.q(m.IsAbsOne(&a)))
	serviceMust(h.t, m.Neg(&a))
	h.ensure(!h.q(m.IsOne(&a)) && !h.q(m.IsZero(&a)) && h.q(m.IsMinusOne(&a)) && h.q(m.IsAbsOne(&a)))
	serviceMust(h.t, m.SetInt32(&a, 3))
	h.ensure(!h.q(m.IsOne(&a)) && !h.q(m.IsZero(&a)) && !h.q(m.IsMinusOne(&a)))
	serviceMust(h.t, m.SetInt32(&a, 3))
	serviceMust(h.t, m.Copy(&b, &a))
	serviceMust(h.t, m.Abs(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.SetInt32(&a, -3))
	serviceMust(h.t, m.Copy(&b, &a))
	serviceMust(h.t, m.Abs(&a))
	h.ensure(!h.q(m.Equal(&a, &b)) && h.q(m.IsPos(&a)))
	serviceMust(h.t, m.SetInt32(&a, 1))
	serviceMust(h.t, m.Swap(&a, &a))
	h.ensure(h.q(m.IsOne(&a)))
	serviceMust(h.t, m.SetInt32(&b, -1))
	serviceMust(h.t, m.Swap(&a, &b))
	h.ensure(h.q(m.IsOne(&b)) && h.q(m.IsMinusOne(&a)))
	serviceMust(h.t, m.Neg(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
}
func (h *ffSource) limits(p uint32) {
	m := h.manager(p)
	defer h.finish(m)
	var a, b, two FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	defer m.Del(&two)
	serviceMust(h.t, m.SetMax(&a))
	h.ensure(h.q(m.IsPos(&a)))
	serviceMust(h.t, m.SetMin(&b))
	h.ensure(h.q(m.IsNeg(&b)))
	serviceMust(h.t, m.Neg(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.SetMax(&a))
	serviceMust(h.t, m.SetMax(&b))
	serviceMust(h.t, m.RoundToMinusInf())
	serviceMust(h.t, m.Inc(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(h.q(m.Less(&a, &b)))
	serviceMust(h.t, m.SetMax(&a))
	serviceMust(h.t, m.RoundToPlusInf())
	e := m.Inc(&a)
	overflow := errors.Is(e, ErrFastFloatOverflow)
	h.verify(overflow)
	serviceMust(h.t, m.SetMax(&a))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.SetMin(&a))
	serviceMust(h.t, m.SetMin(&b))
	serviceMust(h.t, m.RoundToMinusInf())
	serviceMust(h.t, m.Inc(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
	overflow = true
	e = m.Dec(&a)
	if errors.Is(e, ErrFastFloatOverflow) {
		overflow = true
	}
	h.ensure(overflow) // Exact source typo retained; meaningful extra check below.
	if !errors.Is(e, ErrFastFloatOverflow) {
		h.t.Fatal("added independent negative overflow", e)
	}
	serviceMust(h.t, m.RoundToPlusInf())
	serviceMust(h.t, m.SetMin(&a))
	serviceMust(h.t, m.Inc(&a))
	h.ensure(h.q(m.Greater(&a, &b)))
	serviceMust(h.t, m.SetMin(&a))
	serviceMust(h.t, m.Dec(&a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.SetPlusEpsilon(&a))
	serviceMust(h.t, m.SetPlusEpsilon(&b))
	h.ensure(!h.q(m.IsZero(&a)) && h.q(m.IsPos(&a)))
	serviceMust(h.t, m.SetInt32(&two, 2))
	serviceMust(h.t, m.RoundToPlusInf())
	serviceMust(h.t, m.Div(&a, &two, &a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.RoundToMinusInf())
	serviceMust(h.t, m.Div(&a, &two, &a))
	h.ensure(h.q(m.IsZero(&a)))
	for _, up := range []bool{true, false} {
		serviceMust(h.t, m.SetRounding(up))
		serviceMust(h.t, m.SetPlusEpsilon(&a))
		serviceMust(h.t, m.Add(&a, &a, &a))
		h.ensure(h.q(m.Greater(&a, &b)))
	}
	serviceMust(h.t, m.SetPlusEpsilon(&a))
	serviceMust(h.t, m.Sub(&a, &a, &a))
	h.ensure(h.q(m.IsZero(&a)))
	serviceMust(h.t, m.SetPlusEpsilon(&a))
	h.ensure(h.q(m.IsPlusEpsilon(&a)))
	h.ensure(!h.q(m.IsMinusEpsilon(&a)))
	serviceMust(h.t, m.Neg(&a))
	h.ensure(!h.q(m.IsPlusEpsilon(&a)))
	h.ensure(h.q(m.IsMinusEpsilon(&a)))
	for _, up := range []bool{true, false} {
		serviceMust(h.t, m.SetRounding(up))
		serviceMust(h.t, m.SetPlusEpsilon(&a))
		serviceMust(h.t, m.FloorInPlace(&a))
		h.ensure(h.q(m.IsZero(&a)))
		serviceMust(h.t, m.SetPlusEpsilon(&a))
		serviceMust(h.t, m.CeilInPlace(&a))
		h.ensure(h.q(m.IsOne(&a)))
		serviceMust(h.t, m.SetMinusEpsilon(&a))
		serviceMust(h.t, m.FloorInPlace(&a))
		h.ensure(h.q(m.IsMinusOne(&a)))
		serviceMust(h.t, m.SetMinusEpsilon(&a))
		serviceMust(h.t, m.CeilInPlace(&a))
		h.ensure(h.q(m.IsZero(&a)))
	}
	serviceMust(h.t, m.SetMinusEpsilon(&a))
	serviceMust(h.t, m.SetMinusEpsilon(&b))
	h.ensure(!h.q(m.IsZero(&a)) && h.q(m.IsNeg(&a)))
	serviceMust(h.t, m.SetInt32(&two, 2))
	serviceMust(h.t, m.RoundToMinusInf())
	serviceMust(h.t, m.Div(&a, &two, &a))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.RoundToPlusInf())
	serviceMust(h.t, m.Div(&a, &two, &a))
	h.ensure(h.q(m.IsZero(&a)))
	for _, up := range []bool{true, false} {
		serviceMust(h.t, m.SetRounding(up))
		serviceMust(h.t, m.SetMinusEpsilon(&a))
		serviceMust(h.t, m.Add(&a, &a, &a))
		h.ensure(h.q(m.Less(&a, &b)))
	}
	serviceMust(h.t, m.SetMinusEpsilon(&a))
	serviceMust(h.t, m.Sub(&a, &a, &a))
	h.ensure(h.q(m.IsZero(&a)))
	serviceMust(h.t, m.SetMinusEpsilon(&a))
	h.ensure(!h.q(m.IsPlusEpsilon(&a)))
	h.ensure(h.q(m.IsMinusEpsilon(&a)))
	serviceMust(h.t, m.Neg(&a))
	h.ensure(h.q(m.IsPlusEpsilon(&a)))
	h.ensure(!h.q(m.IsMinusEpsilon(&a)))
}
func (h *ffSource) bugs() {
	m := h.manager(4)
	defer h.finish(m)
	var a, b FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	serviceMust(h.t, m.SetInt32(&b, 1))
	serviceMust(h.t, m.Sub(&a, &b, &b))
	serviceMust(h.t, m.SetInt32(&a, -1))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.SetInt32(&a, 1))
	serviceMust(h.t, m.SetInt32(&b, 0))
	serviceMust(h.t, m.Sub(&a, &b, &a))
	serviceMust(h.t, m.SetInt32(&b, 1))
	h.ensure(h.q(m.Equal(&a, &b)))
	serviceMust(h.t, m.SetInt32(&a, 1))
	serviceMust(h.t, m.SetInt32(&b, 1))
	serviceMust(h.t, m.Sub(&a, &b, &a))
	h.ensure(h.q(m.IsZero(&a)))
}
func (h *ffSource) core(op uint32, n1 int64, d1 uint64, n2 int64, d2 uint64, p, exp uint32) {
	m := h.manager(p)
	defer h.finish(m)
	q, e := NewRationalManager(RationalConfig{Integers: serviceConfig(m.config.Storage.Descriptors.Budget).Integers})
	serviceMust(h.t, e)
	defer q.Close()
	qa, qb, qc, qt := NewRational(), NewRational(), NewRational(), NewRational()
	defer q.Del(&qa)
	defer q.Del(&qb)
	defer q.Del(&qc)
	defer q.Del(&qt)
	var a, b, c1, c2 FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	defer m.Del(&c1)
	defer m.Del(&c2)
	serviceMust(h.t, m.SetFraction64(&a, n1, d1))
	if exp != 0 {
		v := int32(h.rand() % exp)
		if h.rand()%2 == 0 {
			v = -v
		}
		serviceMust(h.t, m.SetExponent(&a, v))
	}
	serviceMust(h.t, m.ToRational(&a, q, &qa))
	serviceMust(h.t, m.SetFraction64(&b, n2, d2))
	if exp != 0 {
		v := int32(h.rand() % exp)
		if h.rand()%2 == 0 {
			v = -v
		}
		serviceMust(h.t, m.SetExponent(&b, v))
	}
	serviceMust(h.t, m.ToRational(&b, q, &qb))
	switch op {
	case 0:
		e = q.Add(&qa, &qb, &qc)
	case 1:
		e = q.Sub(&qa, &qb, &qc)
	case 2:
		e = q.Mul(&qa, &qb, &qc)
	default:
		e = q.Div(&qa, &qb, &qc)
	}
	serviceMust(h.t, e)
	for _, up := range []bool{true, false} {
		serviceMust(h.t, m.SetRounding(up))
		out := &c1
		if !up {
			out = &c2
		}
		switch op {
		case 0:
			e = m.Add(&a, &b, out)
		case 1:
			e = m.Sub(&a, &b, out)
		case 2:
			e = m.Mul(&a, &b, out)
		default:
			e = m.Div(&a, &b, out)
		}
		serviceMust(h.t, e)
		serviceMust(h.t, m.ToRational(out, q, &qt))
		if up {
			h.ensure(h.le(q, &qc, &qt))
		} else {
			h.ensure(h.le(q, &qt, &qc))
		}
	}
	h.ensure(h.q(m.LessEqual(&c2, &c1)))
}
func TestFastFloatSourceActiveMatrix(t *testing.T) {
	h := ffSource{t: t, rng: 1}
	for _, p := range []uint32{2, 4} {
		h.limits(p)
	}
	for _, p := range []uint32{2, 4, 8} {
		h.signs(p)
	}
	for _, p := range []uint32{2, 4, 18} {
		h.power(p)
	}
	for _, p := range []uint32{2, 4, 8, 16} {
		h.capacity(p)
	}
	for _, p := range []uint32{2, 4, 6} {
		h.set64(p)
	}
	h.bugs()
	for _, max := range []uint32{1024, ^uint32(0)} {
		for _, op := range []uint32{1, 3, 2, 0} {
			for _, p := range []uint32{2, 4} {
				for i := 0; i < 1000; i++ {
					n1, d1, n2, d2 := int64(h.rand()%max), uint64(h.rand()%max+1), int64(h.rand()%max), uint64(h.rand()%max+1)
					if h.rand()%2 == 0 {
						n1 = -n1
					}
					if h.rand()%2 == 0 {
						n2 = -n2
					}
					if op == 3 && n2 == 0 {
						n2 = 1
					}
					h.core(op, n1, d1, n2, d2, p, 0)
					h.core(op, n1, d1, n2, d2, p, 512)
				}
			}
		}
	}
	h.bugs()
	h.core(3, 679, 396, 279, 756, 2, 0)
	h.core(0, 1, 1, 1, 1, 2, 0)
	h.core(0, 1, 3, 2, 3, 2, 0)
	// tst_bug: original rational 41/36 <= converted upward MPFF.
	m := h.manager(2)
	q, e := NewRationalManager(RationalConfig{Integers: serviceConfig(m.config.Storage.Descriptors.Budget).Integers})
	serviceMust(t, e)
	var a FastFloat
	b, c := NewRational(), NewRational()
	serviceMust(t, m.RoundToPlusInf())
	serviceMust(t, m.SetFraction32(&a, 41, 36))
	serviceMust(t, q.SetFraction(&b, 41, 36))
	serviceMust(t, m.ToRational(&a, q, &c))
	h.ensure(h.le(q, &b, &c))
	serviceMust(t, m.Del(&a))
	serviceMust(t, q.Del(&b))
	serviceMust(t, q.Del(&c))
	q.Close()
	h.finish(m)
	for _, tc := range []struct {
		n int64
		d uint64
		k uint32
	}{{-10, 1, 0}, {0, 1, 0}, {1, 1, 0}, {2, 1, 1}, {3, 1, 1}, {4, 1, 2}, {5, 1, 2}, {8, 1, 3}, {9, 1, 3}, {9, 2, 2}, {9, 4, 1}, {9, 5, 0}, {(1 << 60) + 1, 1, 60}, {1 << 60, 1, 60}, {(1 << 60) - 1, 1, 59}, {1 << 60, 3, 58}} {
		m = h.manager(2)
		a = FastFloat{}
		serviceMust(t, m.SetFraction64(&a, tc.n, tc.d))
		k, e := m.PrevPowerOfTwo(&a)
		serviceMust(t, e)
		h.ensure(k == tc.k)
		serviceMust(t, m.Del(&a))
		h.finish(m)
	}
	if h.count != 908297 || h.verifies != 2 {
		t.Fatal("source execution counts excluding5basic+14decimal", h.count, h.verifies)
	}
}

func (h *ffSource) le(q *RationalManager, a, b *Rational) bool {
	c, e := q.Compare(b, a)
	serviceMust(h.t, e)
	return !(c < 0)
}
