// Source: scripts/mk_genfile_common.py:895-939; src/util/gparams.cpp:598-605;
// src/util/params.cpp:325-339,639-648,902-928; src/smt/params/smt_params_helper.pyg:59-66.
// Source author: Leonardo de Moura (params: 2011-05-09; gparams: 2012-11-29).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// This concrete owner is embedded in the actual parameter facade header. No
// second allocation/charge is made for it. A frame represents only the reached
// lexical helper: borrowed local reference and one acquired module reference.
type smtModuleParamsOwner struct {
	self    *smtModuleParamsOwner
	context *ParamsContext
	globals *GlobalParamsService
	budget  *arena.Budget
	helpers *smtModuleParamsFrame
	serial  uint64
	running bool
}
type smtModuleParamsFrame struct {
	self                                     *smtModuleParamsFrame
	owner                                    *smtModuleParamsOwner
	next                                     *smtModuleParamsFrame
	local                                    *ParamsRef
	value                                    ParamsRef
	receipt                                  ParamsDecrementReceipt
	header, invocation                       uint64
	acquired, destructorStarted, sourceDone  bool
	bodyCause, destructorCause, cleanupCause error
}

func (m *smtModuleParamsOwner) bind(c *ParamsContext, g *GlobalParamsService) error {
	if m == nil || m.self != m || m.running || m.helpers != nil {
		return ErrParamsPrecondition
	}
	if c == nil || c.self != c || g == nil || g.self != g || g.context != c || g.config.Budget != m.budget {
		return ErrParamsOwner
	}
	// Borrow only configuration identities. Actual symbol/global/local lifetimes
	// are read by the reached generated constructor and getter engines.
	m.context = c
	m.globals = g
	return nil
}
func (m *smtModuleParamsOwner) beginHelper(p *ParamsRef, module string) (*smtModuleParamsFrame, error) {
	if m == nil || m.self != m || !m.running || m.helpers != nil {
		return nil, ErrParamsPrecondition
	}
	if m.serial == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	n := uint64(reflect.TypeOf(smtModuleParamsFrame{}).Size())
	if e := m.budget.Reserve(n); e != nil {
		return nil, e
	}
	m.serial++
	f := &smtModuleParamsFrame{owner: m, local: p, header: n, invocation: m.serial, next: m.helpers}
	f.self = f
	m.helpers = f
	var e error
	f.value, e = m.globals.SourceGetModuleName(module)
	if e == nil {
		f.acquired = true
	} // the actual returned reference is the sole authority.
	return f, e
}
func (m *smtModuleParamsOwner) executingFrame(f *smtModuleParamsFrame) error {
	if m == nil || m.self != m || !m.running || f == nil || f.self != f || f.owner != m || m.helpers != f || f.invocation == 0 || f.invocation > m.serial {
		return ErrParamsPrecondition
	}
	return nil
}
func (m *smtModuleParamsOwner) frameBool(f *smtModuleParamsFrame, key string, def bool) (bool, error) {
	if e := m.executingFrame(f); e != nil {
		return false, e
	}
	if !f.acquired || f.destructorStarted {
		return false, ErrParamsPrecondition
	}
	return m.context.GetBoolFallbackName(f.local, &key, &f.value, def)
}
func (m *smtModuleParamsOwner) frameUInt(f *smtModuleParamsFrame, key string, def uint32) (uint32, error) {
	if e := m.executingFrame(f); e != nil {
		return 0, e
	}
	if !f.acquired || f.destructorStarted {
		return 0, ErrParamsPrecondition
	}
	return m.context.GetUIntFallbackName(f.local, &key, &f.value, def)
}
func (m *smtModuleParamsOwner) frameDouble(f *smtModuleParamsFrame, key string, def float64) (float64, error) {
	if e := m.executingFrame(f); e != nil {
		return 0, e
	}
	if !f.acquired || f.destructorStarted {
		return 0, ErrParamsPrecondition
	}
	return m.context.GetDoubleFallbackName(f.local, &key, &f.value, def)
}
func (m *smtModuleParamsOwner) finishHelper(f *smtModuleParamsFrame, body error) error {
	if e := m.executingFrame(f); e != nil {
		return errors.Join(body, e)
	}
	if f.bodyCause == nil {
		f.bodyCause = body
	}
	if f.destructorCause != nil {
		return errors.Join(body, f.destructorCause)
	}
	if f.acquired && !f.sourceDone {
		f.destructorStarted = true
		if e := m.context.DelTracked(&f.value, &f.receipt); e != nil {
			f.destructorCause = e
			return errors.Join(body, e)
		}
		f.sourceDone = true
	}
	m.helpers = f.next
	m.budget.Release(f.header)
	f.header = 0
	return body
}
func (m *smtModuleParamsOwner) closeHelpers() error {
	if m == nil || m.self != m || m.running {
		return ErrParamsPrecondition
	}
	m.running = true
	defer func() { m.running = false }()
	for m.helpers != nil {
		f := m.helpers
		if e := m.executingFrame(f); e != nil {
			return e
		}
		if f.acquired && !f.sourceDone {
			var e error
			if f.destructorStarted {
				if f.receipt.invocation == 0 {
					return f.destructorCause
				}
				e = m.context.CompleteDecrement(&f.value, f.receipt)
			} else {
				f.destructorStarted = true
				e = m.context.DelTracked(&f.value, &f.receipt)
			}
			if e != nil {
				if f.cleanupCause == nil {
					f.cleanupCause = e
				}
				return e
			}
			f.sourceDone = true
		}
		m.helpers = f.next
		m.budget.Release(f.header)
		f.header = 0
	}
	return nil
}
