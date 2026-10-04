// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
)

var ErrRationalLexicalPending = errors.New("mpnum: owning lexical cleanup is pending")

// RationalLexicalState describes actual embedded owning records, not allocator
// registry membership. Completed roots precede partial-constructor abort roots.
type RationalLexicalState struct {
	Roots, Completed, Aborts uint8
	Pending                  bool
	Cause, OriginalCause     error
}

// Nine includes the external ext_gcd source stack: six persistent locals,
// coefficient product, completed inner expression work, and partial returned-
// Copy adapter token. Destructor refusal can retain all nine records. Existing
// Inf expression stacks fit eight. All fields are charged by actual owner size.
const rationalLexicalRootLimit = 9

type rationalLexicalScope struct {
	roots                [rationalLexicalRootLimit]Rational
	abort                [rationalLexicalRootLimit]bool
	components           [rationalLexicalRootLimit]uint8
	count, cursor, depth uint8
	done                 uint16
	cause, originalCause error
}

func (l *rationalLexicalScope) pending() bool { return l.cursor < l.count }
func (l *rationalLexicalScope) retain(a *Rational, abort bool) uint8 {
	if a.num.cell == 0 && a.den.cell == 0 {
		*a = Rational{}
		return rationalLexicalRootLimit
	}
	if l.count == rationalLexicalRootLimit {
		panic("mpnum: audited lexical root bound violated")
	}
	index := l.count
	l.roots[index] = *a
	l.abort[index] = abort
	l.count++
	*a = Rational{}
	return index
}
func (l *rationalLexicalScope) state() RationalLexicalState {
	v := RationalLexicalState{Pending: l.pending(), Cause: l.cause, OriginalCause: l.originalCause}
	for i := uint8(0); i < l.count; i++ {
		if l.done&(1<<i) != 0 {
			continue
		}
		v.Roots++
		if l.abort[i] {
			v.Aborts++
		} else {
			v.Completed++
		}
	}
	return v
}
func lexicalDelete(z *IntegerManager, a *Rational, abort bool, component *uint8) error {
	for *component < 2 {
		p := &a.num
		if *component == 1 {
			p = &a.den
		}
		if abort || p.cell == z.failedRef { // explicit Go abort of an invalid failed-copy pointer
			if e := z.Del(p); e != nil {
				return e
			}
		} else {
			if _, e := sourceNumeralDel(z, p); e != nil {
				return e
			}
		}
		*component++
	}
	*a = Rational{}
	return nil
}

// Failed-constructor storage is an owning Go obligation. It is queued until
// every completed source object in the enclosing lexical stack has unwound.
func (l *rationalLexicalScope) dispose(z *IntegerManager, a *Rational, abort bool) error {
	if abort {
		l.retain(a, true)
		return l.cause
	}
	if l.cause != nil {
		l.retain(a, false)
		return l.cause
	}
	var component uint8
	if e := lexicalDelete(z, a, false, &component); e != nil {
		l.cause = e
		index := l.retain(a, false)
		if index < rationalLexicalRootLimit {
			l.components[index] = component
		}
		return e
	}
	return nil
}
func (l *rationalLexicalScope) cleanup(z *IntegerManager) error {
	// Completed source destructor order precedes separate partial-owning aborts,
	// even when a partial inner copy was queued before the outer unwind began.
	for l.pending() {
		index := uint8(rationalLexicalRootLimit)
		for i := uint8(0); i < l.count; i++ {
			if l.done&(1<<i) == 0 && !l.abort[i] {
				index = i
				break
			}
		}
		if index == rationalLexicalRootLimit {
			for i := uint8(0); i < l.count; i++ {
				if l.done&(1<<i) == 0 {
					index = i
					break
				}
			}
		}
		if e := lexicalDelete(z, &l.roots[index], true, &l.components[index]); e != nil {
			l.cause = e
			return e
		}
		l.done |= 1 << index
		for l.cursor < l.count && l.done&(1<<l.cursor) != 0 {
			l.cursor++
		}
	}
	depth := l.depth
	*l = rationalLexicalScope{depth: depth}
	return nil
}
func (s *RationalService) LexicalState() (RationalLexicalState, error) {
	if s == nil || s.self != s {
		return RationalLexicalState{}, ErrRationalServiceCopy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lexical.state(), nil
}
func (s *RationalService) CleanupLexical() error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if s.lexical.depth != 0 {
		return ErrRationalLexicalPending
	}
	return s.lexical.cleanup(s.manager.z)
}
func (s *RationalService) lexicalReady() error {
	if s == nil || s.self != s {
		return ErrRationalServiceCopy
	}
	if s.lexical.pending() {
		return ErrRationalLexicalPending
	}
	return nil
}
func (s *RationalService) destroyLexicalRational(a *Rational) error {
	return s.lexical.dispose(s.manager.z, a, false)
}
func (s *RationalService) abortLexicalRational(a *Rational) error {
	return s.lexical.dispose(s.manager.z, a, true)
}

