// Source: src/util/buffer.h:29-75,90-96,146-151,194-210;
// src/util/rational.h:41-43,67;
// src/ast/rewriter/poly_rewriter_def.h:570-586.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// RationalBuffer16 specializes the source buffer<rational> operations used by
// polynomial coefficient collection. The inline slots are initially raw storage,
// not sixteen constructed rationals. Each successful PushCopy constructs one.
// The actual Go owner, inline storage and exact append invocation are charged.
type RationalBuffer16 struct {
	self                                         *RationalBuffer16
	service                                      *RationalService
	inline                                       [16]OwnedRational
	values                                       []OwnedRational
	appendCall                                   RationalServiceInvocation
	count, capacity, sourceCursor, cleanupCursor uint32
	header, bytes                                uint64
	pending, stopped, sourceEnded, ownStarted    bool
	falseDone, trueDone, closed                  bool
	bodyCause, sourceCause, ownCause             error
	childSource                                  error
}

func NewRationalBuffer16(s *RationalService) (*RationalBuffer16, error) {
	if e := s.lock(); e != nil {
		return nil, e
	}
	s.mu.Unlock()
	header := uint64(reflect.TypeOf(RationalBuffer16{}).Size())
	if e := s.Manager.BudgetField.Reserve(header); e != nil {
		return nil, e
	}
	b := &RationalBuffer16{service: s, capacity: 16, header: header}
	b.self = b
	return b, nil
}

func (b *RationalBuffer16) check() error {
	if b == nil || b.self != b {
		return ErrScopedCopy
	}
	if b.closed {
		return arena.ErrClosed
	}
	return nil
}

func (b *RationalBuffer16) open() error {
	if e := b.check(); e != nil {
		return e
	}
	if b.sourceEnded || b.ownStarted {
		return arena.ErrClosed
	}
	if b.stopped || b.sourceCause != nil || b.sourceCursor != 0 {
		return ErrRationalLexicalPending
	}
	return nil
}

func (b *RationalBuffer16) slot(i uint32) *OwnedRational {
	if b.values == nil {
		return &b.inline[i]
	}
	return &b.values[i]
}

func (b *RationalBuffer16) Len() (uint32, error) {
	if e := b.open(); e != nil {
		return 0, e
	}
	return b.count, nil
}

// At borrows the physical coefficient until expansion or source/owning end.
// Callers must not keep this address across PushCopy, or copy its owning token.
func (b *RationalBuffer16) At(i uint32) (*OwnedRational, error) {
	if e := b.open(); e != nil {
		return nil, e
	}
	if i >= b.count {
		return nil, arena.ErrBounds
	}
	return b.slot(i), nil
}

func (b *RationalBuffer16) expand() error {
	capacity := uint64(b.capacity) * 2
	width := uint64(reflect.TypeOf(OwnedRational{}).Size())
	if capacity > uint64(^uint32(0)) || capacity > uint64(^uint(0)>>1)/width {
		return arena.ErrBounds
	}
	bytes := capacity * width
	if e := b.service.Manager.BudgetField.Reserve(bytes); e != nil {
		return e
	}
	values := make([]OwnedRational, int(capacity))
	// Source memcpy relocates the owning fields. Erase the abandoned Go tokens
	// without numerical DT/copy/move, leaving exactly one owner for each value.
	for i := uint32(0); i < b.count; i++ {
		old := b.slot(i)
		values[i] = *old
		*old = OwnedRational{}
	}
	b.service.Manager.BudgetField.Release(b.bytes)
	b.values, b.capacity, b.bytes = values, uint32(capacity), bytes
	return nil
}

