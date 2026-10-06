// Source: src/util/mpq.h:199-201.
// Source: src/util/mpq.h; src/util/mpq.cpp. Licensed under the MIT license.
// Source: src/util/mpq.h and src/util/mpq.cpp, Leonardo de Moura (2010-06-21).
package mpnum

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

var ErrUninitializedRational = errors.New("mpnum: rational has zero denominator; use NewRational")

// Rational is the source pair of pointer-free, owner-local integer numerals.
// NewRational is required: a raw Go zero value has an invalid denominator.
// Big values must never be shallow-copied or used with another manager.
type Rational struct{ num, den Int }

func NewRational() Rational       { return Rational{den: NewInt(1)} }
func (a *Rational) IsSmall() bool { return a.num.IsSmall() && a.den.IsSmall() }

type RationalConfig struct {
	Integers     IntegerConfig
	Synchronized bool
}

// RationalManager owns an IntegerManager and the source scalar scratch fields.
// Synchronized selects source local-temporary algorithms; false selects the
// source reusable scratch algorithms. Both are owner-confined, not concurrent.
// Manager header bytes include all retained scalar rational scratch members.
type RationalManager struct {
	lexical                                        rationalManagerLexicalScope
	rawArithmetic                                  rationalRawArithmeticScope
	rawGeneration                                  uint64
	self                                           *RationalManager
	originalCause, cleanupCause                    error
	cleanupStage                                   uint8
	closing                                        bool
	sourceFinalizing, sourceFinalized, sourceGuard bool
	sourceCause                                    error
	z                                              *IntegerManager
	nTmp, addTmp1, addTmp2                         Int
	addMulTmp, ltTmp1, ltTmp2                      Rational
	synchronized, closed                           bool
	budget                                         *arena.Budget
	headerBytes                                    uint64
}

func NewRationalManager(config RationalConfig) (*RationalManager, error) {
	if config.Integers.Cells.Budget != config.Integers.Words.Descriptors.Budget {
		return nil, arena.ErrConfig
	}
	budget := config.Integers.Cells.Budget
	header := uint64(reflect.TypeOf(RationalManager{}).Size())
	if e := budget.Reserve(header); e != nil {
		return nil, e
	}
	m := &RationalManager{synchronized: config.Synchronized, budget: budget, headerBytes: header, addMulTmp: NewRational(), ltTmp1: NewRational(), ltTmp2: NewRational()}
	m.self = m
	z, e := NewIntegerManager(config.Integers)
	if e != nil {
		budget.Release(header)
		return nil, e
	}
	m.z = z
	return m, nil
}
func (m *RationalManager) Integers() *IntegerManager { return m.z }

