// Source: src/util/debug.cpp:28-68,73-117; src/util/debug.h:42-65;
// src/test/mem_initializer.cpp:13-20.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrDebugGlobalOwner = errors.New("g3: invalid default debug service owner")

type DebugGlobalServiceConfig struct {
	Runtime *Runtime
	Budget  *arena.Budget
	Region  *arena.BytesRegion
	Profile DebugActionProfile
}

// One explicit source-global namespace uses the actual process standard files
// and one assertion/tag Context. The service owns only the acquired Go helper
// headers. Region bytes and OS descriptors remain caller/process owned. Native
// iostream static initialization and locale are separate representation
// obligations; optional Runtime borrows bind actual lazy tag storage; this does not install package singleton state.
type DebugGlobalService struct {
	self    *DebugGlobalService
	budget  *arena.Budget
	context *DebugContext
	streams *MemoryStandardStreams
	input   *NumericCharInput
	actions *DebugActionContext
	header  uint64
	closed  bool
}

func NewDebugGlobalService(c DebugGlobalServiceConfig) (*DebugGlobalService, error) {
	if c.Budget == nil || c.Region == nil || c.Profile > DebugNoDebugger || c.Runtime != nil && c.Runtime.Budget() != c.Budget {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(DebugGlobalService{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	s := &DebugGlobalService{budget: c.Budget, header: n}
	s.self = s
	var e error
	s.streams, e = NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: c.Budget})
	if e == nil {
		s.input, e = NewDebuggerCharInput(s.streams, nil)
	}
	if e == nil {
		s.actions, e = NewDebugActionContext(DebugActionConfig{Budget: c.Budget, Streams: s.streams, Input: s.input, Profile: c.Profile})
	}
	if e == nil {
		s.context, e = NewDebugContext(c.Region, c.Budget)
	}
	if e == nil && c.Runtime != nil {
		e = s.context.BindTagRuntime(c.Runtime)
	}
	if e == nil {
		e = s.context.BindAssertionStreams(s.streams)
	}
	if e == nil {
		e = s.context.BindAssertionActions(s.actions)
	}
	if e != nil {
		if cleanup := s.Close(); cleanup != nil {
			return s, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return s, nil
}
func (s *DebugGlobalService) open() error {
	if s == nil || s.self != s {
		return ErrDebugGlobalOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}

// Context returns the exact borrowed owner supplied as MemoryProcess.Debug.
// No assertion scalar or tag pointer is duplicated in the enclosing service.
func (s *DebugGlobalService) Context() (*DebugContext, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	return s.context, nil
}
func (s *DebugGlobalService) FinalizeDebug() error {
	if e := s.open(); e != nil {
		return e
	}
	return s.context.FinalizeDebug()
}

// Owning Close follows actual Go preparation in reverse. Generated source
// finalize_debug destroys only the lazy table, never these process IO borrows.
func (s *DebugGlobalService) Close() error {
	if s == nil || s.self != s {
		return ErrDebugGlobalOwner
	}
	if s.closed {
		return nil
	}
	if s.context != nil {
		if e := s.context.Close(); e != nil {
			return e
		}
	}
	if s.actions != nil {
		if e := s.actions.Close(); e != nil {
			return e
		}
	}
	if s.input != nil {
		if e := s.input.Close(); e != nil {
			return e
		}
	}
	if s.streams != nil {
		if e := s.streams.Close(); e != nil {
			return e
		}
	}
	s.budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}
