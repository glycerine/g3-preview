// Source: src/util/rational.cpp:64-83; src/util/rational.h:67.
// Leonardo de Moura, 2006-09-18. Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// ErrRationalServiceNoManager marks the reached source m() access while its
// global pointer is null. It is a guarded undefined source-lifetime domain.
var ErrRationalServiceNoManager = errors.New("mpnum: rational source manager is null")

// PrepareRationalService owns only concrete cold context and source static
// defaults. It does not complete or publish the source manager constructor.
func PrepareRationalService(c RationalServiceConfig) (*RationalService, error) {
	if c.Integers.Cells.Budget != c.Integers.Words.Descriptors.Budget {
		return nil, arena.ErrConfig
	}
	id, e := nextRationalServiceID()
	if e != nil {
		return nil, e
	}
	b := c.Integers.Cells.Budget
	header := uint64(reflect.TypeOf(RationalService{}).Size())
	if e = b.Reserve(header); e != nil {
		return nil, e
	}
	s := &RationalService{ownerID: id, zero: NewRational(), one: NewRational(), minusOne: NewRational(), headerBytes: header, initializationBudget: b, initializationConfig: c}
	s.self = s
	s.infZero, s.infOne, s.infMinusOne = s.infEmpty(), s.infEmpty(), s.infEmpty()
	s.infIntZero, s.infIntOne, s.infIntMinusOne = s.newInfInt(), s.newInfInt(), s.newInfInt()
	return s, nil
}

// InitializeSource preserves rational.cpp's pointer-present shortcut. A failed
// completed-manager body is retained, never automatically aborted or retried.
// Any actual uncompleted manager constructor receipt is a distinct Go owner.
func (s *RationalService) InitializeSource() error {
	if s == nil || s.self != s {
		return ErrRationalServiceCopy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.closing || s.sourceFinalizing {
		return arena.ErrClosed
	}
	if s.manager != nil {
		return nil
	}
	// Go pending-own domain: source rational::initialize would attempt another
	// null-pointer constructor; preserve the real retained failed child instead.
	// Generated memory::initialize also has its distinct failed-first wait domain.
	if s.initializationFailure != nil {
		return s.initializationCause
	}
	m, e := NewRationalManager(RationalConfig{Integers: s.initializationConfig.Integers, Synchronized: true})
	if e != nil {
		s.initializationCause = e
		var retained *IntegerManager
		if errors.As(e, &retained) {
			s.initializationFailure = retained
		}
		return e
	}
	s.manager = m // actual source pointer publication precedes every constant set
	s.initializationStage = 1
	e = s.initializeConstantsSource(m)
	s.initializationCause = e
	return e
}

func (s *RationalService) initializeConstantsSource(m *RationalManager) error {
	if e := m.SetInt64(&s.zero, 0); e != nil {
		return e
	}
	s.initializationStage = 2
	if e := m.SetInt64(&s.one, 1); e != nil {
		return e
	}
	s.initializationStage = 3
	if e := m.SetInt64(&s.minusOne, -1); e != nil {
		return e
	}
	s.initializationStage = 4
	if e := s.initInfRationalConstants(); e != nil {
		return e
	}
	s.initializationStage = 5
	if e := s.initInfIntRationalConstants(); e != nil {
		return e
	}
	s.initializationStage = 6
	return nil
}
