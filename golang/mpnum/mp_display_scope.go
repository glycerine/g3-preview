// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: mpq.cpp156-184 raw decimal-display locals.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
)

var ErrRawDisplayPending = errors.New("mpnum: raw display local cleanup is pending")

// These are the actual source locals. Their concrete storage is charged with
// the IntegerManager header. Raw mpz locals have no C++ numeral destructor:
// a failed body retains its cells until explicitly requested owning cleanup.
type rawDisplayScope struct {
	roots             [3]Int
	order             [3]uint8
	count, cursor     uint8
	active            bool
	original, cleanup error
}

func (m *IntegerManager) beginRawDisplay(count uint8, order [3]uint8) error {
	if m.rawDisplay.active {
		return ErrRawDisplayPending
	}
	m.rawDisplay = rawDisplayScope{count: count, order: order, active: true}
	return nil
}

// Only a successful body reaches the explicit source end label. Errors during
// its Del sequence remain observable, with the exact release cursor retained.
func (m *IntegerManager) finishRawDisplay(err *error, length *uint32) {
	s := &m.rawDisplay
	if *err == nil {
		for s.cursor < s.count {
			if _, e := sourceNumeralDel(m, &s.roots[s.order[s.cursor]]); e != nil {
				s.cleanup = e
				*err = e
				*length = 0
				return
			}
			s.cursor++
		}
		*s = rawDisplayScope{}
		return
	}
	s.original = *err
	for i := range s.roots {
		if s.roots[i].cell != 0 {
			return
		}
	}
	*s = rawDisplayScope{}
}

// CleanupRawDisplay aborts actual retained raw locals. It is owning Go cleanup,
// rather than a claim that the failed C++ function executed its end label.
func (m *IntegerManager) CleanupRawDisplay() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	s := &m.rawDisplay
	if !s.active {
		return nil
	}
	for i := range s.roots {
		p := &s.roots[i]
		if p.cell == 0 {
			continue
		}
		var e error
		if m.sourceFinalized || m.constructorUnwound {
			if p.cell != m.failedRef {
				e = m.freeAfterSourceFinalize(p.cell)
			}
			if e == nil {
				p.cell = 0
			}
		} else {
			e = m.Del(p)
		}
		if e != nil {
			s.cleanup = e
			return e
		}
	}
	*s = rawDisplayScope{}
	return nil
}