// checkOwnerReady guards the cold owner without reading either numeral field.
func (m *RationalManager) checkOwnerReady() error {
	if m == nil || m.self != m || m.z == nil || m.z.self != m.z {
		return memory.ErrRuntimeOwner
	}
	if m.closed || m.closing || m.sourceFinalizing || m.sourceFinalized || m.z.closed || m.z.sourceFinalizing || m.z.constructionFailed {
		return arena.ErrClosed
	}
	return nil
}
func (m *RationalManager) check(a *Rational) error {
	if e := m.checkOwnerReady(); e != nil {
		return e
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if e := m.z.check(&a.num); e != nil {
		return e
	}
	if e := m.z.check(&a.den); e != nil {
		return e
	}
	if a.den.val == 0 {
		return ErrUninitializedRational
	}
	return nil
}
func (m *RationalManager) checkCleanup(a *Rational) error {
	if e := m.checkOwnerReady(); e != nil {
		return e
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if e := m.z.checkCleanup(&a.num); e != nil {
		return e
	}
	if e := m.z.checkCleanup(&a.den); e != nil {
		return e
	}
	// Cleanup/reset acts on owned components, including finite source results
	// with denominator zero. This does not authorize ordinary numeral reads.
	return nil
}
func (m *RationalManager) drop(a *Rational) { m.z.Del(&a.num); m.z.Del(&a.den) }
func (m *RationalManager) Close() {
	if e := m.CloseChecked(); e != nil {
		panic(m)
	}
}
func (m *RationalManager) resetDenominator(a *Rational) error {
	if e := m.z.Del(&a.den); e != nil {
		return e
	}
	a.den.val = 1
	return nil
}
func isIntegerRational(a *Rational) bool { return a.den.IsSmall() && a.den.val == 1 }
func isOneInteger(a *Int) bool           { return a.IsSmall() && a.val == 1 }

func (m *RationalManager) Del(a *Rational) error {
	if e := m.checkCleanup(a); e != nil {
		return e
	}
	if e := m.z.Del(&a.num); e != nil {
		return e
	}
	return m.z.Del(&a.den)
}
func (m *RationalManager) Reset(a *Rational) error {
	if e := m.checkScalarValue(a); e != nil {
		return e
	}
	if e := m.z.Reset(&a.num); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) Neg(a *Rational) error {
	return m.negWithSourceContext(a, integerAllocationInvocationContext{})
}
func (m *RationalManager) negWithSourceContext(a *Rational, context integerAllocationInvocationContext) error {
	if e := m.checkScalarValue(a); e != nil {
		return e
	}
	return m.z.negWithAllocationContext(&a.num, context)
}
func (m *RationalManager) Abs(a *Rational) error {
	if e := m.checkScalarValue(a); e != nil {
		return e
	}
	return m.z.Abs(&a.num)
}
func (m *RationalManager) Move(dst, src *Rational) error {
	if e := m.checkScalarValue(dst); e != nil {
		return e
	}
	if e := m.checkScalarValue(src); e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if e := m.z.Move(&dst.num, &src.num); e != nil {
		return e
	}
	return m.z.Move(&dst.den, &src.den)
}
func (m *RationalManager) Swap(a, b *Rational) error {
	if e := m.checkScalarValue(a); e != nil {
		return e
	}
	if e := m.checkScalarValue(b); e != nil {
		return e
	}
	if e := m.z.Swap(&a.num, &b.num); e != nil {
		return e
	}
	return m.z.Swap(&a.den, &b.den)
}
func (m *RationalManager) Copy(dst, src *Rational) error {
	if e := m.checkScalarValue(dst); e != nil {
		return e
	}
	if e := m.checkScalarValue(src); e != nil {
		return e
	}
	if e := m.z.Copy(&dst.num, &src.num); e != nil {
		return e
	}
	return m.z.Copy(&dst.den, &src.den)
}

// checkSetter authorizes explicit initialization/reinitialization, including
// inline denominator0. Failed/stale Integer handles remain ordinary errors;
// arithmetic and source-read checks retain their separate denominator policy.
func (m *RationalManager) checkSetter(a *Rational) error {
	return m.checkIntegerFields(a)
}

// checkIntegerFields validates actual component ownership without imposing an
// ordinary rational-read domain. Source Mul/Div can consume and publish zero
// denominators; their arithmetic determines the first endangered operation.
func (m *RationalManager) checkIntegerFields(a *Rational) error {
	if e := m.checkOwnerReady(); e != nil {
		return e
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	if e := m.z.check(&a.num); e != nil {
		return e
	}
	return m.z.check(&a.den)
}
func (m *RationalManager) SetInt32(a *Rational, n int32) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if e := m.z.SetInt32(&a.num, n); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) SetUint32(a *Rational, n uint32) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if e := m.z.SetUint32(&a.num, n); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) SetInt64(a *Rational, n int64) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if e := m.z.SetInt64(&a.num, n); e != nil {
		return e
	}
	return m.resetDenominator(a)
}
func (m *RationalManager) SetFraction(a *Rational, n int64, d uint64) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if modularSourceDebug && d == 0 {
		return ErrIntegerAssertion
	}
	if e := m.z.SetInt64(&a.num, n); e != nil {
		return e
	}
	if e := m.z.SetUint64(&a.den, d); e != nil {
		return e
	}
	return m.normalize(a)
}

