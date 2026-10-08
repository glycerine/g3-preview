// Source: src/util/ext_numeral.h:25-345.
package util

import "errors"

type ExtNumeralKind int32

const (
	ExtMinusInfinity ExtNumeralKind = iota
	ExtFinite
	ExtPlusInfinity
)

// ExtNumeralSlot names the borrowed source operand in a call-local binding.
// A manager implementation keeps its concrete numeral pointers; no numeral
// payload is boxed in an interface or copied into an extended-number record.
type ExtNumeralSlot uint8

const (
	ExtA ExtNumeralSlot = iota
	ExtB
	ExtC
)

// ExtNumeralManager is the non-generic translation of the source manager
// concept. Callers bind A/B/C to actual concrete operands and output. Methods
// are reached only where the source reads/calls that manager; infinity branches
// may erase the binding entirely. Concrete manager bindings are separate from
// the shared branch algorithms, and retain their existing allocation ownership.
type ExtNumeralManager interface {
	Field() bool
	IsZero(ExtNumeralSlot) (bool, error)
	IsPos(ExtNumeralSlot) (bool, error)
	IsNeg(ExtNumeralSlot) (bool, error)
	Set(ExtNumeralSlot, ExtNumeralSlot) error
	Reset(ExtNumeralSlot) error
	Neg(ExtNumeralSlot) error
	Inv(ExtNumeralSlot) error
	Add() error
	Sub() error
	Mul() error
	Div() error
	Power(ExtNumeralSlot, uint32) error
	Eq(ExtNumeralSlot, ExtNumeralSlot) (bool, error)
	Lt(ExtNumeralSlot, ExtNumeralSlot) (bool, error)
	Display(*NumericTextStream, ExtNumeralSlot, bool) error
}

var ErrExtNumeralAssertion = errors.New("g3: ext_numeral source assertion")
var ErrExtNumeralKind = errors.New("g3: reached unreachable ext_numeral kind")

