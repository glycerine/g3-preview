// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
// Source mpz.cpp1412 and scoped_numeral.h31-33.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

var ErrSMT2DisplayPending = errors.New("mpnum: scoped SMT2 display cleanup is pending")

// A completed DEFAULT scoped numeral precedes its assignment. Its actual
// owning record and retained causes are included in the IntegerManager charge.
type smt2DisplayScope struct {
	value                   Int
	active, guard, assigned bool
	original, cleanup       error
}

type SMT2DisplayState struct {
	Pending, Guard              bool
	OriginalCause, CleanupCause error
}

func (m *IntegerManager) SMT2DisplayState() SMT2DisplayState {
	if m == nil || m.self != m {
		return SMT2DisplayState{CleanupCause: memory.ErrRuntimeOwner}
	}
	s := &m.smt2Display
	return SMT2DisplayState{s.active, s.guard, s.original, s.cleanup}
}

func (m *IntegerManager) beginSMT2Display() error {
	if m.smt2Display.active {
		return ErrSMT2DisplayPending
	}
	m.smt2Display = smt2DisplayScope{active: true}
	return nil
}

// Fresh source target.m_ptr stays null if allocate's RHS throws. The Go
// failedRef is diagnostic adapter metadata, not a dangling source pointer.
func (m *IntegerManager) copySMT2Temporary(a *Int) error {
	e := m.Copy(&m.smt2Display.value, a)
	m.smt2Display.assigned = e == nil
	return e
}

func (m *IntegerManager) finishSMT2Display(length *uint32, result *error) {
	s := &m.smt2Display
	s.original = *result
	n := m.natural
	// A stopped inner completed destructor prevents the older outer destructor.
	// Failed-constructor allocations alone are independent Go abort obligations.
	var cleanup error
	if n.formatting.completed != 0 || n.charFormatting.completed {
		cleanup = n.formatting.cleanupCause
		if cleanup == nil {
			cleanup = n.charFormatting.cleanupCause
		}
		if cleanup == nil {
			cleanup = n.formatting.originalCause
		}
		if cleanup == nil {
			cleanup = ErrSMT2DisplayPending
		}
	} else if !s.assigned && s.value.cell == m.failedRef {
		cleanup = m.Del(&s.value) // source DEFAULT destructor sees null; abort Go token only
	} else {
		s.guard, cleanup = sourceNumeralDel(m, &s.value)
	}
	if cleanup != nil {
		s.cleanup = cleanup
		*length = 0
		if *result == nil {
			*result = cleanup
		} else {
			*result = errors.Join(*result, cleanup)
		}
		return
	}
	*s = smt2DisplayScope{}
}

// CleanupSMT2Display resumes completed inner destruction, then the outer
// numeral. A guarded invalid source pointer is handled only as owning Go abort;
// no source destructor event, counter rollback or pointer repair is invented.
func (m *IntegerManager) CleanupSMT2Display() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	s := &m.smt2Display
	if !s.active {
		return nil
	}
	n := m.natural
	if n.formatting.completed != 0 {
		if e := n.unwindFormattingLocals(); e != nil {
			s.cleanup = e
			return e
		}
		n.formatting.active = false
		n.formatting.pending = true
	}
	if e := n.destroyIntegerCharacters(); e != nil {
		s.cleanup = e
		return e
	}
	var e error
	if s.guard {
		e = m.Del(&s.value)
	} else {
		s.guard, e = sourceNumeralDel(m, &s.value)
	}
	if e != nil {
		s.cleanup = e
		return e
	}
	*s = smt2DisplayScope{}
	return nil
}
