// Source: src/util/mpn.cpp:360-399; src/util/buffer.h.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Ephemeral by-value borrowed destinations; no interface, slice or pointer is
// persisted in a hot record, and no destination owner escapes per call.

type sourceDecimalDestination struct {
	character []byte
	external  []byte
	assembly  *NaturalManager
	offset    uint32
}

func (d sourceDecimalDestination) write(i uint64, n byte) error {
	if d.character != nil {
		if i >= uint64(len(d.character)) {
			return arena.ErrBounds
		}
		d.character[i] = n
		return nil
	}
	if d.assembly != nil {
		if i > uint64(^uint32(0))-uint64(d.offset) {
			return arena.ErrBounds
		}
		index := uint64(d.offset) + i
		if index >= uint64(len(d.assembly.text)) {
			return arena.ErrBounds
		}
		d.assembly.text[index] = n
		return nil
	}
	if i >= uint64(len(d.external)) {
		return arena.ErrBounds
	}
	d.external[int(i)] = n
	return nil
}
func (d sourceDecimalDestination) read(i uint64) (byte, error) {
	if d.character != nil {
		return characterByte(d.character, i)
	}
	if d.assembly != nil {
		if i > uint64(^uint32(0))-uint64(d.offset) {
			return 0, arena.ErrBounds
		}
		index := uint64(d.offset) + i
		if index >= uint64(len(d.assembly.text)) {
			return 0, arena.ErrBounds
		}
		return d.assembly.text[index], nil
	}
	if i >= uint64(len(d.external)) {
		return 0, arena.ErrBounds
	}
	return d.external[int(i)], nil
}
func readDecimalLimb(v limbView, i uint32) (uint32, error) {
	if i >= v.size {
		return 0, arena.ErrBounds
	}
	if v.source != nil {
		return v.source.read(v.runtime, i)
	}
	if v.allocator != nil {
		return v.backing.ReadWord(v.allocator, 8+4*uint64(i))
	}

	if uint64(i) >= uint64(len(v.external)) {
		return 0, arena.ErrBounds
	}
	return v.external[i], nil
}
func (m *NaturalManager) sourceToCString(a limbView, dst sourceDecimalDestination, lbuf uint64) error {
	return m.sourceToCStringTracked(a, dst, lbuf, nil)
}
func (m *NaturalManager) sourceToCStringTracked(a limbView, dst sourceDecimalDestination, lbuf uint64, receipt *NumeralInsertionReceipt) error {
	return m.sourceToCStringScopedTracked(a, dst, lbuf, &m.Formatting, receipt)
}
func (m *NaturalManager) sourceToCStringScoped(a limbView, dst sourceDecimalDestination, lbuf uint64, scope *naturalFormattingScope) error {
	return m.sourceToCStringScopedTracked(a, dst, lbuf, scope, nil)
}
func (m *NaturalManager) sourceToCStringScopedTracked(a limbView, dst sourceDecimalDestination, lbuf uint64, scope *naturalFormattingScope, receipt *NumeralInsertionReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if SourceExtendedGCDDebug && (lbuf == 0 || (dst.character == nil && dst.assembly == nil && dst.external == nil)) {
		return ErrIntegerAssertion
	}
	if a.size == 1 {
		value, e := readDecimalLimb(a, 0)
		if e != nil {
			return e
		}
		var digits [10]byte
		n := 0
		for {
			digits[n] = '0' + byte(value%10)
			n++
			value /= 10
			if value == 0 {
				break
			}
		}
		// snprintf's zero-size case writes nothing. Its return count is ignored
		// in source; Cptr insertion subsequently reads the existing destination.
		if lbuf == 0 {
			return nil
		}
		count := uint64(n)
		if count >= lbuf {
			count = lbuf - 1
		}
		for i := uint64(0); i < count; i++ {
			if e = dst.write(i, digits[n-1-int(i)]); e != nil {
				return e
			}
		}
		return dst.write(count, 0)
	}
	if e := m.beginFormattingScopeTracked(scope, a.size, receipt); e != nil {
		return e
	}
	e := m.sourceCStringBodyScoped(a, dst, scope)
	cleanup := m.finishFormattingScope(scope)
	if cleanup == nil && receipt != nil {
		receipt.formattingOwned = false
	}
	if e != nil {
		if cleanup != nil {
			scope.originalCause = e
			scope.cleanupCause = cleanup
			return &NumeralSourceFailure{BodyCause: e, DestructorCause: cleanup}
		}
		return e
	}
	return cleanup
}
func (m *NaturalManager) sourceCStringBody(a limbView, dst sourceDecimalDestination) error {
	return m.sourceCStringBodyScoped(a, dst, &m.Formatting)
}
func (m *NaturalManager) sourceCStringBodyScoped(a limbView, dst sourceDecimalDestination, scope *naturalFormattingScope) error {
	temp := limbView{external: scope.Locals[0], size: a.size}
	numer := limbView{external: scope.Locals[1], size: a.size + 1}
	denom := limbView{external: scope.Locals[2], size: 1}
	for i := uint32(0); i < a.size; i++ {
		v, e := readDecimalLimb(a, i)
		if e != nil {
			return e
		}
		temp.write(i, v)
	}
	const d = 28
	denom.write(0, uint32(10)<<d)
	var j uint64
	for temp.size > 0 && (temp.size > 1 || temp.read(0) != 0) {
		numer.size = temp.size + 1
		numer.write(temp.size, temp.read(temp.size-1)>>(32-d))
		for i := temp.size - 1; i > 0; i-- {
			numer.write(i, temp.read(i)<<d|temp.read(i-1)>>(32-d))
		}
		numer.write(0, temp.read(0)<<d)
		divideViewOne(numer, denom.read(0), temp)
		if e := dst.write(j, '0'+byte(numer.read(0)>>d)); e != nil {
			return e
		}
		j++
		if m.runtime.SizeTBits() == 32 {
			j = uint64(uint32(j))
		}
		for temp.size > 0 && temp.read(temp.size-1) == 0 {
			temp.size--
		}
	}
	if e := dst.write(j, 0); e != nil {
		return e
	}
	// Preserve source size_t underflow and guard only its actual endangered
	// reversal access, while all completed inner local objects remain live.
	j--
	if m.runtime.SizeTBits() == 32 {
		j = uint64(uint32(j))
	}
	mid := j/2 + j%2
	for i := uint64(0); i < mid; i++ {
		left, e := dst.read(i)
		if e != nil {
			return e
		}
		right, e := dst.read(j - i)
		if e != nil {
			return e
		}
		if e = dst.write(i, right); e != nil {
			return e
		}
		if e = dst.write(j-i, left); e != nil {
			return e
		}
	}
	return nil
}