// SetFractionSigned is the source signed32 pair overload. Widening the two
// local negations defines intended INT_MIN magnitudes where C++ unary signed
// negation is undefined; it preserves the existing service adapter.
func (m *RationalManager) SetFractionSigned(a *Rational, n, d int32) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if modularSourceDebug && d == 0 {
		return ErrIntegerAssertion
	}
	nn, dd := int64(n), int64(d)
	if dd < 0 {
		nn = -nn
		dd = -dd
	}
	if e := m.z.SetInt64(&a.num, nn); e != nil {
		return e
	}
	if e := m.z.SetInt64(&a.den, dd); e != nil {
		return e
	}
	return m.normalize(a)
}
func (m *RationalManager) SetParts(a *Rational, n, d *Int) error {
	if e := m.checkSetter(a); e != nil {
		return e
	}
	if e := m.z.check(d); e != nil {
		return e
	}
	negative := d.val < 0 // source branch decision precedes both caller Copies
	if e := m.z.Copy(&a.num, n); e != nil {
		return e
	}
	if e := m.z.Copy(&a.den, d); e != nil {
		return e
	}
	if negative {
		if e := m.z.Neg(&a.num); e != nil {
			return e
		}
		if e := m.z.Neg(&a.den); e != nil {
			return e
		}
	}
	return m.normalize(a)
}
func (m *RationalManager) normalize(a *Rational) (_rawErr error) {
	tmp := &m.nTmp
	if m.synchronized {
		r := &m.rawArithmetic
		if r.normActive {
			return ErrRationalRawArithmeticPending
		}
		generation, e := m.beginRawGeneration()
		if e != nil {
			return e
		}
		r.normGeneration = generation
		r.norm = Int{}
		r.normActive = true
		r.normEnd = false
		r.normGuard = false
		r.normCause = nil
		tmp = &r.norm
		defer m.finishRawNormalize(&_rawErr)
	}
	if e := m.z.GCD(&a.num, &a.den, tmp); e != nil {
		return e
	}
	if !isOneInteger(tmp) {
		if e := m.z.Div(&a.num, tmp, &a.num); e != nil {
			return e
		}
		if e := m.z.Div(&a.den, tmp, &a.den); e != nil {
			return e
		}
	}
	if m.synchronized {
		return m.rawNormalizeEnd()
	}
	return nil
}

// Inv copies into the actual destination before the source in-place action.
func (m *RationalManager) Inv(a, out *Rational) error {
	if e := m.Copy(out, a); e != nil {
		return e
	}
	return m.InvInPlace(out)
}

// InvInPlace retains the source debug assertion and release finite zero Swap.
// A resulting denominator0 is invalid for ordinary reads but remains owned and
// can be destroyed or explicitly Reset using cleanup-only validation.
func (m *RationalManager) InvInPlace(a *Rational) error {
	if e := m.check(a); e != nil {
		return e
	}
	if modularSourceDebug && a.num.val == 0 {
		return ErrIntegerAssertion
	}
	if a.num.val < 0 {
		if e := m.z.Neg(&a.num); e != nil {
			return e
		}
		if e := m.z.Neg(&a.den); e != nil {
			return e
		}
	}
	return m.z.Swap(&a.num, &a.den)
}

func (m *RationalManager) Equal(a, b *Rational) (bool, error) {
	if e := m.checkScalarValue(a); e != nil {
		return false, e
	}
	if e := m.checkScalarValue(b); e != nil {
		return false, e
	}
	n, e := m.z.Compare(&a.num, &b.num)
	if e != nil || n != 0 {
		return false, e
	}
	d, e := m.z.Compare(&a.den, &b.den)
	return d == 0, e
}

// Compare preserves source rat_lt's sign shortcuts and exact cross-products.
func (m *RationalManager) Compare(a, b *Rational) (_rawValue int, _rawErr error) {
	if e := m.checkScalarValue(a); e != nil {
		return 0, e
	}
	if e := m.checkScalarValue(b); e != nil {
		return 0, e
	}
	if isIntegerRational(a) && isIntegerRational(b) {
		return m.z.Compare(&a.num, &b.num)
	}
	sa, sb := a.num.val, b.num.val
	if sa < 0 && sb >= 0 {
		return -1, nil
	}
	if sa == 0 {
		if sb > 0 {
			return -1, nil
		}
		if sb < 0 {
			return 1, nil
		}
		return 0, nil
	}
	if sa > 0 && sb <= 0 {
		return 1, nil
	}
	if sa < 0 && sb == 0 {
		return -1, nil
	}
	one, two := &m.ltTmp1, &m.ltTmp2
	if m.synchronized {
		r := &m.rawArithmetic
		if r.compareActive {
			return 0, ErrRationalRawArithmeticPending
		}
		r.compareOne, r.compareTwo = NewRational(), NewRational()
		r.compareActive = true
		r.compareEnd = false
		r.compareGuard = false
		r.compareCursor = 0
		r.compareCause = nil
		one, two = &r.compareOne, &r.compareTwo
		defer m.finishRawCompare(&_rawErr)
	}
	if e := m.z.Mul(&a.num, &b.den, &one.num); e != nil {
		return 0, e
	}
	if e := m.resetDenominator(one); e != nil {
		return 0, e
	}
	if e := m.z.Mul(&b.num, &a.den, &two.num); e != nil {
		return 0, e
	}
	if e := m.resetDenominator(two); e != nil {
		return 0, e
	}
	value, e := m.z.Compare(&one.num, &two.num)
	if e != nil {
		return 0, e
	}
	if m.synchronized {
		if e := m.rawCompareEnd(); e != nil {
			return 0, e
		}
	}
	return value, nil
}