// PushCopy increments the source position only after the copy constructor
// returns. A failed constructor retains its physical slot and exact child for
// owning cleanup, and rejects further body work.
func (b *RationalBuffer16) PushCopy(value *OwnedRational) (result error) {
	if e := b.open(); e != nil {
		return e
	}
	defer func() {
		if result != nil {
			b.stopped = true
			b.bodyCause = result
			b.childSource = b.appendCall.SourceReady()
		}
	}()
	if b.count >= b.capacity {
		if e := b.expand(); e != nil {
			return e
		}
	}
	v := b.slot(b.count)
	var e error
	*v, e = b.service.NewValue()
	if e != nil {
		return e
	}
	b.pending = true
	if e = b.service.CopyWithInvocation(v, value, &b.appendCall); e != nil {
		return e
	}
	b.count++
	b.pending = false
	if e = b.appendCall.SourceReady(); e != nil {
		return e
	}
	if e = b.appendCall.Cleanup(false); e != nil {
		return e
	}
	if e = b.appendCall.Cleanup(true); e != nil {
		return e
	}
	if e = b.appendCall.EndContext(); e != nil {
		return e
	}
	b.appendCall = RationalServiceInvocation{}
	return nil
}

// SourceEnd executes the ascending element DT and then releases expanded
// backing. Failed body calls cannot establish a returned native source scope;
// use owning Cleanup/Close for those prefixes instead of simulating unwinding.
func (b *RationalBuffer16) SourceEnd() error {
	if e := b.check(); e != nil {
		return e
	}
	if b.sourceEnded {
		return nil
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.ownStarted || b.stopped {
		return ErrRationalLexicalPending
	}
	if e := b.appendCall.SourceReady(); e != nil {
		b.sourceCause = e
		return e
	}
	for b.sourceCursor < b.count {
		if e := b.service.Del(b.slot(b.sourceCursor)); e != nil {
			b.sourceCause = e
			return e
		}
		b.sourceCursor++
	}
	b.releaseBacking()
	b.sourceEnded = true
	return nil
}

func (b *RationalBuffer16) releaseBacking() {
	b.service.Manager.BudgetField.Release(b.bytes)
	b.bytes = 0
	b.values = nil
}

// Cleanup joins this buffer's exact append child. It never adopts another
// caller's numerical frame. Historical body/source failures remain unchanged.
func (b *RationalBuffer16) Cleanup(abort bool) (result error) {
	if e := b.check(); e != nil {
		return e
	}
	if abort && !b.falseDone {
		return ErrRationalLexicalPending
	}
	if !abort && b.falseDone || abort && b.trueDone {
		return nil
	}
	b.ownStarted = true
	defer func() {
		if result != nil && b.ownCause == nil {
			b.ownCause = result
		}
	}()
	if e := b.appendCall.Cleanup(abort); e != nil {
		return e
	}
	if abort {
		b.trueDone = true
		return nil
	}
	// Source-consumed values have already been cleared. A failed placement
	// constructor is outside count but its real fields also require cleanup.
	if !b.sourceEnded {
		end := b.count
		if b.pending {
			end++
		}
		for b.cleanupCursor < end {
			v := b.slot(b.cleanupCursor)
			if v.ownerID != 0 {
				if e := b.service.CleanupValue(v); e != nil {
					return e
				}
			}
			b.cleanupCursor++
		}
	}
	b.falseDone = true
	return nil
}

// Causes reports immutable source/body history, including after owning close.
func (b *RationalBuffer16) Causes() (body, source, own error) {
	if b == nil || b.self != b {
		return ErrScopedCopy, nil, nil
	}
	return b.bodyCause, JoinNumericErrors(b.childSource, b.sourceCause), b.ownCause
}

// Close finishes Go ownership. A successful source caller first calls SourceEnd;
// failed source calls use Close without claiming native unwinding or resumability.
func (b *RationalBuffer16) Close() error {
	if b == nil || b.self != b {
		return ErrScopedCopy
	}
	if b.closed {
		return nil
	}
	if e := b.Cleanup(false); e != nil {
		return e
	}
	if e := b.Cleanup(true); e != nil {
		return e
	}
	if e := b.appendCall.EndContext(); e != nil {
		return e
	}
	b.releaseBacking()
	b.service.Manager.BudgetField.Release(b.header)
	b.header = 0
	b.closed = true
	return nil
}
