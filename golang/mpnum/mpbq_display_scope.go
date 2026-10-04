// Licensed under the MIT license.
// Source mpbq.cpp426–515.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

var ErrBinaryDisplayPending = errors.New("mpnum: raw binary display locals pending")

// Raw mpz locals have no automatic source destructor. A thrown operation does
// not jump to the source end label. These actual owning scalar records and cold
// cursors are charged in the BinaryRationalManager header.
type binaryDisplayScope struct {
	values                                      [6]Int // n1,v1,n2,v2,two_k1,two_k2
	active, interval, sourceEnd, guard, aborted bool
	sourceStage, owningStage                    uint8
	original, cleanup                           error
}
type BinaryDisplayState struct {
	Pending, SourceEnd, Guard, OwningAbort bool
	SourceStage, OwningStage               uint8
	OriginalCause, CleanupCause            error
}

func (m *BinaryRationalManager) DisplayState() BinaryDisplayState {
	if m == nil || m.self != m {
		return BinaryDisplayState{CleanupCause: ErrBinaryOwner}
	}
	s := &m.display
	return BinaryDisplayState{s.active, s.sourceEnd, s.guard, s.aborted, s.sourceStage, s.owningStage, s.original, s.cleanup}
}
func (m *BinaryRationalManager) beginBinaryDisplay(interval bool) error {
	if m.display.active {
		return ErrBinaryDisplayPending
	}
	m.display = binaryDisplayScope{active: true, interval: interval}
	return nil
}
func (s *binaryDisplayScope) count() uint8 {
	if s.interval {
		return 6
	}
	return 3
}
func (s *binaryDisplayScope) target(stage uint8) *Int {
	if !s.interval && stage == 2 {
		return &s.values[4]
	}
	return &s.values[stage]
}
func (m *BinaryRationalManager) sourceBinaryDisplayEnd() error {
	s := &m.display
	for s.sourceStage < s.count() {
		guard, e := sourceNumeralDel(m.z, s.target(s.sourceStage))
		if guard {
			s.guard = true
		}
		if e != nil {
			s.cleanup = e
			return e
		}
		s.sourceStage++
	}
	s.active = false
	s.cleanup = nil
	return nil
}
func (m *BinaryRationalManager) finishBinaryDisplay(length *uint32, result *error) {
	s := &m.display
	if !s.active {
		return
	}
	if *result != nil {
		s.original = *result
		return
	} // no source end label on exception
	s.sourceEnd = true
	if e := m.sourceBinaryDisplayEnd(); e != nil {
		*length = 0
		*result = e
	}
}

// CleanupDisplay is explicit owning cleanup of abandoned raw locals. It first
// completes real inner formatter roots, then either resumes the actual end-label
// cursor or releases retained raw locals without claiming a source end label.
// Failed-source-pointer guard history remains distinct from this owning cursor.
func (m *BinaryRationalManager) CleanupDisplay() error {
	if m == nil || m.self != m {
		return ErrBinaryOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	s := &m.display
	if !s.active {
		return nil
	}
	if e := m.z.natural.cleanupOutputFormatting(); e != nil {
		s.cleanup = e
		return e
	}
	if s.sourceEnd && !s.guard {
		return m.sourceBinaryDisplayEnd()
	}
	s.aborted = true
	for s.owningStage < s.count() {
		if e := m.z.Del(s.target(s.owningStage)); e != nil {
			s.cleanup = e
			return e
		}
		s.owningStage++
	}
	s.active = false
	s.cleanup = nil
	return nil
}