func (m *RationalManager) Add(a, b, out *Rational) error { return m.addSub(a, b, out, false) }
func (m *RationalManager) Sub(a, b, out *Rational) error { return m.addSub(a, b, out, true) }
func (m *RationalManager) addSub(a, b, out *Rational, subtract bool) (_rawErr error) {
	for _, n := range []*Rational{a, b, out} {
		if e := m.checkScalarValue(n); e != nil {
			return e
		}
	}
	if isIntegerRational(a) && isIntegerRational(b) {
		var e error
		if subtract {
			e = m.z.Sub(&a.num, &b.num, &out.num)
		} else {
			e = m.z.Add(&a.num, &b.num, &out.num)
		}
		if e != nil {
			return e
		}
		if e = m.resetDenominator(out); e != nil {
			return e
		}
		return nil
	}
	one, two := &m.addTmp1, &m.addTmp2
	if m.synchronized {
		r := &m.rawArithmetic
		if r.addActive {
			return ErrRationalRawArithmeticPending
		}
		generation, e := m.beginRawGeneration()
		if e != nil {
			return e
		}
		r.addGeneration = generation
		r.addOne = Int{}
		r.addTwo = Int{}
		r.addActive = true
		r.addEnd = false
		r.addGuard = false
		r.addCursor = 0
		r.addCause = nil
		one, two = &r.addOne, &r.addTwo
		defer m.finishRawAdd(&_rawErr)
	}
	if e := m.z.Mul(&a.num, &b.den, one); e != nil {
		return e
	}
	if e := m.z.Mul(&b.num, &a.den, two); e != nil {
		return e
	}
	if e := m.z.Mul(&a.den, &b.den, &out.den); e != nil {
		return e
	}
	var e error
	if subtract {
		e = m.z.Sub(one, two, &out.num)
	} else {
		e = m.z.Add(one, two, &out.num)
	}
	if e != nil {
		return e
	}
	if e = m.normalize(out); e != nil {
		return e
	}
	if m.synchronized {
		return m.rawAddEnd()
	}
	return nil
}
func (m *RationalManager) Mul(a, b, out *Rational) error {
	for _, n := range []*Rational{a, b, out} {
		if e := m.checkScalarValue(n); e != nil {
			return e
		}
	}
	// Source chooses this branch before output aliases can mutate operands.
	integers, e := m.IsInt(a)
	if e != nil {
		return e
	}
	if integers {
		integers, e = m.IsInt(b)
		if e != nil {
			return e
		}
	}
	if e := m.z.Mul(&a.num, &b.num, &out.num); e != nil {
		return e
	}
	if integers {
		return m.resetDenominator(out)
	}
	if e := m.z.Mul(&a.den, &b.den, &out.den); e != nil {
		return e
	}
	return m.normalize(out)
}

// Div preserves the actual divisor/destination branch and component order.
// Errors retain completed caller fields; no staged pair masks normalization.
func (m *RationalManager) Div(a, b, out *Rational) (_rawErr error) {
	for _, n := range []*Rational{a, b, out} {
		if e := m.checkScalarValue(n); e != nil {
			return e
		}
	}
	if b == out {
		r := &m.rawArithmetic
		if r.divActive {
			return ErrRationalRawArithmeticPending
		}
		r.div = Int{}
		r.divActive, r.divEnd, r.divGuard = true, false, false
		r.divCursor, r.divCause = 0, nil
		defer m.finishRawDiv(&_rawErr)
		if e := m.z.Mul(&a.num, &b.den, &r.div); e != nil {
			return e
		}
		if e := m.z.Mul(&a.den, &b.num, &out.den); e != nil {
			return e
		}
		if e := m.z.Copy(&out.num, &r.div); e != nil {
			return e
		}
		if e := m.rawDivEnd(); e != nil {
			return e
		}
	} else {
		if e := m.z.Mul(&a.num, &b.den, &out.num); e != nil {
			return e
		}
		if e := m.z.Mul(&a.den, &b.num, &out.den); e != nil {
			return e
		}
	}
	if out.den.val < 0 {
		if e := m.z.Neg(&out.num); e != nil {
			return e
		}
		if e := m.z.Neg(&out.den); e != nil {
			return e
		}
	}
	return m.normalize(out)
}