// Local completion flags are concrete stack scalars; no scope ticket or heap
// context is created for an expression, and no callback/function variable is stored.
func (s *RationalService) beginLexical() error {
	if e := s.lexicalReady(); e != nil {
		return e
	}
	if e := s.manager.checkOwnerReady(); e != nil {
		return e
	}
	if s.lexical.depth == 255 {
		return arena.ErrBounds
	}
	s.lexical.depth++
	return nil
}
func (s *RationalService) endLexicalLocal(a *Rational, completed *bool) {
	if *completed {
		s.destroyLexicalRational(a)
	} else {
		s.abortLexicalRational(a)
	}
}
func (s *RationalService) endLexicalError(e *error) {
	s.finishLexicalError(e)
	if s.lexical.depth > 0 {
		s.lexical.depth--
	}
	if s.lexical.depth == 0 && s.lexical.cause == nil {
		if next := s.lexical.cleanup(s.manager.z); next != nil && *e == nil {
			*e = next
		}
	}
	s.finishLexicalError(e)
}
func (s *RationalService) endLexicalRational(out *OwnedRational, e *error) {
	s.endLexicalError(e)
	s.finishLexicalRational(out, e)
}
func (s *RationalService) endLexicalInf(out *OwnedInfRational, e *error) {
	s.endLexicalError(e)
	s.finishLexicalInf(out, e)
}
func (s *RationalService) endLexicalInfInt(out *OwnedInfIntRational, e *error) {
	s.endLexicalError(e)
	s.finishLexicalInfInt(out, e)
}
func (s *RationalService) endLexicalEps(out *OwnedInfEpsInfRational, e *error) {
	s.endLexicalError(e)
	s.finishLexicalEps(out, e)
}
func (s *RationalService) finishLexicalError(e *error) {
	if s.lexical.pending() && *e != nil && s.lexical.originalCause == nil && !errors.Is(*e, s.lexical.cause) {
		s.lexical.originalCause = *e
	}
	if s.lexical.pending() && *e == nil {
		*e = s.lexical.cause
	}
}
func (s *RationalService) finishLexicalRational(out *OwnedRational, e *error) {
	if s.lexical.pending() {
		if out.ownerID != 0 {
			s.destroyLexicalRational(&out.raw)
			*out = OwnedRational{}
		}
		s.finishLexicalError(e)
	}
}
func (s *RationalService) finishLexicalInf(out *OwnedInfRational, e *error) {
	if s.lexical.pending() {
		if out.ownerID != 0 {
			s.infDestroy(out)
		}
		s.finishLexicalError(e)
	}
}
func (s *RationalService) finishLexicalInfInt(out *OwnedInfIntRational, e *error) {
	if s.lexical.pending() {
		if out.ownerID != 0 {
			s.destroyLexicalRational(&out.first)
			*out = OwnedInfIntRational{}
		}
		s.finishLexicalError(e)
	}
}
func (s *RationalService) finishLexicalEps(out *OwnedInfEpsInfRational, e *error) {
	if s.lexical.pending() {
		if out.ownerID != 0 {
			s.infEpsInfDestroy(out)
		}
		s.finishLexicalError(e)
	}
}
func (m *RationalManager) CleanupLexical() error {
	if e := m.checkOwnerReady(); e != nil {
		return e
	}
	return m.lexical.cleanup(m.z)
}
func (m *RationalManager) LexicalState() (RationalLexicalState, error) {
	if m == nil || m.self != m {
		return RationalLexicalState{}, arena.ErrInvalidSlot
	}
	return m.lexical.state(), nil
}

// Cross-owner Rational.CopyFrom has exactly one staged Rational local.
type rationalManagerLexicalScope struct {
	roots                    [1]Rational
	abort                    [1]bool
	count, cursor, component uint8
	cause, originalCause     error
}

func (l *rationalManagerLexicalScope) pending() bool { return l.cursor < l.count }
func (l *rationalManagerLexicalScope) retain(a *Rational, abort bool) {
	if a.num.cell == 0 && a.den.cell == 0 {
		*a = Rational{}
		return
	}
	// Every caller is a fixed source expression graph bounded above. Never drop
	// ownership on a violated bound: callers stop before producing another local.
	if l.count == uint8(len(l.roots)) {
		panic("mpnum: audited lexical root bound violated")
	}
	l.roots[l.count] = *a
	l.abort[l.count] = abort
	l.count++
	*a = Rational{}
}
func (l *rationalManagerLexicalScope) state() RationalLexicalState {
	v := RationalLexicalState{Pending: l.pending(), Cause: l.cause, OriginalCause: l.originalCause}
	for i := l.cursor; i < l.count; i++ {
		v.Roots++
		if l.abort[i] {
			v.Aborts++
		} else {
			v.Completed++
		}
	}
	return v
}
func (l *rationalManagerLexicalScope) dispose(z *IntegerManager, a *Rational, abort bool) error {
	if l.pending() {
		l.retain(a, abort)
		return l.cause
	}
	var component uint8
	if e := lexicalDelete(z, a, abort, &component); e != nil {
		l.cause = e
		l.retain(a, abort)
		l.component = component
		return e
	}
	return nil
}
func (l *rationalManagerLexicalScope) cleanup(z *IntegerManager) error {
	for l.pending() {
		// This public recovery is owning Go cleanup; it may abort a source-invalid
		// failed-copy pointer. Integer.Del resumes its persisted physical release.
		if e := lexicalDelete(z, &l.roots[l.cursor], true, &l.component); e != nil {
			l.cause = e
			return e
		}
		l.cursor++
		l.component = 0
	}
	*l = rationalManagerLexicalScope{}
	return nil
}

func (m *RationalManager) finishLexicalCopy(a *Rational, e *error) {
	if next := m.lexical.dispose(m.z, a, false); next != nil {
		if *e == nil {
			*e = next
		} else {
			m.lexical.originalCause = *e
		}
	}
}

// Public Del retains the caller's scalar token on refusal. Each callback is
// reached in source order; no unaccessed later component is prevalidated.
func callerDeleteRational(z *IntegerManager, a *Rational) error {
	if e := z.Del(&a.num); e != nil {
		return e
	}
	return z.Del(&a.den)
}
