// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpq.cpp:130-141; src/util/mpz.cpp:1385-1410.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

// NumeralSourceFailure records a reached lexical source destructor stop.
// The body cause and first destructor cause remain separate through owning ACK.
type NumeralSourceFailure struct{ BodyCause, DestructorCause error }

func (e *NumeralSourceFailure) Error() string {
	return "numeric ostream source destructor stopped: " + e.DestructorCause.Error()
}
func (e *NumeralSourceFailure) Unwrap() []error { return []error{e.BodyCause, e.DestructorCause} }

// RationalSourceStream is a call-local borrowed ostream bridge, never stored in
// hot numeric records. Each method is one actual source insertion.
type RationalSourceStream interface {
	AppendInt32(int32) error
	AppendCString(string) error
	AppendNumeralCString(NumeralCStringView) error
}

// NumeralCStringView borrows the exact completed lexical char buffer invocation.
// It performs reached byte reads without building a host string or assembly.
type NumeralCStringView struct {
	owner      *NaturalManager
	invocation uint64
}

func (v NumeralCStringView) Byte(i uint64) (byte, error) {
	if v.owner == nil || v.owner.self != v.owner || v.owner.closed {
		return 0, arena.ErrInvalidSlot
	}
	s := &v.owner.charFormatting
	if !s.active || !s.completed || s.invocation != v.invocation {
		return 0, arena.ErrInvalidSlot
	}
	return s.buffer.readPhysical(v.owner.runtime, i)
}

func (s *RationalService) InsertInto(out RationalSourceStream, v *OwnedRational) error {
	return s.InsertIntoTracked(out, v, nil)
}
func (s *RationalService) InsertIntoTracked(out RationalSourceStream, v *OwnedRational, receipt *NumeralInsertionReceipt) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	s.sourceInsertionExecuting = true
	defer func() { s.sourceInsertionExecuting = false }()
	if e := s.check(v); e != nil {
		return e
	}
	if out == nil {
		return arena.ErrConfig
	}
	if e := s.manager.z.insertSourceIntegerTracked(out, &v.raw.num, s, receipt); e != nil {
		return e
	}
	if isIntegerRational(&v.raw) {
		return nil
	}
	if e := out.AppendCString("/"); e != nil {
		return e
	}
	return s.manager.z.insertSourceIntegerTracked(out, &v.raw.den, s, receipt)
}
func (m *IntegerManager) insertSourceInteger(out RationalSourceStream, a *Int) error {
	return m.insertSourceIntegerTracked(out, a, nil, nil)
}
func (m *IntegerManager) insertSourceIntegerTracked(out RationalSourceStream, a *Int, service *RationalService, receipt *NumeralInsertionReceipt) (result error) {
	if e := m.check(a); e != nil {
		return e
	}
	if a.IsSmall() {
		return out.AppendInt32(a.val)
	}
	if a.val < 0 {
		if e := out.AppendCString("-"); e != nil {
			return e
		}
	}
	c := m.cell(a.cell)
	size, e := c.backing.readWord(m.allocator, 0)
	if e != nil {
		return e
	}
	n := m.natural
	if receipt != nil && receipt.active {
		return arena.ErrInvalidSlot
	}
	if e = n.beginIntegerCharactersTracked(uint32(11*size), service, receipt); e != nil {
		if receipt != nil && receipt.active {
			receipt.bodyCause = e
			receipt.sourceDone = true
		}
		return e
	}
	defer func() {
		if receipt != nil && receipt.active {
			if result != nil && receipt.bodyCause == nil {
				receipt.bodyCause = result
			}
			var stopped *NumeralSourceFailure
			if errors.As(result, &stopped) {
				receipt.sourceCause = stopped.DestructorCause
				return
			}
			receipt.sourceDone = true
			if result == nil {
				receipt.active = false
				receipt.retired = true
			}
		}
	}()
	e = n.sourceToCStringTracked(limbView{allocator: m.allocator, backing: c.backing, size: size}, sourceDecimalDestination{character: &n.charFormatting.buffer, assembly: n}, uint64(n.charFormatting.buffer.pos), receipt)
	if e == nil {
		e = out.AppendNumeralCString(NumeralCStringView{n, n.charFormatting.invocation})
	}
	if e != nil && n.charFormatting.originalCause == nil {
		n.charFormatting.originalCause = e
	}
	// A younger source local that stopped destruction protects the older char
	// buffer. Failed construction likewise has no source destructor.
	var stopped *NumeralSourceFailure
	if errors.As(e, &stopped) {
		return e
	}
	if cleanup := n.destroyIntegerCharacters(); cleanup != nil {
		return &NumeralSourceFailure{BodyCause: e, DestructorCause: cleanup}
	}
	if receipt != nil {
		receipt.characterOwned = false
	}
	return e
}