func extZero(m ExtNumeralManager, s ExtNumeralSlot, k ExtNumeralKind) (bool, error) {
	if k != ExtFinite {
		return false, nil
	}
	return m.IsZero(s)
}
func extPos(m ExtNumeralManager, s ExtNumeralSlot, k ExtNumeralKind) (bool, error) {
	if k == ExtPlusInfinity {
		return true, nil
	}
	if k != ExtFinite {
		return false, nil
	}
	return m.IsPos(s)
}
func ExtIsZero(m ExtNumeralManager, k ExtNumeralKind) (bool, error) { return extZero(m, ExtA, k) }
func ExtIsPos(m ExtNumeralManager, k ExtNumeralKind) (bool, error)  { return extPos(m, ExtA, k) }
func ExtIsNeg(m ExtNumeralManager, k ExtNumeralKind) (bool, error) {
	if k == ExtMinusInfinity {
		return true, nil
	}
	if k != ExtFinite {
		return false, nil
	}
	return m.IsNeg(ExtA)
}
func ExtIsInfinite(k ExtNumeralKind) bool { return k != ExtFinite }
func ExtSet(m ExtNumeralManager, ak *ExtNumeralKind, bk ExtNumeralKind) error {
	if e := m.Set(ExtA, ExtB); e != nil {
		return e
	}
	*ak = bk
	return nil
}
func ExtReset(m ExtNumeralManager, ak *ExtNumeralKind) error {
	if e := m.Reset(ExtA); e != nil {
		return e
	}
	*ak = ExtFinite
	return nil
}
func ExtNeg(m ExtNumeralManager, ak *ExtNumeralKind) error {
	switch *ak {
	case ExtMinusInfinity:
		*ak = ExtPlusInfinity
	case ExtFinite:
		return m.Neg(ExtA)
	case ExtPlusInfinity:
		*ak = ExtMinusInfinity
	}
	return nil
}
func ExtInv(m ExtNumeralManager, ak *ExtNumeralKind) error {
	if sourceDebug && !m.Field() {
		return ErrExtNumeralAssertion
	}
	switch *ak {
	case ExtMinusInfinity, ExtPlusInfinity:
		*ak = ExtFinite
		return m.Reset(ExtA)
	case ExtFinite:
		if sourceDebug {
			zero, e := m.IsZero(ExtA)
			if e != nil {
				return e
			}
			if zero {
				return ErrExtNumeralAssertion
			}
		}
		return m.Inv(ExtA)
	}
	return nil
}
func ExtAdd(m ExtNumeralManager, ak, bk ExtNumeralKind, ck *ExtNumeralKind) error {
	if sourceDebug && (ak == ExtMinusInfinity && bk == ExtPlusInfinity || ak == ExtPlusInfinity && bk == ExtMinusInfinity) {
		return ErrExtNumeralAssertion
	}
	if ak != ExtFinite {
		if e := m.Reset(ExtC); e != nil {
			return e
		}
		*ck = ak
	} else if bk != ExtFinite {
		if e := m.Reset(ExtC); e != nil {
			return e
		}
		*ck = bk
	} else {
		if e := m.Add(); e != nil {
			return e
		}
		*ck = ExtFinite
	}
	return nil
}
func ExtSub(m ExtNumeralManager, ak, bk ExtNumeralKind, ck *ExtNumeralKind) error {
	if sourceDebug && (ak == ExtMinusInfinity && bk == ExtMinusInfinity || ak == ExtPlusInfinity && bk == ExtPlusInfinity) {
		return ErrExtNumeralAssertion
	}
	if ak != ExtFinite {
		if sourceDebug && bk == ak {
			return ErrExtNumeralAssertion
		}
		if e := m.Reset(ExtC); e != nil {
			return e
		}
		*ck = ak
	} else {
		switch bk {
		case ExtMinusInfinity:
			if e := m.Reset(ExtC); e != nil {
				return e
			}
			*ck = ExtPlusInfinity
		case ExtFinite:
			if e := m.Sub(); e != nil {
				return e
			}
			*ck = ExtFinite
		case ExtPlusInfinity:
			if e := m.Reset(ExtC); e != nil {
				return e
			}
			*ck = ExtMinusInfinity
		}
	}
	return nil
}
func ExtMul(m ExtNumeralManager, ak, bk ExtNumeralKind, ck *ExtNumeralKind) error {
	zero, e := extZero(m, ExtA, ak)
	if e != nil {
		return e
	}
	if !zero {
		zero, e = extZero(m, ExtB, bk)
		if e != nil {
			return e
		}
	}
	if zero {
		if e = m.Reset(ExtC); e != nil {
			return e
		}
		*ck = ExtFinite
	} else if ExtIsInfinite(ak) || ExtIsInfinite(bk) {
		apos, e := extPos(m, ExtA, ak)
		if e != nil {
			return e
		}
		bpos, e := extPos(m, ExtB, bk)
		if e != nil {
			return e
		}
		if apos == bpos {
			*ck = ExtPlusInfinity
		} else {
			*ck = ExtMinusInfinity
		}
		return m.Reset(ExtC)
	} else {
		*ck = ExtFinite
		return m.Mul()
	}
	return nil
}
func ExtDiv(m ExtNumeralManager, ak, bk ExtNumeralKind, ck *ExtNumeralKind) error {
	if sourceDebug {
		zero, e := extZero(m, ExtB, bk)
		if e != nil {
			return e
		}
		if zero {
			return ErrExtNumeralAssertion
		}
	}
	zero, e := extZero(m, ExtA, ak)
	if e != nil {
		return e
	}
	if zero {
		if sourceDebug {
			zero, e = extZero(m, ExtB, bk)
			if e != nil {
				return e
			}
			if zero {
				return ErrExtNumeralAssertion
			}
		}
		if e = m.Reset(ExtC); e != nil {
			return e
		}
		*ck = ExtFinite
	} else if ExtIsInfinite(ak) {
		if sourceDebug && ExtIsInfinite(bk) {
			return ErrExtNumeralAssertion
		}
		apos, e := extPos(m, ExtA, ak)
		if e != nil {
			return e
		}
		bpos, e := extPos(m, ExtB, bk)
		if e != nil {
			return e
		}
		if apos == bpos {
			*ck = ExtPlusInfinity
		} else {
			*ck = ExtMinusInfinity
		}
		return m.Reset(ExtC)
	} else if ExtIsInfinite(bk) {
		if sourceDebug && ExtIsInfinite(ak) {
			return ErrExtNumeralAssertion
		}
		if e = m.Reset(ExtC); e != nil {
			return e
		}
		*ck = ExtFinite
	} else {
		*ck = ExtFinite
		return m.Div()
	}
	return nil
}
func ExtPower(m ExtNumeralManager, ak *ExtNumeralKind, n uint32) error {
	switch *ak {
	case ExtMinusInfinity:
		if n%2 == 0 {
			*ak = ExtPlusInfinity
		}
	case ExtFinite:
		return m.Power(ExtA, n)
	case ExtPlusInfinity:
	}
	return nil
}
func ExtEq(m ExtNumeralManager, ak, bk ExtNumeralKind) (bool, error) {
	if ak == ExtFinite {
		if bk != ExtFinite {
			return false, nil
		}
		return m.Eq(ExtA, ExtB)
	}
	return ak == bk, nil
}
func ExtNeq(m ExtNumeralManager, ak, bk ExtNumeralKind) (bool, error) {
	yes, e := ExtEq(m, ak, bk)
	return !yes, e
}
func extLt(m ExtNumeralManager, a, b ExtNumeralSlot, ak, bk ExtNumeralKind) (bool, error) {
	switch ak {
	case ExtMinusInfinity:
		return bk != ExtMinusInfinity, nil
	case ExtFinite:
		switch bk {
		case ExtMinusInfinity:
			return false, nil
		case ExtFinite:
			return m.Lt(a, b)
		case ExtPlusInfinity:
			return true, nil
		default:
			return false, ErrExtNumeralKind
		}
	case ExtPlusInfinity:
		return false, nil
	default:
		return false, ErrExtNumeralKind
	}
}
func ExtLt(m ExtNumeralManager, ak, bk ExtNumeralKind) (bool, error) {
	return extLt(m, ExtA, ExtB, ak, bk)
}
func ExtGt(m ExtNumeralManager, ak, bk ExtNumeralKind) (bool, error) {
	return extLt(m, ExtB, ExtA, bk, ak)
}
func ExtLe(m ExtNumeralManager, ak, bk ExtNumeralKind) (bool, error) {
	yes, e := ExtGt(m, ak, bk)
	return !yes, e
}
func ExtGe(m ExtNumeralManager, ak, bk ExtNumeralKind) (bool, error) {
	yes, e := ExtLt(m, ak, bk)
	return !yes, e
}
func ExtDisplay(out *NumericTextStream, m ExtNumeralManager, ak ExtNumeralKind) error {
	return extDisplay(out, m, ak, false)
}
func ExtDisplayPP(out *NumericTextStream, m ExtNumeralManager, ak ExtNumeralKind) error {
	return extDisplay(out, m, ak, true)
}
func extDisplay(out *NumericTextStream, m ExtNumeralManager, k ExtNumeralKind, pretty bool) error {
	switch k {
	case ExtMinusInfinity:
		if pretty {
			return out.AppendCString("-&infin;")
		}
		return out.AppendCString("-oo")
	case ExtFinite:
		return m.Display(out, ExtA, pretty)
	case ExtPlusInfinity:
		if pretty {
			return out.AppendCString("+&infin;")
		}
		return out.AppendCString("+oo")
	}
	return nil
}

// ExtRationalBinding supplies the actual mpq-manager instantiation, including
// in-place source inv/power and raw numeral ostream insertion. The binding is
// call-local borrowed control storage and owns none of its operand pointers.
type ExtRationalBinding struct {
	Manager  *RationalManager
	Operands [3]*Rational
}

func (b *ExtRationalBinding) Field() bool { return true }
func (b *ExtRationalBinding) IsZero(s ExtNumeralSlot) (bool, error) {
	return b.Manager.IsZero(b.Operands[s])
}
func (b *ExtRationalBinding) IsPos(s ExtNumeralSlot) (bool, error) {
	return b.Manager.IsPos(b.Operands[s])
}
func (b *ExtRationalBinding) IsNeg(s ExtNumeralSlot) (bool, error) {
	return b.Manager.IsNeg(b.Operands[s])
}
func (b *ExtRationalBinding) Set(a, c ExtNumeralSlot) error {
	return b.Manager.Copy(b.Operands[a], b.Operands[c])
}
func (b *ExtRationalBinding) Reset(s ExtNumeralSlot) error { return b.Manager.Reset(b.Operands[s]) }
func (b *ExtRationalBinding) Neg(s ExtNumeralSlot) error   { return b.Manager.Neg(b.Operands[s]) }
func (b *ExtRationalBinding) Inv(s ExtNumeralSlot) error {
	return b.Manager.InvInPlace(b.Operands[s])
}
func (b *ExtRationalBinding) Add() error {
	return b.Manager.Add(b.Operands[ExtA], b.Operands[ExtB], b.Operands[ExtC])
}
func (b *ExtRationalBinding) Sub() error {
	return b.Manager.Sub(b.Operands[ExtA], b.Operands[ExtB], b.Operands[ExtC])
}
func (b *ExtRationalBinding) Mul() error {
	return b.Manager.Mul(b.Operands[ExtA], b.Operands[ExtB], b.Operands[ExtC])
}
func (b *ExtRationalBinding) Div() error {
	return b.Manager.Div(b.Operands[ExtA], b.Operands[ExtB], b.Operands[ExtC])
}
func (b *ExtRationalBinding) Power(s ExtNumeralSlot, n uint32) error {
	return b.Manager.Power(b.Operands[s], n, b.Operands[s])
}
func (b *ExtRationalBinding) Eq(a, c ExtNumeralSlot) (bool, error) {
	return b.Manager.Equal(b.Operands[a], b.Operands[c])
}
func (b *ExtRationalBinding) Lt(a, c ExtNumeralSlot) (bool, error) {
	v, e := b.Manager.Compare(b.Operands[a], b.Operands[c])
	return v < 0, e
}
func (b *ExtRationalBinding) Display(out *NumericTextStream, s ExtNumeralSlot, pretty bool) error {
	a := b.Operands[s]
	if e := b.Manager.Z.insertSourceInteger(out, &a.Num); e != nil {
		return e
	}
	if !isIntegerRational(a) {
		if e := out.AppendCString("/"); e != nil {
			return e
		}
		return b.Manager.Z.insertSourceInteger(out, &a.Den)
	}
	return nil
}
